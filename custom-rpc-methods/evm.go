package customrpcmethods

import (
	"encoding/json"
	"fmt"

	gethRPC "github.com/ethereum/go-ethereum/rpc"
	"github.com/stateless-solutions/compatibility-layer/models"
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

	evmMethodNameToCustomHandlder = make(map[string]func(*models.RPCReq) ([]*gethRPC.BlockNumberOrHash, error))
)

func init() {
	SaveCustomHandlersToMap(evmCustomHandlersHolder{}, evmMethodNameToCustomHandlder)
}

func remarshalBlockNumberOrHash(current interface{}) (*gethRPC.BlockNumberOrHash, error) {
	jv, err := json.Marshal(current)
	if err != nil {
		return nil, err
	}

	var bnh gethRPC.BlockNumberOrHash
	err = bnh.UnmarshalJSON(jv)
	if err != nil {
		return nil, err
	}

	return &bnh, nil
}

type EVMImpl struct{}

func (e EVMImpl) GetChainType() ChainType {
	return ChainTypeEVM
}

func (e EVMImpl) SupportsRange() bool {
	return true
}

func (e EVMImpl) GetDefaultGetter() *gethRPC.BlockNumberOrHash {
	bnl, _ := remarshalBlockNumberOrHash("latest")
	return bnl
}

func (e EVMImpl) GetDefaultGetterRange() []*gethRPC.BlockNumberOrHash {
	bne, _ := remarshalBlockNumberOrHash("earliest")
	bnl, _ := remarshalBlockNumberOrHash("latest")
	return []*gethRPC.BlockNumberOrHash{bne, bnl}
}

func (e EVMImpl) GetCustomHandlerMap() map[string]func(*models.RPCReq) ([]*gethRPC.BlockNumberOrHash, error) {
	return evmMethodNameToCustomHandlder
}

func (e EVMImpl) FromGetterTypeToHolder(gth GetterTypesHolder) *gethRPC.BlockNumberOrHash {
	return gth.EVM
}

func (e EVMImpl) FromHolderToGetterType(gt *gethRPC.BlockNumberOrHash) GetterTypesHolder {
	return GetterTypesHolder{
		EVM: gt,
	}
}

func (e EVMImpl) ExtractGetter(param interface{}) (*gethRPC.BlockNumberOrHash, error) {
	bnh, err := remarshalBlockNumberOrHash(param)
	if err != nil {
		s, ok := param.(string)
		if ok {
			block, err := remarshalBlockNumberOrHash(s)
			if err != nil {
				return nil, ErrParseErr
			}
			return block, nil
		} else {
			return nil, ErrParseErr
		}
	}

	return bnh, nil
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

func (e EVMImpl) BuildGetterReq(id string, gt *gethRPC.BlockNumberOrHash) (*models.RPCReq, error) {
	if gt.BlockNumber != nil && gt.BlockHash != nil {
		return nil, ErrParseErr
	}

	if gt.BlockHash != nil {
		bH := gt.BlockHash.String()
		return buildGetBlockByHashReq(bH, id), nil
	}

	switch *gt.BlockNumber {
	case gethRPC.PendingBlockNumber:
		return buildGetBlockByNumberReq("pending", id), nil
	case gethRPC.EarliestBlockNumber:
		return buildGetBlockByNumberReq("earliest", id), nil
	case gethRPC.FinalizedBlockNumber:
		return buildGetBlockByNumberReq("finalized", id), nil
	case gethRPC.SafeBlockNumber:
		return buildGetBlockByNumberReq("safe", id), nil
	case gethRPC.LatestBlockNumber:
		return buildGetBlockByNumberReq("latest", id), nil
	}

	return nil, ErrParseErr
}

func (e EVMImpl) GetIndexOfIDHolder(gt *gethRPC.BlockNumberOrHash) (string, error) {
	if gt.BlockNumber != nil && gt.BlockHash != nil {
		return "", ErrParseErr
	}

	if gt.BlockHash != nil {
		return gt.BlockHash.String(), nil
	}

	switch *gt.BlockNumber {
	case gethRPC.PendingBlockNumber:
		return "pending", nil
	case gethRPC.EarliestBlockNumber:
		return "earliest", nil
	case gethRPC.FinalizedBlockNumber:
		return "finalized", nil
	case gethRPC.SafeBlockNumber:
		return "safe", nil
	case gethRPC.LatestBlockNumber:
		return "latest", nil
	}

	return "", nil
}

func (e EVMImpl) ExtractGetterReturnFromResponse(res *models.RPCResJSON) (string, error) {
	resMap, ok := res.Result.(map[string]interface{})
	if !ok {
		return "", ErrInternalBlockNumberMethodNotMap
	}

	block, ok := resMap["number"].(string)
	if !ok {
		return "", ErrInternalBlockNumberMethodNotNumberEntry
	}

	return block, nil
}

func (e EVMImpl) ExtractGetterReturnFromType(gt *gethRPC.BlockNumberOrHash) (string, error) {
	if gt.BlockHash != nil {
		return "", ErrParseErr
	}

	switch *gt.BlockNumber {
	case gethRPC.PendingBlockNumber:
		return "", ErrParseErr
	case gethRPC.EarliestBlockNumber:
		return "", ErrParseErr
	case gethRPC.FinalizedBlockNumber:
		return "", ErrParseErr
	case gethRPC.SafeBlockNumber:
		return "", ErrParseErr
	case gethRPC.LatestBlockNumber:
		return "", ErrParseErr
	}

	return gt.BlockNumber.String(), nil
}

func (e EVMImpl) ExtractGetterStruct(res *models.RPCResJSON, gr string) (blockNumberResult, error) {
	return blockNumberResult{
		Data:        res.Result,
		BlockNumber: gr,
	}, nil
}

func (e EVMImpl) ExtractGetterRangeStruct(res *models.RPCResJSON, grTo, grFrom string) (blockRangeResult, error) {
	return blockRangeResult{
		Data:          res.Result,
		StartingBlock: grTo,
		EndingBlock:   grFrom,
	}, nil
}

func NewEVMMethodBuilder() CustomRpcMethodBuilder {
	return NewGenericConv(EVMImpl{})
}
