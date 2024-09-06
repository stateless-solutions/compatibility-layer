package attestation

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"

	"github.com/stateless-solutions/stateless-compatibility-layer/models"
	"golang.org/x/crypto/ssh"
)

func newErrorResponse(err *models.RPCErr, id json.RawMessage) models.RPCResJSONAttested {
	return models.RPCResJSONAttested{
		JSONRPC: "2.0",
		Error:   err,
	}
}

func AttestableError(jsonErr *models.RPCErr) ([]byte, error) {
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

func Attest(data []byte, identity string, signer ssh.Signer, full bool) (models.Attestation, error) {
	msgFixed := sha256.Sum256(data)
	msg := msgFixed[:]
	sig, err := signer.Sign(rand.Reader, msg)
	if err != nil {
		return models.Attestation{}, err
	}
	var attestation models.Attestation
	if full {
		attestation = models.Attestation{
			SignatureFormat: sig.Format,
			MsgHash:         hex.EncodeToString(msg),
			HashAlgo:        "sha256",
			Identiy:         identity,
			Signature:       hex.EncodeToString(sig.Blob),
		}
	} else {
		attestation = models.Attestation{
			MsgHash:   hex.EncodeToString(msg),
			Signature: hex.EncodeToString(sig.Blob),
		}

	}
	return attestation, nil
}

func Attestor(input *models.RPCResJSON, identity string, signer ssh.Signer, full bool) (*models.RPCResJSONAttested, error) {
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
	attested := &models.RPCResJSONAttested{
		Result:      input.Result,
		Error:       input.Error,
		JSONRPC:     input.JSONRPC,
		ID:          input.ID,
		Attestation: &attestation,
	}
	return attested, nil
}

func AttestRess(ress []*models.RPCResJSON, identity string, signer ssh.Signer) ([]*models.RPCResJSONAttested, error) {
	var attestedRess []*models.RPCResJSONAttested
	for i, result := range ress {
		attested, err := Attestor(result, identity, signer, i == 0)
		if err != nil {
			return nil, err
		}
		attestedRess = append(attestedRess, attested)
	}

	return attestedRess, nil
}
