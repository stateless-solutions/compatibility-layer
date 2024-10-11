package customrpcmethods

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	gethRPC "github.com/ethereum/go-ethereum/rpc"
	solanaRPC "github.com/gagliardetto/solana-go/rpc"
	"github.com/stateless-solutions/stateless-compatibility-layer/models"
)

type ChainType string

const (
	ChainTypeEVM    ChainType = "evm"
	ChainTypeSolana ChainType = "solana"
)

var (
	validChainTypes = map[ChainType]bool{
		ChainTypeEVM:    true,
		ChainTypeSolana: true,
	}

	chainTypeToCustomHandlerHolder = map[ChainType]interface{}{
		ChainTypeEVM: evmCustomHandlersHolder{},
	}

	rangeSupportChainTypes = map[ChainType]bool{
		ChainTypeEVM: true,
	}

	errDifferentChainTypes = errors.New("different chain types in the same batch")
)

type Method struct {
	OriginalMethod       string `json:"originalMethod"`
	CustomMethod         string `json:"customMethod"`
	PositionsGetterParam []int  `json:"positionsGetterParam,omitempty"`
	CustomHandler        string `json:"customHandler,omitempty"`
	IsRange              bool   `json:"isRange"`
}

type MethodsConfig struct {
	ChainType ChainType `json:"chainType"`
	Methods   []Method  `json:"methods"`
}

// GetterTypes is a generic for the type of the data that needs to be gotten from a chain type
// this one is most likely related to blocks and can be input as tags
type GetterTypes interface {
	*gethRPC.BlockNumberOrHash | solanaRPC.CommitmentType
}

type GetterReturns interface {
	string | int
}
type GetterStructs interface {
	blockNumberResult | contextResult
}

type noRangeSupported struct{} // placeholder struct for chains that don't support ranges
type GetterRangeStructs interface {
	blockRangeResult | noRangeSupported
}

// functions in this interface are in the order they are called
type CustomRpcMethodBuilderGeneric[T GetterTypes] interface {
	// GetCustomMethodsMap returns map of custom rpc methods to GetterType slice, slice needed in case of range
	GetCustomMethodsMap(rpcReqs []*models.RPCReq) (map[string][]T, error)
	// ChangeCustomMethods changes custom methods in rpc reqs slice to their original counterparts and returns map of the ID to original method
	ChangeCustomMethods(rpcReqs []*models.RPCReq) (map[string]string, error)
	// AddGetterMethodsIfNeeded returns rpc reqs with the getter rpc method for the getter struct and a map of tags to ID of the getter method
	AddGetterMethodsIfNeeded(rpcReqs []*models.RPCReq, customMethodsMap map[string][]T) ([]*models.RPCReq, map[string]string, error)
	// ChangeCustomMethodsResponses returns responses originally input as rpc reqs based on the responses of previous funcs
	ChangeCustomMethodsResponses(responses []*models.RPCResJSON, changedMethods, idsHolder map[string]string, customMethodsMap map[string][]T) ([]*models.RPCResJSON, error)
}

// struct needed because we cannot diretcly assign CustomRpcMethodBuilderGeneric in a map
type CustomMethodBuilders struct {
	EVM    CustomRpcMethodBuilderGeneric[*gethRPC.BlockNumberOrHash]
	Solana CustomRpcMethodBuilderGeneric[solanaRPC.CommitmentType]
}

type CustomMethodHolder struct {
	CustomRpcMethodBuilders   CustomMethodBuilders
	CustomMethodToChainType   map[string]ChainType
	OriginalMethodToChainType map[string]ChainType
}

func NewCustomMethodHolder(configFiles string) *CustomMethodHolder {
	ch := &CustomMethodHolder{
		CustomMethodToChainType:   map[string]ChainType{},
		OriginalMethodToChainType: map[string]ChainType{},
	}

	configsMap := make(map[ChainType][]MethodsConfig, len(validChainTypes))
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

		if !validChainTypes[config.ChainType] {
			panic(fmt.Sprintf("invalid chain type: %s", config.ChainType))
		}

		configsMap[config.ChainType] = append(configsMap[config.ChainType], config)

		for _, method := range config.Methods {
			ch.CustomMethodToChainType[method.CustomMethod] = config.ChainType
			ch.OriginalMethodToChainType[method.OriginalMethod] = config.ChainType
		}
	}

	for chainType, configs := range configsMap {
		if chainType == ChainTypeEVM {
			ch.CustomRpcMethodBuilders.EVM = NewBlockNumberConv(configs)
		}
		if chainType == ChainTypeSolana {
			ch.CustomRpcMethodBuilders.Solana = NewContextConv(configs)
		}
	}

	return ch
}

func (ch *CustomMethodHolder) getChainTypeFromRPCReqCustomMethods(rpcReqs []*models.RPCReq) (ChainType, error) {
	var customMethods []string
	for _, rpcReq := range rpcReqs {
		customMethods = append(customMethods, rpcReq.Method)
	}

	return ch.getChainTypeFromCustomMethods(customMethods)
}

func (ch *CustomMethodHolder) getChainTypeFromCustomMethodsMaps(customMethodsMap map[string]string) (ChainType, error) {
	var customMethods []string
	for _, customMethod := range customMethodsMap {
		customMethods = append(customMethods, customMethod)
	}

	return ch.getChainTypeFromCustomMethods(customMethods)
}

