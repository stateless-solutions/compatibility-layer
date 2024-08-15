package rpccontext

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stateless-solutions/stateless-compatibility-layer/attestation"
	blocknumber "github.com/stateless-solutions/stateless-compatibility-layer/block-number"
	"github.com/stateless-solutions/stateless-compatibility-layer/models"
	"golang.org/x/crypto/ssh"
)

type RPCContext struct {
	Identity        string
	ChainURL        string
	HTTPPort        string
	BlockNumberConv *blocknumber.BlockNumberConv
	SigningKey      ssh.Signer
}

type reqHandler struct {
	IsSlice         bool
	IsGzip          bool
	HTTPResponse    *http.Response
	RPCReqs         []*models.RPCReq
	RPCRess         []*models.RPCResJSON
	RPCRessAttested []*models.RPCResJSONAttested
	BlockMap        map[string][]*rpc.BlockNumberOrHash
	ChangedMethods  map[string]string
	IDsHolder       map[string]string
}

func (c *RPCContext) parseRPCReq(w http.ResponseWriter, r *http.Request, rh *reqHandler) error {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return errors.New("Failed to read request body")
	}
	defer r.Body.Close()

	// Parse the request body
	var rpcReq *models.RPCReq
	var rpcReqs []*models.RPCReq

	// Check if the body is a single RPCReq or a slice of RPCReq
	err = json.Unmarshal(body, &rpcReq)
	if err != nil {
		if err := json.Unmarshal(body, &rpcReqs); err != nil {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return errors.New("Invalid request format")
		}
		rh.IsSlice = true
		rh.RPCReqs = rpcReqs
	} else {
		rh.IsSlice = false
		rh.RPCReqs = []*models.RPCReq{rpcReq}
	}

	return nil
}

func (c *RPCContext) modifyReq(w http.ResponseWriter, rh *reqHandler) error {
	blockMap, err := c.BlockNumberConv.GetBlockNumberMap(rh.RPCReqs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	rh.BlockMap = blockMap
	rh.ChangedMethods = c.BlockNumberConv.ChangeBlockNumberMethods(rh.RPCReqs)

	rh.RPCReqs, rh.IDsHolder, err = blocknumber.AddBlockNumberMethodsIfNeeded(rh.RPCReqs, blockMap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	return nil
}

func (c *RPCContext) doRPCCall(w http.ResponseWriter, r *http.Request, rh *reqHandler) error {
	// Marshal the modified request body
	var modifiedBody []byte
	var err error
	modifiedBody, err = json.Marshal(rh.RPCReqs)
	if err != nil {
		http.Error(w, "Failed to marshal modified request", http.StatusInternalServerError)
		return errors.New("Failed to marshal modified request")
	}

	// Create a new request to forward to the second server
	req, err := http.NewRequest("POST", c.ChainURL, bytes.NewBuffer(modifiedBody))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return errors.New("Failed to create request")
	}

	// Copy the original headers
	for k, v := range r.Header {
		req.Header[k] = v
	}

	// Forward the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to forward request", http.StatusInternalServerError)
		return errors.New("Failed to forward request")
	}
	defer resp.Body.Close()

	rh.HTTPResponse = resp

	// Check if the response is gzipped and decompress if necessary
	var respBody []byte
	if resp.Header.Get("Content-Encoding") == "gzip" {
		rh.IsGzip = true
		gzr, err := gzip.NewReader(resp.Body)
		if err != nil {
			http.Error(w, "Failed to create gzip reader", http.StatusInternalServerError)
			return errors.New("Failed to create gzip reader")
		}
		defer gzr.Close()
		respBody, err = io.ReadAll(gzr)
		if err != nil {
			http.Error(w, "Failed to read gzipped response body", http.StatusInternalServerError)
			return errors.New("Failed to read gzipped response body")
		}
	} else {
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Failed to read response body", http.StatusInternalServerError)
			return errors.New("Failed to read response body")
		}
	}

	if resp.StatusCode != http.StatusOK {
		// Copy the headers from the response
		for k, v := range resp.Header {
			w.Header()[k] = v
		}

		// Send the modified response back to the original client
		w.Header().Set("Content-Type", "application/json")

		if rh.IsGzip {
			var buf bytes.Buffer
			gzw := gzip.NewWriter(&buf)
			if _, err := gzw.Write(respBody); err != nil {
				http.Error(w, "Failed to compress response body", http.StatusInternalServerError)
				return errors.New("Failed to compress response body")
			}
			if err := gzw.Close(); err != nil {
				http.Error(w, "Failed to close gzip writer", http.StatusInternalServerError)
				return errors.New("Failed to close gzip writer")
			}
			w.Write(buf.Bytes())
		} else {
			w.Write(respBody)
		}

		return errors.New("Rsponse was not 200ok")
	}

	// Parse the response body
	var rpcRes *models.RPCResJSON
	var rpcRess []*models.RPCResJSON

	// Check if the body is a single rpcResJSON or a slice of rpcResJSON
	err = json.Unmarshal(respBody, &rpcRes)
	if err != nil {
		if err := json.Unmarshal(respBody, &rpcRess); err != nil {
			http.Error(w, "Invalid response format", http.StatusBadRequest)
			return errors.New("Invalid response format")
		}
		rh.RPCRess = rpcRess
	} else {
		rh.RPCRess = []*models.RPCResJSON{rpcRes}
	}

	return nil
}

