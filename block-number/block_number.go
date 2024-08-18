package blocknumber

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stateless-solutions/stateless-compatibility-layer/models"
)

type blockNumberResult struct {
	Data        interface{} `json:"data"`
	BlockNumber string      `json:"blockNumber"`
}

type blockRangeResult struct {
	Data          interface{} `json:"data"`
	StartingBlock string      `json:"startingBlock"`
	EndingBlock   string      `json:"endingBlock"`
}

type Method struct {
	OriginalMethod            string `json:"originalMethod"`
	BlockNumberMethod         string `json:"blockNumberMethod"`
	PositionsBlockNumberParam []int  `json:"positionsBlockNumberParam,omitempty"`
	CustomHandler             string `json:"customHandler,omitempty"`
	IsBlockRange              bool   `json:"isBlockRange"`
}

type MethodsConfig struct {
	Methods []Method `json:"methods"`
}

var (
	JSONRPCErrorInternal = -32000

	ErrInternalBlockNumberMethodNotMap = &models.RPCErr{
		Code:          JSONRPCErrorInternal - 23,
		Message:       "block number response is not a map",
		HTTPErrorCode: 500,
	}

	ErrInternalBlockNumberMethodNotNumberEntry = &models.RPCErr{
		Code:          JSONRPCErrorInternal - 24,
		Message:       "block number response does not have number entry",
		HTTPErrorCode: 500,
	}

	ErrInternalCustomHandlerNotFound = &models.RPCErr{
		Code:          JSONRPCErrorInternal - 25,
		Message:       "custom handler of function not found",
		HTTPErrorCode: 500,
	}

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

// this validates on start if all custom handlers have correct structure
func init() {
	expectedType := reflect.TypeOf(func(customHandlersHolder, *models.RPCReq) ([]*rpc.BlockNumberOrHash, error) { return nil, nil })
	holderType := reflect.TypeOf(customHandlersHolder{})

	for i := 0; i < holderType.NumMethod(); i++ {
		method := holderType.Method(i)

		if method.Type != expectedType {
			panic(fmt.Sprintf("method %s has an incorrect signature: expected %v, got %v", method.Name, expectedType, method.Type))
		}
	}
}

type BlockNumberConv struct {
	configfile                       string
	blockNumberToRegular             map[string]string
	blockNumberMethodToPos           map[string][]int
	blockNumberMethodToIsBlockRange  map[string]bool
	blockNumberMethodToCustomHandler map[string]string
}

func NewBlockNumberConv(configFiles string) *BlockNumberConv {
	bnc := &BlockNumberConv{
		configfile:                       configFiles,
		blockNumberToRegular:             map[string]string{},
		blockNumberMethodToPos:           map[string][]int{},
		blockNumberMethodToIsBlockRange:  map[string]bool{},
		blockNumberMethodToCustomHandler: map[string]string{},
	}

	files := strings.Split(configFiles, ",")
	for _, file := range files {
		byteValue, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}

		var config MethodsConfig
		err = json.Unmarshal(byteValue, &config)
		if err != nil {
			panic(err)
		}

		for _, method := range config.Methods {
			bnc.blockNumberToRegular[method.BlockNumberMethod] = method.OriginalMethod
			bnc.blockNumberMethodToPos[method.BlockNumberMethod] = method.PositionsBlockNumberParam
			bnc.blockNumberMethodToIsBlockRange[method.BlockNumberMethod] = method.IsBlockRange
			bnc.blockNumberMethodToCustomHandler[method.BlockNumberMethod] = method.CustomHandler
		}
	}

	return bnc
}

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

func (b *BlockNumberConv) getBlockNumbers(req *models.RPCReq) ([]*rpc.BlockNumberOrHash, error) {
	_, ok := b.blockNumberToRegular[req.Method]
	if ok {
		if b.blockNumberMethodToCustomHandler[req.Method] != "" {
			method := reflect.ValueOf(customHandlersHolder{}).MethodByName(b.blockNumberMethodToCustomHandler[req.Method])
			if !method.IsValid() {
				return nil, ErrInternalCustomHandlerNotFound
			}
			result := method.Call([]reflect.Value{
				reflect.ValueOf(req),
			})
			blockNumbers := result[0].Interface().([]*rpc.BlockNumberOrHash)
			err, _ := result[1].Interface().(error)

			return blockNumbers, err
		}

		var p []interface{}
		err := json.Unmarshal(req.Params, &p)
		if err != nil {
			return nil, err
		}

		var bns []*rpc.BlockNumberOrHash
		for _, pos := range b.blockNumberMethodToPos[req.Method] {
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
					bns = append(bns, block)
				} else {
					return nil, ErrParseErr
				}
			} else {
				bns = append(bns, bnh)
			}
		}

		return bns, nil
	}

	return nil, nil
}

func (b *BlockNumberConv) GetBlockNumberMap(rpcReqs []*models.RPCReq) (map[string][]*rpc.BlockNumberOrHash, error) {
	bnMethodsBlockNumber := make(map[string][]*rpc.BlockNumberOrHash, len(rpcReqs))

	for _, req := range rpcReqs {
		bn, err := b.getBlockNumbers(req)
		if err != nil {
			return nil, err
		}
		if bn != nil {
			bnMethodsBlockNumber[string(req.ID)] = bn
		}
	}

	return bnMethodsBlockNumber, nil
}

