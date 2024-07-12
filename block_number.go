package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/rpc"
)

type callResultAndBlockNumber struct {
	Result      interface{} `json:"result"`
	BlockNumber string      `json:"blockNumber"`
}

type balanceAndBlockNumber struct {
	Balance     interface{} `json:"balance"`
	BlockNumber string      `json:"blockNumber"`
}

type storageAndBlockNumber struct {
	Storage     interface{} `json:"storage"`
	BlockNumber string      `json:"blockNumber"`
}

type transactionCountAndBlockNumber struct {
	TransactionCount interface{} `json:"transactionCount"`
	BlockNumber      string      `json:"blockNumber"`
}

type codeAndBlockNumber struct {
	Code        interface{} `json:"code"`
	BlockNumber string      `json:"blockNumber"`
}

type blockTransactionCountAndBlockNumber struct {
	TransactionCount interface{} `json:"transactionCount"`
	BlockNumber      string      `json:"blockNumber"`
}

type rawTransactionAndBlockNumber struct {
	RawTransaction interface{} `json:"rawTransaction"`
	BlockNumber    string      `json:"blockNumber"`
}

type uncleCountAndBlockNumber struct {
	UncleCount  interface{} `json:"uncleCount"`
	BlockNumber string      `json:"blockNumber"`
}

type logsAndBlockRange struct {
	Logs          interface{} `json:"logs"`
	StartingBlock string      `json:"startingBlock"`
	EndingBlock   string      `json:"endingBlock"`
}

type balanceValuesAndBlockNumber struct {
	Values      interface{} `json:"values"`
	BlockNumber string      `json:"blockNumber"`
}

var (
	blockNumberToRegular = map[string]string{
		"eth_callAndBlockNumber":                                   "eth_call",
		"eth_getBalanceAndBlockNumber":                             "eth_getBalance",
		"eth_getStorageAtAndBlockNumber":                           "eth_getStorageAt",
		"eth_getTransactionCountAndBlockNumber":                    "eth_getTransactionCount",
		"eth_getCodeAndBlockNumber":                                "eth_getCode",
		"eth_getBlockTransactionCountAndBlockNumberByNumber":       "eth_getBlockTransactionCountByNumber",
		"eth_getRawTransactionAndBlockNumberByBlockNumberAndIndex": "eth_getRawTransactionByBlockNumberAndIndex",
		"eth_getUncleCountAndBlockNumberByBlockNumber":             "eth_getUncleCountByBlockNumber",
		"eth_getLogsAndBlockRange":                                 "eth_getLogs",
		"eth_getBalanceValuesAndBlockNumber":                       "eth_getBalanceValues",
	}

	methodToPos = map[string]int{
		"eth_callAndBlockNumber":                                   1,
		"eth_getBalanceAndBlockNumber":                             1,
		"eth_getStorageAtAndBlockNumber":                           2,
		"eth_getTransactionCountAndBlockNumber":                    1,
		"eth_getCodeAndBlockNumber":                                1,
		"eth_getBlockTransactionCountAndBlockNumberByNumber":       0,
		"eth_getRawTransactionAndBlockNumberByBlockNumberAndIndex": 0,
		"eth_getUncleCountAndBlockNumberByBlockNumber":             0,
		"eth_getLogsAndBlockRange":                                 0,
		"eth_getBalanceValuesAndBlockNumber":                       1,
	}

	JSONRPCErrorInternal = -32000

	ErrInternalBlockNumberMethodNotMap = &RPCErr{
		Code:          JSONRPCErrorInternal - 23,
		Message:       "block number response is not a map",
		HTTPErrorCode: 500,
	}

	ErrInternalBlockNumberMethodNotNumberEntry = &RPCErr{
		Code:          JSONRPCErrorInternal - 24,
		Message:       "block number response does not have number entry",
		HTTPErrorCode: 500,
	}

	ErrParseErr = &RPCErr{
		Code:          -32700,
		Message:       "parse error",
		HTTPErrorCode: 400,
	}

	ErrInternal = &RPCErr{
		Code:          JSONRPCErrorInternal,
		Message:       "internal error",
		HTTPErrorCode: 500,
	}
)

