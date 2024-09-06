package models

import "encoding/json"

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

type Attestation struct {
	SignatureFormat string `json:"signatureFormat,omitempty"`
	HashAlgo        string `json:"hashAlgo,omitempty"`
	Identiy         string `json:"identity,omitempty"`
	MsgHash         string `json:"msg"`
	Signature       string `json:"signature"`
}

type RPCResJSONAttested struct {
	JSONRPC     string          `json:"jsonrpc,omitempty"`
	ID          json.RawMessage `json:"id,omitempty"`
	Error       *RPCErr         `json:"error,omitempty"`
	Result      interface{}     `json:"result,omitempty"`
	Attestation *Attestation    `json:"attestation,omitempty"`
}
