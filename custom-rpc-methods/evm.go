package customrpcmethods

import (
	"encoding/json"
	"fmt"
	"reflect"

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

var (
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
)

// this validates on start if all custom handlers have the correct structure
// reflect was used because it was the easiest form to iterate over the methods of a structure
func init() {
	expectedType := reflect.TypeOf(func(evmCustomHandlersHolder, *models.RPCReq) ([]*rpc.BlockNumberOrHash, error) { return nil, nil })
	holderType := reflect.TypeOf(evmCustomHandlersHolder{})

	for i := 0; i < holderType.NumMethod(); i++ {
		method := holderType.Method(i)

		if method.Type != expectedType {
			panic(fmt.Sprintf("method %s has an incorrect signature: expected %v, got %v", method.Name, expectedType, method.Type))
		}
	}
}

type BlockNumberConv struct {
	blockNumberToRegular             map[string]string
	blockNumberMethodToPos           map[string][]int
	blockNumberMethodToIsBlockRange  map[string]bool
	blockNumberMethodToCustomHandler map[string]func(*models.RPCReq) ([]*rpc.BlockNumberOrHash, error)
}

func NewBlockNumberConv(configs []MethodsConfig) *BlockNumberConv {
	bnc := &BlockNumberConv{
		blockNumberToRegular:             map[string]string{},
		blockNumberMethodToPos:           map[string][]int{},
		blockNumberMethodToIsBlockRange:  map[string]bool{},
		blockNumberMethodToCustomHandler: map[string]func(*models.RPCReq) ([]*rpc.BlockNumberOrHash, error){},
	}

	for _, config := range configs {
		for _, method := range config.Methods {
			bnc.blockNumberToRegular[method.CustomMethod] = method.OriginalMethod
			bnc.blockNumberMethodToPos[method.CustomMethod] = method.PositionsGetterParam
			bnc.blockNumberMethodToIsBlockRange[method.CustomMethod] = method.IsRange
			if method.CustomHandler != "" {
				handler := reflect.ValueOf(evmCustomHandlersHolder{}).MethodByName(method.CustomHandler)
				if !handler.IsValid() {
					panic(fmt.Sprintf("custom handler %s for method %s is not implemented", method.CustomHandler, method.CustomMethod))
				}
				// on init it already validates all the methods on custom handler are of the expected signature
				handlerFunc := handler.Interface().(func(*models.RPCReq) ([]*rpc.BlockNumberOrHash, error))
				bnc.blockNumberMethodToCustomHandler[method.CustomMethod] = handlerFunc
			}
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

// TODO: fix this, pos can actually be shorter on other chains
func (b *BlockNumberConv) getBlockNumbers(req *models.RPCReq) ([]*rpc.BlockNumberOrHash, error) {
	// in case params are empty
	// default block tags are used: latest for non range and earliest to latest in range
	if req.Params == nil {
		if b.blockNumberMethodToIsBlockRange[req.Method] {
			bnl, _ := remarshalBlockNumberOrHash("latest")
			bne, _ := remarshalBlockNumberOrHash("earliest")
			return []*rpc.BlockNumberOrHash{bne, bnl}, nil
		} else {
			bnl, _ := remarshalBlockNumberOrHash("latest")
			return []*rpc.BlockNumberOrHash{bnl}, nil
		}
	}

	_, ok := b.blockNumberToRegular[req.Method]
	if ok {
		customHandler, ok := b.blockNumberMethodToCustomHandler[req.Method]
		if ok {
			return customHandler(req)
		}

		var p []interface{}
		err := json.Unmarshal(req.Params, &p)
		if err != nil {
			return nil, err
		}

		// in case no param position or params are empty
		// default block tags are used: latest for non range and earliest to latest in range
		if len(b.blockNumberMethodToPos[req.Method]) == 0 || len(p) == 0 {
			if b.blockNumberMethodToIsBlockRange[req.Method] {
				bnl, _ := remarshalBlockNumberOrHash("latest")
				bne, _ := remarshalBlockNumberOrHash("earliest")
				return []*rpc.BlockNumberOrHash{bne, bnl}, nil
			} else {
				bnl, _ := remarshalBlockNumberOrHash("latest")
				return []*rpc.BlockNumberOrHash{bnl}, nil
			}
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

func (b *BlockNumberConv) GetCustomMethodsMap(rpcReqs []*models.RPCReq) (map[string][]*rpc.BlockNumberOrHash, error) {
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

func (b *BlockNumberConv) AddGetterMethodsIfNeeded(rpcReqs []*models.RPCReq, bnMethodsBlockNumber map[string][]*rpc.BlockNumberOrHash) ([]*models.RPCReq, map[string]string, error) {
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
					id, err := generateRandomNumberStringWithRetries(rpcReqs)
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
					id, err := generateRandomNumberStringWithRetries(rpcReqs)
					if err != nil {
						return nil, nil, err
					}
					idsHolder["pending"] = id
					rpcReqs = append(rpcReqs, buildGetBlockByNumberReq("pending", id))
				}
			case rpc.EarliestBlockNumber:
				_, ok := idsHolder["earliest"]
				if !ok {
					id, err := generateRandomNumberStringWithRetries(rpcReqs)
					if err != nil {
						return nil, nil, err
					}
					idsHolder["earliest"] = id
					rpcReqs = append(rpcReqs, buildGetBlockByNumberReq("earliest", id))
				}
			case rpc.FinalizedBlockNumber:
				_, ok := idsHolder["finalized"]
				if !ok {
					id, err := generateRandomNumberStringWithRetries(rpcReqs)
					if err != nil {
						return nil, nil, err
					}
					idsHolder["finalized"] = id
					rpcReqs = append(rpcReqs, buildGetBlockByNumberReq("finalized", id))
				}
			case rpc.SafeBlockNumber:
				_, ok := idsHolder["safe"]
				if !ok {
					id, err := generateRandomNumberStringWithRetries(rpcReqs)
					if err != nil {
						return nil, nil, err
					}
					idsHolder["safe"] = id
					rpcReqs = append(rpcReqs, buildGetBlockByNumberReq("safe", id))
				}
			case rpc.LatestBlockNumber:
				_, ok := idsHolder["latest"]
				if !ok {
					id, err := generateRandomNumberStringWithRetries(rpcReqs)
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

func (b *BlockNumberConv) ChangeCustomMethods(rpcReqs []*models.RPCReq) (map[string]string, error) {
	changedMethods := make(map[string]string, len(rpcReqs))

	for _, rpcReq := range rpcReqs {
		regMethod, ok := b.blockNumberToRegular[rpcReq.Method]
		if !ok {
			continue
		}

		changedMethods[string(rpcReq.ID)] = rpcReq.Method
		rpcReq.Method = regMethod
	}

	return changedMethods, nil
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

func (b *BlockNumberConv) ChangeCustomMethodsResponses(responses []*models.RPCResJSON, changedMethods, idsHolder map[string]string, bnMethodsBlockNumber map[string][]*rpc.BlockNumberOrHash) ([]*models.RPCResJSON, error) {
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
		if bn.BlockHash != nil {
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