func remarshalBlockNumberOrHash(current interface{}) (*rpc.BlockNumberOrHash, error) {
	jv, err := json.Marshal(current)
	if err != nil {
		return nil, err
	}

	var bnh rpc.BlockNumberOrHash
	err = bnh.UnmarshalJSON(jv)
	if err != nil {
		return nil, err
	}

	return &bnh, nil
}

func remarshalTagMap(m map[string]interface{}, key string) (*rpc.BlockNumberOrHash, error) {
	if m[key] == nil || m[key] == "" {
		return nil, nil
	}

	current, ok := m[key].(string)
	if !ok {
		return nil, errors.New("expected string")
	}

	return remarshalBlockNumberOrHash(current)
}

func getBlockNumbers(req *RPCReq) ([]*rpc.BlockNumberOrHash, error) {
	_, ok := blockNumberToRegular[req.Method]
	if ok {
		pos := methodToPos[req.Method]

		if req.Method == "eth_getLogsAndBlockRange" {
			var p []map[string]interface{}
			err := json.Unmarshal(req.Params, &p)
			if err != nil {
				return nil, err
			}

			if len(p) <= pos {
				return nil, ErrParseErr
			}

			block, err := remarshalTagMap(p[pos], "blockHash")
			if err != nil {
				return nil, err
			}
			if block != nil && block.BlockHash != nil {
				return []*rpc.BlockNumberOrHash{block}, nil // if block hash is set fromBlock and toBlock are ignored
			}

			fromBlock, err := remarshalTagMap(p[pos], "fromBlock")
			if err != nil {
				return nil, err
			}
			if fromBlock == nil || fromBlock.BlockNumber == nil {
				b := rpc.BlockNumberOrHashWithNumber(rpc.EarliestBlockNumber)
				fromBlock = &b
			}
			toBlock, err := remarshalTagMap(p[pos], "toBlock")
			if err != nil {
				return nil, err
			}
			if toBlock == nil || toBlock.BlockNumber == nil {
				b := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
				toBlock = &b
			}
			return []*rpc.BlockNumberOrHash{fromBlock, toBlock}, nil // always keep this order
		}

		var p []interface{}
		err := json.Unmarshal(req.Params, &p)
		if err != nil {
			return nil, err
		}
		if len(p) <= pos {
			return nil, ErrParseErr
		}

		bnh, err := remarshalBlockNumberOrHash(p[pos])
		if err != nil {
			s, ok := p[pos].(string)
			if ok {
				block, err := remarshalBlockNumberOrHash(s)
				if err != nil {
					return nil, ErrParseErr
				}
				return []*rpc.BlockNumberOrHash{block}, nil
			} else {
				return nil, ErrParseErr
			}
		} else {
			return []*rpc.BlockNumberOrHash{bnh}, nil
		}
	}

	return nil, nil
}

func getBlockNumberMap(rpcReqs []*RPCReq) (map[string][]*rpc.BlockNumberOrHash, error) {
	bnMethodsBlockNumber := make(map[string][]*rpc.BlockNumberOrHash, len(rpcReqs))

	for _, req := range rpcReqs {
		bn, err := getBlockNumbers(req)
		if err != nil {
			return nil, err
		}
		if bn != nil {
			bnMethodsBlockNumber[string(req.ID)] = bn
		}
	}

	return bnMethodsBlockNumber, nil
}

