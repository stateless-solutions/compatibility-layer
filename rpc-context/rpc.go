package rpccontext

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/stateless-solutions/stateless-compatibility-layer/attestation"
	customrpcmethods "github.com/stateless-solutions/stateless-compatibility-layer/custom-rpc-methods"
	"github.com/stateless-solutions/stateless-compatibility-layer/models"
	"golang.org/x/crypto/ssh"
)

type RPCContext struct {
	Identity           string
	DefaultChainURL    string
	HTTPPort           string
	CustomMethodHolder *customrpcmethods.CustomMethodHolder
	UseAttestation     bool
	SigningKey         ssh.Signer
}

type reqHandler struct {
	ChainURL         string
	IsSlice          bool
	IsGzip           bool
	HTTPResponse     *http.Response
	RPCReqs          []*models.RPCReq
	RPCRess          []*models.RPCResJSON
	RPCRessAttested  []*models.RPCResJSONAttested
	CustomMethodsMap interface{} // this should always be of the type map[string][]T from custom-rpc-methods/custom_rpc_methods.go
	ChangedMethods   map[string]string
	IDsHolder        map[string]string
}

func (c *RPCContext) parseRPCReq(w http.ResponseWriter, r *http.Request, rh *reqHandler) error {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return fmt.Errorf("failed to read request body: %w", err)
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
			return fmt.Errorf("failed to unmarshal request: %w", err)
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
	customMethodsMap, err := c.CustomMethodHolder.GetCustomMethodsMap(rh.RPCReqs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	rh.CustomMethodsMap = customMethodsMap
	rh.ChangedMethods, err = c.CustomMethodHolder.ChangeCustomMethods(rh.RPCReqs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	rh.RPCReqs, rh.IDsHolder, err = c.CustomMethodHolder.AddGetterMethodsIfNeeded(rh.RPCReqs, customMethodsMap)
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
		return fmt.Errorf("failed to marshal modified request: %w", err)
	}

	// Create a new request to forward to the second server
	req, err := http.NewRequest("POST", rh.ChainURL, bytes.NewBuffer(modifiedBody))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return fmt.Errorf("failed to create request: %w", err)
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
		return fmt.Errorf("failed to forward request: %w", err)
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
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzr.Close()
		respBody, err = io.ReadAll(gzr)
		if err != nil {
			http.Error(w, "Failed to read gzipped response body", http.StatusInternalServerError)
			return fmt.Errorf("failed to read gzipped response body: %w", err)
		}
	} else {
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Failed to read response body", http.StatusInternalServerError)
			return fmt.Errorf("failed to read response body: %w", err)
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
				return fmt.Errorf("failed to compress response body: %w", err)
			}
			if err := gzw.Close(); err != nil {
				http.Error(w, "Failed to close gzip writer", http.StatusInternalServerError)
				return fmt.Errorf("failed to close gzip writer: %w", err)
			}
			w.Write(buf.Bytes())
		} else {
			w.Write(respBody)
		}

		return errors.New("Response was not 200 ok")
	}

	// Parse the response body
	var rpcRes *models.RPCResJSON
	var rpcRess []*models.RPCResJSON

	// Check if the body is a single rpcResJSON or a slice of rpcResJSON
	err = json.Unmarshal(respBody, &rpcRes)
	if err != nil {
		if err := json.Unmarshal(respBody, &rpcRess); err != nil {
			http.Error(w, "Invalid response format", http.StatusBadRequest)
			return fmt.Errorf("failed to unmarshal response body: %w", err)
		}
		rh.RPCRess = rpcRess
	} else {
		rh.RPCRess = []*models.RPCResJSON{rpcRes}
	}

	return nil
}

func (c *RPCContext) modifyRes(w http.ResponseWriter, rh *reqHandler) error {
	var err error
	rh.RPCRess, err = c.CustomMethodHolder.ChangeCustomMethodsResponses(rh.RPCRess, rh.ChangedMethods, rh.IDsHolder, rh.CustomMethodsMap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	if c.UseAttestation {
		rh.RPCRessAttested, err = attestation.AttestRess(rh.RPCRess, c.Identity, c.SigningKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}
	}

	return nil
}

func (c *RPCContext) marshalBody(rh *reqHandler) ([]byte, error) {
	var modifiedRespBody []byte
	var err error
	if c.UseAttestation {
		if rh.IsSlice {
			modifiedRespBody, err = json.Marshal(rh.RPCRessAttested)
		} else {
			modifiedRespBody, err = json.Marshal(rh.RPCRessAttested[0])
		}
	} else {
		if rh.IsSlice {
			modifiedRespBody, err = json.Marshal(rh.RPCRess)
		} else {
			modifiedRespBody, err = json.Marshal(rh.RPCRess[0])
		}
	}

	return modifiedRespBody, err
}

func (c *RPCContext) returnRes(w http.ResponseWriter, rh *reqHandler) error {
	// Marshal the modified response body
	modifiedRespBody, err := c.marshalBody(rh)
	if err != nil {
		http.Error(w, "Failed to marshal modified response", http.StatusInternalServerError)
		return fmt.Errorf("failed to marshal modified response: %w", err)
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
			return fmt.Errorf("failed to compress response body: %w", err)
		}
		if err := gzw.Close(); err != nil {
			http.Error(w, "Failed to close gzip writer", http.StatusInternalServerError)
			return fmt.Errorf("failed to close gzip writer: %w", err)
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(buf.Bytes())))
		w.Write(buf.Bytes())
	} else {
		w.Header().Set("Content-Length", strconv.Itoa(len(modifiedRespBody)))
		w.Write(modifiedRespBody)
	}

	return nil
}

func (c *RPCContext) newReqHandler(w http.ResponseWriter, r *http.Request) (*reqHandler, error) {
	rh := &reqHandler{
		ChainURL: c.DefaultChainURL,
	}

	headerChainURL := r.Header.Get("Stateless-Chain-URL")
	if headerChainURL != "" {
		// Validate the URL
		_, err := url.ParseRequestURI(headerChainURL)
		if err != nil {
			http.Error(w, "Invalid Chain URL", http.StatusBadRequest)
			return nil, errors.New("invalid chain URL")
		}
		rh.ChainURL = headerChainURL
	}

	if rh.ChainURL == "" {
		http.Error(w, "Chain URL is not set", http.StatusBadRequest)
		return nil, errors.New("chain URL is not set")
	}

	return rh, nil
}

func (c *RPCContext) EnableAttestation(keyFile, keyFilePassword, identity string) {
	if keyFile == "" {
		panic("KEY_FILE env must be set if USE_ATTESTATION is true")
	}
	if identity == "" {
		panic("IDENTITY env must be set if USE_ATTESTATION is true")
	}

	var signer ssh.Signer
	var err error
	if keyFilePassword != "" {
		signer, err = attestation.GetSigningKeyFromKeyFileWithPassphrase(keyFile, keyFilePassword)
		if err != nil {
			panic(err)
		}
	} else {
		signer, err = attestation.GetSigningKeyFromKeyFile(keyFile)
		if err != nil {
			panic(err)
		}
	}

	c.SigningKey = signer
	c.UseAttestation = true
}

func (c *RPCContext) Handler(w http.ResponseWriter, r *http.Request) {
	rh, err := c.newReqHandler(w, r)
	if err != nil {
		return
	}

	err = c.parseRPCReq(w, r, rh)
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
