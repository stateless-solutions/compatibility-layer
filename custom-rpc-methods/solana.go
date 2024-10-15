package customrpcmethods

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	solanaRPC "github.com/gagliardetto/solana-go/rpc"
	"github.com/stateless-solutions/stateless-compatibility-layer/models"
)

type context struct {
	Slot int `json:"slot"`
}

type contextResult struct {
	Value   interface{} `json:"value"`
	Context context     `json:"context"`
}

var (
	ErrInternalSlotResultNotExpectedType = &models.RPCErr{
		Code:          JSONRPCErrorInternal - 25,
		Message:       "slot response is not of an expected type",
		HTTPErrorCode: 500,
	}
)

type SolanaImpl struct{}

func (s SolanaImpl) GetChainType() ChainType {
	return ChainTypeSolana
}

func (s SolanaImpl) SupportsRange() bool {
	return false
}

func (s SolanaImpl) GetDefaultGetter() solanaRPC.CommitmentType {
	return solanaRPC.CommitmentFinalized
}

func (s SolanaImpl) GetDefaultGetterRange() []solanaRPC.CommitmentType {
	return nil
}

func (s SolanaImpl) GetCustomHandlerMap() map[string]func(*models.RPCReq) ([]solanaRPC.CommitmentType, error) {
	return nil
}

func (s SolanaImpl) FromGetterTypeToHolder(gth GetterTypesHolder) solanaRPC.CommitmentType {
	return gth.Solana
}

func (s SolanaImpl) FromHolderToGetterType(gt solanaRPC.CommitmentType) GetterTypesHolder {
	return GetterTypesHolder{
		Solana: gt,
	}
}

func (s SolanaImpl) ExtractGetter(param interface{}) (solanaRPC.CommitmentType, error) {
	cMap, ok := param.(map[string]interface{})
	if !ok {
		return s.GetDefaultGetter(), nil // params can be a string in some cases, in which default commitment should be used
	}

	cTypeRaw, ok := cMap["commitment"]
	if !ok {
		return s.GetDefaultGetter(), nil // this just means no commitment was input and should return default
	}

	cTypeString, ok := cTypeRaw.(string)
	if !ok {
		return "", ErrParseErr
	}

	return solanaRPC.CommitmentType(cTypeString), nil
}

var validCommitmentTypes = map[solanaRPC.CommitmentType]bool{
	solanaRPC.CommitmentConfirmed: true,
	solanaRPC.CommitmentFinalized: true,
	solanaRPC.CommitmentProcessed: true,
}

func (s SolanaImpl) BuildGetterReq(id string, gt solanaRPC.CommitmentType) (*models.RPCReq, error) {
	if !validCommitmentTypes[gt] {
		return nil, ErrParseErr
	}

	return &models.RPCReq{
		JSONRPC: "2.0",
		Method:  "getSlot",
		ID:      json.RawMessage(id),
		Params:  json.RawMessage(fmt.Sprintf(`[{"commitment":"%s"}]`, gt)),
	}, nil
}

func (s SolanaImpl) GetIndexOfIDHolder(gt solanaRPC.CommitmentType) (string, error) {
	if !validCommitmentTypes[gt] {
		return "", ErrParseErr
	}

	return string(gt), nil
}

func (s SolanaImpl) ExtractGetterReturnFromResponse(res *models.RPCResJSON) (int, error) {
	switch v := res.Result.(type) {
	case float64:
		floatStr := strconv.FormatFloat(v, 'f', -1, 64)
		floatStrWithoutDot := strings.Replace(floatStr, ".", "", 1)
		return strconv.Atoi(floatStrWithoutDot)
	case int:
		return v, nil
	default:
		return 0, ErrInternalSlotResultNotExpectedType
	}
}

func (s SolanaImpl) ExtractGetterReturnFromType(gt solanaRPC.CommitmentType) (int, error) {
	return 0, ErrParseErr
}

func (s SolanaImpl) ExtractGetterStruct(res *models.RPCResJSON, gr int) (contextResult, error) {
	return contextResult{
		Value: res.Result,
		Context: context{
			Slot: gr,
		},
	}, nil
}

func (s SolanaImpl) ExtractGetterRangeStruct(res *models.RPCResJSON, grTo, grFrom int) (noRangeSupported, error) {
	return noRangeSupported{}, ErrParseErr
}

func NewSolanaMethodBuilder() CustomRpcMethodBuilder {
	return NewGenericConv(SolanaImpl{})
}