func (c *RPCContext) modifyRes(w http.ResponseWriter, rh *reqHandler) error {
	var err error
	rh.RPCRess, err = c.BlockNumberConv.ChangeBlockNumberResponses(rh.RPCRess, rh.ChangedMethods, rh.IDsHolder, rh.BlockMap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	rh.RPCRessAttested, err = attestation.AttestRess(rh.RPCRess, c.Identity, c.SigningKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	return nil
}

func (c *RPCContext) returnRes(w http.ResponseWriter, rh *reqHandler) error {
	// Marshal the modified response body
	var modifiedRespBody []byte
	var err error
	if rh.IsSlice {
		modifiedRespBody, err = json.Marshal(rh.RPCRessAttested)
	} else {
		modifiedRespBody, err = json.Marshal(rh.RPCRessAttested[0])
	}
	if err != nil {
		http.Error(w, "Failed to marshal modified response", http.StatusInternalServerError)
		return errors.New("Failed to marshal modified response")
	}

	// Copy the headers from the response
	for k, v := range rh.HTTPResponse.Header {
		w.Header()[k] = v
	}

	// Send the modified response back to the original client
	w.Header().Set("Content-Type", "application/json")
	if rh.IsGzip {
		// Compress the response
		var buf bytes.Buffer
		gzw := gzip.NewWriter(&buf)
		if _, err := gzw.Write(modifiedRespBody); err != nil {
			http.Error(w, "Failed to compress response body", http.StatusInternalServerError)
			return errors.New("Failed to compress response body")
		}
		if err := gzw.Close(); err != nil {
			http.Error(w, "Failed to close gzip writer", http.StatusInternalServerError)
			return errors.New("Failed to close gzip writer")
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(buf.Bytes())))
		w.Write(buf.Bytes())
	} else {
		w.Header().Set("Content-Length", strconv.Itoa(len(modifiedRespBody)))
		w.Write(modifiedRespBody)
	}

	return nil
}

func (c *RPCContext) Handler(w http.ResponseWriter, r *http.Request) {
	rh := &reqHandler{}

	err := c.parseRPCReq(w, r, rh)
	if err != nil {
		return
	}

	err = c.modifyReq(w, rh)
	if err != nil {
		return
	}

	err = c.doRPCCall(w, r, rh)
	if err != nil {
		return
	}

	err = c.modifyRes(w, rh)
	if err != nil {
		return
	}

	err = c.returnRes(w, rh)
	if err != nil {
		return
	}
}
