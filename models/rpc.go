package models

import (
	"encoding/json"
	"log/slog"
)

type RPCReq struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      json.RawMessage `json:"id"`
}

func (req RPCReq) LogAttrs() []slog.Attr {
	return []slog.Attr{
		slog.String("jsonrpc", req.JSONRPC),
		slog.String("method", req.Method),
		slog.String("params", string(req.Params)),
		slog.String("id", string(req.ID)),
	}
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

func (res RPCResJSON) LogAttrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("jsonrpc", res.JSONRPC),
		slog.Any("result", res.Result),
		slog.String("id", string(res.ID)),
	}

	if res.Error != nil {
		attrs = append(attrs, slog.String("error", res.Error.Error()))
	}

	return attrs
}

type Attestation struct {
	SignatureFormat string `json:"signatureFormat,omitempty"`
	HashAlgo        string `json:"hashAlgo,omitempty"`
	Identiy         string `json:"identity,omitempty"`
	MsgHash         string `json:"msg"`
	Signature       string `json:"signature"`
}

func (a Attestation) LogAttrs() []slog.Attr {
	return []slog.Attr{
		slog.String("signatureFormat", a.SignatureFormat),
		slog.String("hashAlgo", a.HashAlgo),
		slog.String("identity", a.Identiy),
		slog.String("msg", a.MsgHash),
		slog.String("signature", a.Signature),
	}
}

type RPCResJSONAttested struct {
	JSONRPC     string          `json:"jsonrpc,omitempty"`
	ID          json.RawMessage `json:"id,omitempty"`
	Error       *RPCErr         `json:"error,omitempty"`
	Result      interface{}     `json:"result,omitempty"`
	Attestation *Attestation    `json:"attestation,omitempty"`
}

func (res RPCResJSONAttested) LogAttrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("jsonrpc", res.JSONRPC),
		slog.Any("result", res.Result),
		slog.String("id", string(res.ID)),
	}

	if res.Error != nil {
		attrs = append(attrs, slog.String("error", res.Error.Error()))
	}

	if res.Attestation != nil {
		attrs = append(attrs, slog.Attr{
			Key:   "attestation",
			Value: slog.GroupValue(res.Attestation.LogAttrs()...)})
	}

	return attrs
}