func AddBlockNumberMethodsIfNeeded(rpcReqs []*models.RPCReq, bnMethodsBlockNumber map[string][]*rpc.BlockNumberOrHash) ([]*models.RPCReq, map[string]string, error) {
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

func buildGetBlockByHashReq(hash, id string) *models.RPCReq {
	return &models.RPCReq{
		JSONRPC: "2.0",
		Method:  "eth_getBlockByHash",
		ID:      json.RawMessage(id),
		Params:  json.RawMessage(fmt.Sprintf(`["%s",false]`, hash)),
	}
}

func buildGetBlockByNumberReq(tag, id string) *models.RPCReq {
	return &models.RPCReq{
		JSONRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		ID:      json.RawMessage(id),
		Params:  json.RawMessage(fmt.Sprintf(`["%s",false]`, tag)),
	}
}

func (b *BlockNumberConv) ChangeBlockNumberMethods(rpcReqs []*models.RPCReq) map[string]string {
	changedMethods := make(map[string]string, len(rpcReqs))

	for _, rpcReq := range rpcReqs {
		regMethod, ok := b.blockNumberToRegular[rpcReq.Method]
		if !ok {
			continue
		}

		changedMethods[string(rpcReq.ID)] = rpcReq.Method
		rpcReq.Method = regMethod
	}

	return changedMethods
}

func generateRandomNumberStringWithRetries(rpcReqs []*models.RPCReq, n int) (string, error) {
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

func getBlockHolder(responses []*models.RPCResJSON, idsHolder map[string]string) (map[string]*models.RPCResJSON, []*models.RPCResJSON, error) {
	bnHolder := make(map[string]*models.RPCResJSON, len(idsHolder))
	var responsesWithoutBN []*models.RPCResJSON

	for _, res := range responses {
		var bnMethod bool
		for content, id := range idsHolder {
			if string(res.ID) == id {
				bnHolder[content] = res
				bnMethod = true
			}
		}
		if !bnMethod {
			responsesWithoutBN = append(responsesWithoutBN, res)
		}
	}

	return bnHolder, responsesWithoutBN, nil
}

func (b *BlockNumberConv) ChangeBlockNumberResponses(responses []*models.RPCResJSON, changedMethods, idsHolder map[string]string, bnMethodsBlockNumber map[string][]*rpc.BlockNumberOrHash) ([]*models.RPCResJSON, error) {
	bnHolder, cleanRes, err := getBlockHolder(responses, idsHolder)
	if err != nil {
		return nil, err
	}

	for _, res := range cleanRes {
		originalMethod, ok := changedMethods[string(res.ID)]
		if !ok {
			continue
		}

		err := b.changeResultToBlockNumberStruct(res, bnHolder, bnMethodsBlockNumber, originalMethod)
		if err != nil {
			return nil, err
		}
	}

	return cleanRes, nil
}

func convertFromResultToBNString(result interface{}) (string, error) {
	resMap, ok := result.(map[string]interface{})
	if !ok {
		return "", ErrInternalBlockNumberMethodNotMap
	}

	block, ok := resMap["number"].(string)
	if !ok {
		return "", ErrInternalBlockNumberMethodNotNumberEntry
	}

	return block, nil
}

func getBlockNumber(res *models.RPCResJSON, bnHolder map[string]*models.RPCResJSON, bnMethodsBlockNumber map[string][]*rpc.BlockNumberOrHash) ([]string, *models.RPCErr, error) {
	bns := bnMethodsBlockNumber[string(res.ID)]

	var blocks []string
	var bnError *models.RPCErr
	for _, bn := range bns {
		if bns[0].BlockHash != nil {
			bnh := bnHolder[bn.BlockHash.String()]
			if bnh.Error != nil {
				bnError = bnh.Error
				break // if there was an error the rest of the response is invalid
			}
			bnString, err := convertFromResultToBNString(bnh.Result)
			if err != nil {
				return nil, nil, err
			}
			blocks = append(blocks, bnString)
			continue
		}

		bnString := bn.BlockNumber.String()
		tagBlock, ok := bnHolder[bnString]
		if ok {
			if tagBlock.Error != nil {
				bnError = tagBlock.Error
				break // if there was an error the rest of the response is invalid
			}
			tagBlockString, err := convertFromResultToBNString(tagBlock.Result)
			if err != nil {
				return nil, nil, err
			}
			blocks = append(blocks, tagBlockString)
			continue
		}
		blocks = append(blocks, bnString)
	}

	return blocks, bnError, nil
}

func (b *BlockNumberConv) changeResultToBlockNumberStruct(res *models.RPCResJSON, bnHolder map[string]*models.RPCResJSON, bnMethodsBlockNumber map[string][]*rpc.BlockNumberOrHash, originalMethod string) error {
	if res.Error != nil {
		return nil // if there is an error the rest of the response is invalid
	}
	blockNumber, bnError, err := getBlockNumber(res, bnHolder, bnMethodsBlockNumber)
	if err != nil {
		return err
	}
	if bnError != nil {
		res.Result = nil // if there is an error the rest of the response is invalid
		res.Error = bnError
		return nil
	}

	if b.blockNumberMethodToIsBlockRange[originalMethod] {
		fromBlock := blockNumber[0]
		toBlock := blockNumber[0]
		if len(blockNumber) > 1 {
			toBlock = blockNumber[1]
		}
		res.Result = blockRangeResult{
			Data:          res.Result,
			StartingBlock: fromBlock,
			EndingBlock:   toBlock,
		}
	} else {
		res.Result = blockNumberResult{
			Data:        res.Result,
			BlockNumber: blockNumber[0],
		}
	}

	return nil
}