func addBlockNumberMethodsIfNeeded(rpcReqs []*RPCReq, bnMethodsBlockNumber map[string][]*rpc.BlockNumberOrHash) ([]*RPCReq, map[string]string, error) {
	idsHolder := make(map[string]string, len(bnMethodsBlockNumber))

	for _, bns := range bnMethodsBlockNumber {
		for _, bn := range bns {
			if bn.BlockNumber != nil && bn.BlockHash != nil {
				return nil, nil, ErrParseErr
			}

			if bn.BlockHash != nil {
				bH := bn.BlockHash.String()
				_, ok := idsHolder[bH]
				if !ok {
					id, err := generateRandomNumberStringWithRetries(rpcReqs, 12)
					if err != nil {
						return nil, nil, err
					}
					idsHolder[bH] = id
					rpcReqs = append(rpcReqs, buildGetBlockByHashReq(bH, id))
				}
				continue
			}

			switch *bn.BlockNumber {
			case rpc.PendingBlockNumber:
				_, ok := idsHolder["pending"]
				if !ok {
					id, err := generateRandomNumberStringWithRetries(rpcReqs, 12)
					if err != nil {
						return nil, nil, err
					}
					idsHolder["pending"] = id
					rpcReqs = append(rpcReqs, buildGetBlockByNumberReq("pending", id))
				}
			case rpc.EarliestBlockNumber:
				_, ok := idsHolder["earliest"]
				if !ok {
					id, err := generateRandomNumberStringWithRetries(rpcReqs, 12)
					if err != nil {
						return nil, nil, err
					}
					idsHolder["earliest"] = id
					rpcReqs = append(rpcReqs, buildGetBlockByNumberReq("earliest", id))
				}
			case rpc.FinalizedBlockNumber:
				_, ok := idsHolder["finalized"]
				if !ok {
					id, err := generateRandomNumberStringWithRetries(rpcReqs, 12)
					if err != nil {
						return nil, nil, err
					}
					idsHolder["finalized"] = id
					rpcReqs = append(rpcReqs, buildGetBlockByNumberReq("finalized", id))
				}
			case rpc.SafeBlockNumber:
				_, ok := idsHolder["safe"]
				if !ok {
					id, err := generateRandomNumberStringWithRetries(rpcReqs, 12)
					if err != nil {
						return nil, nil, err
					}
					idsHolder["safe"] = id
					rpcReqs = append(rpcReqs, buildGetBlockByNumberReq("safe", id))
				}
			case rpc.LatestBlockNumber:
				_, ok := idsHolder["latest"]
				if !ok {
					id, err := generateRandomNumberStringWithRetries(rpcReqs, 12)
					if err != nil {
						return nil, nil, err
					}
					idsHolder["latest"] = id
					rpcReqs = append(rpcReqs, buildGetBlockByNumberReq("latest", id))
				}
			}
		}
	}

	return rpcReqs, idsHolder, nil
}

func buildGetBlockByHashReq(hash, id string) *RPCReq {
	return &RPCReq{
		JSONRPC: "2.0",
		Method:  "eth_getBlockByHash",
		ID:      json.RawMessage(id),
		Params:  json.RawMessage(fmt.Sprintf(`["%s",false]`, hash)),
	}
}

func buildGetBlockByNumberReq(tag, id string) *RPCReq {
	return &RPCReq{
		JSONRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		ID:      json.RawMessage(id),
		Params:  json.RawMessage(fmt.Sprintf(`["%s",false]`, tag)),
	}
}

func changeBlockNumberMethods(rpcReqs []*RPCReq) map[string]string {
	changedMethods := make(map[string]string, len(rpcReqs))

	for _, rpcReq := range rpcReqs {
		regMethod, ok := blockNumberToRegular[rpcReq.Method]
		if !ok {
			continue
		}

		changedMethods[string(rpcReq.ID)] = rpcReq.Method
		rpcReq.Method = regMethod
	}

	return changedMethods
}