func (ch *CustomMethodHolder) getChainTypeFromCustomMethods(customMethods []string) (ChainType, error) {
	var chainType ChainType
	for _, customMethod := range customMethods {
		reqChaintype, ok := ch.CustomMethodToChainType[customMethod]
		if ok {
			if chainType == "" {
				chainType = reqChaintype
			} else {
				if chainType != reqChaintype {
					return "", errDifferentChainTypes
				}
			}
		}
	}

	return chainType, nil
}

func (ch *CustomMethodHolder) getChainTypeFromRPCReqOriginalMethods(rpcReqs []*models.RPCReq) (ChainType, error) {
	var originalMethods []string
	for _, rpcReq := range rpcReqs {
		originalMethods = append(originalMethods, rpcReq.Method)
	}

	return ch.getChainTypeFromOriginalMethods(originalMethods)
}

func (ch *CustomMethodHolder) getChainTypeFromOriginalMethods(originalMethods []string) (ChainType, error) {
	var chainType ChainType
	for _, originalMethod := range originalMethods {
		reqChaintype, ok := ch.OriginalMethodToChainType[originalMethod]
		if ok {
			if chainType == "" {
				chainType = reqChaintype
			} else {
				if chainType != reqChaintype {
					return "", errDifferentChainTypes
				}
			}
		}
	}

	return chainType, nil
}

func (ch *CustomMethodHolder) GetCustomMethodsMap(rpcReqs []*models.RPCReq) (interface{}, error) {
	chainType, err := ch.getChainTypeFromRPCReqCustomMethods(rpcReqs)
	if err != nil {
		return nil, err
	}
	if chainType == "" {
		return nil, nil
	}

	if chainType == ChainTypeEVM {
		return ch.CustomRpcMethodBuilders.EVM.GetCustomMethodsMap(rpcReqs)
	}
	if chainType == ChainTypeSolana {
		return ch.CustomRpcMethodBuilders.Solana.GetCustomMethodsMap(rpcReqs)
	}

	return nil, fmt.Errorf("unsupported chain type: %s", chainType)
}

func (ch *CustomMethodHolder) ChangeCustomMethods(rpcReqs []*models.RPCReq) (map[string]string, error) {
	chainType, err := ch.getChainTypeFromRPCReqCustomMethods(rpcReqs)
	if err != nil {
		return nil, err
	}
	if chainType == "" {
		return nil, nil
	}

	if chainType == ChainTypeEVM {
		return ch.CustomRpcMethodBuilders.EVM.ChangeCustomMethods(rpcReqs)
	}
	if chainType == ChainTypeSolana {
		return ch.CustomRpcMethodBuilders.Solana.ChangeCustomMethods(rpcReqs)
	}

	return nil, fmt.Errorf("unsupported chain type: %s", chainType)
}

func (ch *CustomMethodHolder) AddGetterMethodsIfNeeded(rpcReqs []*models.RPCReq, customMethodsMap interface{}) ([]*models.RPCReq, map[string]string, error) {
	chainType, err := ch.getChainTypeFromRPCReqOriginalMethods(rpcReqs)
	if err != nil {
		return nil, nil, err
	}
	if chainType == "" || customMethodsMap == nil {
		return rpcReqs, nil, nil
	}

	if chainType == ChainTypeEVM {
		customMethodsMapToSend := customMethodsMap.(map[string][]*gethRPC.BlockNumberOrHash)
		return ch.CustomRpcMethodBuilders.EVM.AddGetterMethodsIfNeeded(rpcReqs, customMethodsMapToSend)
	}
	if chainType == ChainTypeSolana {
		customMethodsMapToSend := customMethodsMap.(map[string][]solanaRPC.CommitmentType)
		return ch.CustomRpcMethodBuilders.Solana.AddGetterMethodsIfNeeded(rpcReqs, customMethodsMapToSend)
	}

	return nil, nil, fmt.Errorf("unsupported chain type: %s", chainType)
}

func (ch *CustomMethodHolder) ChangeCustomMethodsResponses(responses []*models.RPCResJSON, changedMethods, idsHolder map[string]string, customMethodsMap interface{}) ([]*models.RPCResJSON, error) {
	chainType, err := ch.getChainTypeFromCustomMethodsMaps(changedMethods)
	if err != nil {
		return nil, err
	}
	if chainType == "" || customMethodsMap == nil {
		return responses, nil
	}

	if chainType == ChainTypeEVM {
		customMethodsMapToSend := customMethodsMap.(map[string][]*gethRPC.BlockNumberOrHash)
		return ch.CustomRpcMethodBuilders.EVM.ChangeCustomMethodsResponses(responses, changedMethods, idsHolder, customMethodsMapToSend)
	}
	if chainType == ChainTypeSolana {
		customMethodsMapToSend := customMethodsMap.(map[string][]solanaRPC.CommitmentType)
		return ch.CustomRpcMethodBuilders.Solana.ChangeCustomMethodsResponses(responses, changedMethods, idsHolder, customMethodsMapToSend)
	}

	return nil, fmt.Errorf("unsupported chain type: %s", chainType)
}
