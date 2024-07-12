package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/rpc"
)

type RPCReq struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      json.RawMessage `json:"id"`
}

type RPCErr struct {
	Code          int    `json:"code"`
	Message       string `json:"message"`
	Data          string `json:"data,omitempty"`
	HTTPErrorCode int    `json:"-"`
}

func (r *RPCErr) Error() string {
	return r.Message
}

type RPCResJSON struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCErr         `json:"error,omitempty"`
	ID      json.RawMessage `json:"id"`
}

type RPCContext struct {
	IsSliceReq      bool
	IsSliceRes      bool
	IsGzip          bool
	HTTPResponse    *http.Response
	RPCReqs         []*RPCReq
	RPCRess         []*RPCResJSON
	RPCRessAttested []*RPCResJSONAttested
	BlockMap        map[string][]*rpc.BlockNumberOrHash
	ChangedMethods  map[string]string
	IDsHolder       map[string]string
}

func main() {
	// Start the server on port 8080
	http.HandleFunc("/", handler)
	log.Println("Starting server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (c *RPCContext) parseRPCReq(w http.ResponseWriter, r *http.Request) error {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return errors.New("Failed to read request body")
	}
	defer r.Body.Close()

	// Parse the request body
	var rpcReq *RPCReq
	var rpcReqs []*RPCReq

	// Check if the body is a single RPCReq or a slice of RPCReq
	err = json.Unmarshal(body, &rpcReq)
	if err != nil {
		if err := json.Unmarshal(body, &rpcReqs); err != nil {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return errors.New("Invalid request format")
		}
		c.IsSliceReq = true
		c.RPCReqs = rpcReqs
	} else {
		c.RPCReqs = []*RPCReq{rpcReq}
	}

	return nil
}

func (c *RPCContext) modifyReq(w http.ResponseWriter) error {
	blockMap, err := getBlockNumberMap(c.RPCReqs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	c.BlockMap = blockMap
	c.ChangedMethods = changeBlockNumberMethods(c.RPCReqs)

	c.RPCReqs, c.IDsHolder, err = addBlockNumberMethodsIfNeeded(c.RPCReqs, blockMap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	return nil
}

func (c *RPCContext) doRPCCall(w http.ResponseWriter, r *http.Request) error {
	// Marshal the modified request body
	var modifiedBody []byte
	var err error
	if c.IsSliceReq {
		modifiedBody, err = json.Marshal(c.RPCReqs)
	} else {
		modifiedBody, err = json.Marshal(c.RPCReqs[0])
	}
	if err != nil {
		http.Error(w, "Failed to marshal modified request", http.StatusInternalServerError)
		return errors.New("Failed to marshal modified request")
	}

	// Create a new request to forward to the second server
	req, err := http.NewRequest("POST", "http://localhost:8551", bytes.NewBuffer(modifiedBody))
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

	c.HTTPResponse = resp

	// Check if the response is gzipped and decompress if necessary
	var respBody []byte
	if resp.Header.Get("Content-Encoding") == "gzip" {
		c.IsGzip = true
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

		if c.IsGzip {
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
	var rpcRes *RPCResJSON
	var rpcRess []*RPCResJSON

	// Check if the body is a single rpcResJSON or a slice of rpcResJSON
	err = json.Unmarshal(respBody, &rpcRes)
	if err != nil {
		if err := json.Unmarshal(respBody, &rpcRess); err != nil {
			http.Error(w, "Invalid response format", http.StatusBadRequest)
			return errors.New("Invalid response format")
		}
		c.IsSliceRes = true
		c.RPCRess = rpcRess
	} else {
		c.RPCRess = []*RPCResJSON{rpcRes}
	}

	return nil
}

func (c *RPCContext) modifyRes(w http.ResponseWriter) error {
	var err error
	c.RPCRess, err = changeBlockNumberResponses(c.RPCRess, c.ChangedMethods, c.IDsHolder, c.BlockMap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	signer, err := GetSigningKeyFromKeyFile(".mock_key.pem")
	if err != nil {
		panic(err)
	}

	c.RPCRessAttested, err = AttestRess(c.RPCRess, "mock_identity", signer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	return nil
}

func (c *RPCContext) returnRes(w http.ResponseWriter) error {
	// Marshal the modified response body
	var modifiedRespBody []byte
	var err error
	if c.IsSliceRes {
		modifiedRespBody, err = json.Marshal(c.RPCRessAttested)
	} else {
		modifiedRespBody, err = json.Marshal(c.RPCRessAttested[0])
	}
	if err != nil {
		http.Error(w, "Failed to marshal modified response", http.StatusInternalServerError)
		return errors.New("Failed to marshal modified response")
	}

	// Copy the headers from the response
	for k, v := range c.HTTPResponse.Header {
		w.Header()[k] = v
	}

	// Send the modified response back to the original client
	w.Header().Set("Content-Type", "application/json")
	if c.IsGzip {
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

func handler(w http.ResponseWriter, r *http.Request) {
	context := &RPCContext{}

	err := context.parseRPCReq(w, r)
	if err != nil {
		return
	}

	err = context.modifyReq(w)
	if err != nil {
		return
	}

	err = context.doRPCCall(w, r)
	if err != nil {
		return
	}

	err = context.modifyRes(w)
	if err != nil {
		return
	}

	err = context.returnRes(w)
	if err != nil {
		return
	}
}