func generateRandomNumberStringWithRetries(rpcReqs []*RPCReq, n int) (string, error) {
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

func isIDRepeated(id string, rpcReqs []*RPCReq) bool {
	for _, rpcReq := range rpcReqs {
		if string(rpcReq.ID) == id {
			return true
		}
	}
	return false
}

func getBlockHolder(responses []*RPCResJSON, idsHolder map[string]string) (map[string]string, []*RPCResJSON, error) {
	bnHolder := make(map[string]string, len(idsHolder))
	var responsesWithoutBN []*RPCResJSON

	for _, res := range responses {
		var bnMethod bool
		for content, id := range idsHolder {
			if string(res.ID) == id {
				resMap, ok := res.Result.(map[string]interface{})
				if !ok {
					return nil, nil, ErrInternalBlockNumberMethodNotMap
				}

				block, ok := resMap["number"].(string)
				if !ok {
					return nil, nil, ErrInternalBlockNumberMethodNotNumberEntry
				}

				bnHolder[content] = block
				bnMethod = true
			}
		}
		if !bnMethod {
			responsesWithoutBN = append(responsesWithoutBN, res)
		}
	}

	return bnHolder, responsesWithoutBN, nil
}

func changeBlockNumberResponses(responses []*RPCResJSON, changedMethods, idsHolder map[string]string, bnMethodsBlockNumber map[string][]*rpc.BlockNumberOrHash) ([]*RPCResJSON, error) {
	bnHolder, cleanRes, err := getBlockHolder(responses, idsHolder)
	if err != nil {
		return nil, err
	}

	for _, res := range cleanRes {
		originalMethod, ok := changedMethods[string(res.ID)]
		if !ok {
			continue
		}

		err := changeResultToBlockNumberStruct(res, bnHolder, bnMethodsBlockNumber, originalMethod)
		if err != nil {
			return nil, err
		}
	}

	return cleanRes, nil
}

func getBlockNumber(res *RPCResJSON, bnHolder map[string]string, bnMethodsBlockNumber map[string][]*rpc.BlockNumberOrHash) []string {
	bns := bnMethodsBlockNumber[string(res.ID)]

	var blocks []string
	for _, bn := range bns {
		if bns[0].BlockHash != nil {
			blocks = append(blocks, bnHolder[bn.BlockHash.String()])
			break // block hash can just be one per ID
		}
		bnString := bn.BlockNumber.String()
		tagBlock, ok := bnHolder[bnString]
		if ok {
			blocks = append(blocks, tagBlock)
			continue
		}
		blocks = append(blocks, bnString)
	}

	return blocks
}

func changeResultToBlockNumberStruct(res *RPCResJSON, bnHolder map[string]string, bnMethodsBlockNumber map[string][]*rpc.BlockNumberOrHash, originalMethod string) error {
	blockNumber := getBlockNumber(res, bnHolder, bnMethodsBlockNumber)

	switch originalMethod {
	case "eth_callAndBlockNumber":
		res.Result = callResultAndBlockNumber{
			Result:      res.Result,
			BlockNumber: blockNumber[0],
		}
	case "eth_getBalanceAndBlockNumber":
		res.Result = balanceAndBlockNumber{
			Balance:     res.Result,
			BlockNumber: blockNumber[0],
		}
	case "eth_getStorageAtAndBlockNumber":
		res.Result = storageAndBlockNumber{
			Storage:     res.Result,
			BlockNumber: blockNumber[0],
		}
	case "eth_getTransactionCountAndBlockNumber":
		res.Result = transactionCountAndBlockNumber{
			TransactionCount: res.Result,
			BlockNumber:      blockNumber[0],
		}
	case "eth_getCodeAndBlockNumber":
		res.Result = codeAndBlockNumber{
			Code:        res.Result,
			BlockNumber: blockNumber[0],
		}
	case "eth_getBlockTransactionCountAndBlockNumberByNumber":
		res.Result = blockTransactionCountAndBlockNumber{
			TransactionCount: res.Result,
			BlockNumber:      blockNumber[0],
		}
	case "eth_getRawTransactionAndBlockNumberByBlockNumberAndIndex":
		res.Result = rawTransactionAndBlockNumber{
			RawTransaction: res.Result,
			BlockNumber:    blockNumber[0],
		}
	case "eth_getUncleCountAndBlockNumberByBlockNumber":
		res.Result = uncleCountAndBlockNumber{
			UncleCount:  res.Result,
			BlockNumber: blockNumber[0],
		}
	case "eth_getLogsAndBlockRange":
		fromBlock := blockNumber[0]
		toBlock := blockNumber[0]
		if len(blockNumber) > 1 {
			toBlock = blockNumber[1]
		}
		res.Result = logsAndBlockRange{
			Logs:          res.Result,
			StartingBlock: fromBlock,
			EndingBlock:   toBlock,
		}
	case "eth_getBalanceValuesAndBlockNumber":
		res.Result = balanceValuesAndBlockNumber{
			Values:      res.Result,
			BlockNumber: blockNumber[0],
		}
	}

	return nil
}
