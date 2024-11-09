package customrpcmethods

import (
	"crypto/rand"
	"math/big"

	"github.com/stateless-solutions/compatibility-layer/models"
)

const JSONRPCErrorInternal = -32000

var (
	ErrParseErr = &models.RPCErr{
		Code:          -32700,
		Message:       "parse error",
		HTTPErrorCode: 400,
	}

	ErrInternal = &models.RPCErr{
		Code:          JSONRPCErrorInternal,
		Message:       "internal error",
		HTTPErrorCode: 500,
	}
)

func generateRandomNumberStringWithRetries(rpcReqs []*models.RPCReq) (string, error) {
	retries := 0
	maxRetries := 5
	id := ""
	var err error

	for retries < maxRetries {
		id, err = generateRandomNumberString(12)
		if err != nil {
			return "", ErrInternal
		}

		// Check if the generated ID is repeated in the slice
		if !isIDRepeated(id, rpcReqs) {
			break
		}

		retries++
	}

	if retries == maxRetries {
		return "", ErrInternal
	}

	return id, nil
}

func generateRandomNumberString(n int) (string, error) {
	// The maximum value for a random number with n digits
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)

	// Generate a random number
	randomNumber, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	return randomNumber.String(), nil
}

func isIDRepeated(id string, rpcReqs []*models.RPCReq) bool {
	for _, rpcReq := range rpcReqs {
		if string(rpcReq.ID) == id {
			return true
		}
	}
	return false
}
