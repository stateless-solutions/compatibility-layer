package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"

	"golang.org/x/crypto/ssh"
)

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

func newErrorResponse(err *RPCErr, id json.RawMessage) RPCResJSONAttested {
	return RPCResJSONAttested{
		JSONRPC: "2.0",
		Error:   err,
	}
}

func AttestableError(jsonErr *RPCErr) ([]byte, error) {
	return json.Marshal(jsonErr)
}

func AttestableJSON(result interface{}) ([]byte, error) {
	return json.Marshal(result)
}

func GetSigningKeyFromKeyFile(keyfile string) (ssh.Signer, error) {
	key, err := os.ReadFile(keyfile)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	return signer, nil
}

func GetSigningKeyFromKeyFileWithPassphrase(keyfile string, password string) (ssh.Signer, error) {
	key, err := os.ReadFile(keyfile)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte(password))
	if err != nil {
		return nil, err
	}
	return signer, nil

}

func Attest(data []byte, identity string, signer ssh.Signer, full bool) (Attestation, error) {
	msgFixed := sha256.Sum256(data)
	msg := msgFixed[:]
	sig, err := signer.Sign(rand.Reader, msg)
	if err != nil {
		return Attestation{}, err
	}
	var attestation Attestation
	if full {
		attestation = Attestation{
			SignatureFormat: sig.Format,
			MsgHash:         hex.EncodeToString(msg),
			HashAlgo:        "sha256",
			Identiy:         identity,
			Signature:       hex.EncodeToString(sig.Blob),
		}
	} else {
		attestation = Attestation{
			MsgHash:   hex.EncodeToString(msg),
			Signature: hex.EncodeToString(sig.Blob),
		}

	}
	return attestation, nil
}

func Attestor(input *RPCResJSON, identity string, signer ssh.Signer, full bool) (*RPCResJSONAttested, error) {
	var attestable []byte
	var err error
	if input.Result == nil {
		attestable, err = AttestableError(input.Error)
		if err != nil {
			return nil, err
		}
	} else {
		attestable, err = AttestableJSON(input.Result)
		if err != nil {
			return nil, err
		}
	}
	attestation, err := Attest(attestable, identity, signer, full)
	if err != nil {
		return nil, err
	}
	attested := &RPCResJSONAttested{
		Result:      input.Result,
		Error:       input.Error,
		JSONRPC:     input.JSONRPC,
		ID:          input.ID,
		Attestation: &attestation,
	}
	return attested, nil
}

func AttestRess(ress []*RPCResJSON, identity string, signer ssh.Signer) ([]*RPCResJSONAttested, error) {
	var attestedRess []*RPCResJSONAttested
	for i, result := range ress {
		attested, err := Attestor(result, identity, signer, i == 0)
		if err != nil {
			return nil, err
		}
		attestedRess = append(attestedRess, attested)
	}

	return attestedRess, nil
}
