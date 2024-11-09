package customrpcmethods

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	gethRPC "github.com/ethereum/go-ethereum/rpc"
	solanaRPC "github.com/gagliardetto/solana-go/rpc"
	"github.com/stateless-solutions/compatibility-layer/models"
)

type ChainType string

const (
	ChainTypeEVM    ChainType = "evm"
	ChainTypeSolana ChainType = "solana"
)

var (
	chainTypeToMethodBuilder = map[ChainType]CustomRpcMethodBuilder{
		ChainTypeEVM:    NewEVMMethodBuilder(),
		ChainTypeSolana: NewSolanaMethodBuilder(),
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

// this structure is needed bc you can't return directly a generic in a non generic func
type GetterTypesHolder struct {
	EVM    *gethRPC.BlockNumberOrHash
	Solana solanaRPC.CommitmentType
}

// functions in this interface are in the order they are called
type CustomRpcMethodBuilder interface {
	// PopulateConfig passes the config to the method builder memory
	PopulateConfig(gatewayMode bool, configs []MethodsConfig)
	// HandleGatewayMode changes the rpc methods from regular to custom in case gateway mode is on
	HandleGatewayMode(rpcReqs []*models.RPCReq) ([]*models.RPCReq, error)
	// GetCustomMethodsMap returns map of custom rpc methods to GetterType slice, slice needed in case of range
	GetCustomMethodsMap(rpcReqs []*models.RPCReq) (map[string][]GetterTypesHolder, error)
	// ChangeCustomMethods changes custom methods in rpc reqs slice to their original counterparts and returns map of the ID to original method
	ChangeCustomMethods(rpcReqs []*models.RPCReq) (map[string]string, error)
	// AddGetterMethodsIfNeeded returns rpc reqs with the getter rpc method for the getter struct and a map of tags to ID of the getter method
	AddGetterMethodsIfNeeded(rpcReqs []*models.RPCReq, customMethodsMap map[string][]GetterTypesHolder) ([]*models.RPCReq, map[string]string, error)
	// ChangeCustomMethodsResponses returns responses originally input as rpc reqs based on the responses of previous funcs
	ChangeCustomMethodsResponses(responses []*models.RPCResJSON, changedMethods, idsHolder map[string]string, customMethodsMap map[string][]GetterTypesHolder) ([]*models.RPCResJSON, error)
}

type CustomMethodHolder struct {
	ChainTypeToMethodBuilder  map[ChainType]CustomRpcMethodBuilder
	CustomMethodToChainType   map[string]ChainType
	OriginalMethodToChainType map[string]ChainType
}

func NewCustomMethodHolder(gatewayMode bool, configFiles string) *CustomMethodHolder {
	ch := &CustomMethodHolder{
		ChainTypeToMethodBuilder:  make(map[ChainType]CustomRpcMethodBuilder, len(chainTypeToMethodBuilder)),
		CustomMethodToChainType:   map[string]ChainType{},
		OriginalMethodToChainType: map[string]ChainType{},
	}

	configsMap := make(map[ChainType][]MethodsConfig, len(chainTypeToMethodBuilder))
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

		_, ok := chainTypeToMethodBuilder[config.ChainType]
		if !ok {
			panic(fmt.Sprintf("invalid chain type: %s", config.ChainType))
		}

		configsMap[config.ChainType] = append(configsMap[config.ChainType], config)

		for _, method := range config.Methods {
			ch.CustomMethodToChainType[method.CustomMethod] = config.ChainType
			ch.OriginalMethodToChainType[method.OriginalMethod] = config.ChainType
		}
	}

	for chainType, configs := range configsMap {
		methodBuilder := chainTypeToMethodBuilder[chainType]
		methodBuilder.PopulateConfig(gatewayMode, configs)
		ch.ChainTypeToMethodBuilder[chainType] = methodBuilder
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

func (ch *CustomMethodHolder) HandleGatewayMode(rpcReqs []*models.RPCReq) ([]*models.RPCReq, error) {
	chainType, err := ch.getChainTypeFromRPCReqOriginalMethods(rpcReqs)
	if err != nil {
		return nil, err
	}
	if chainType == "" {
		return rpcReqs, nil
	}

	return ch.ChainTypeToMethodBuilder[chainType].HandleGatewayMode(rpcReqs)
}

func (ch *CustomMethodHolder) GetCustomMethodsMap(rpcReqs []*models.RPCReq) (map[string][]GetterTypesHolder, error) {
	chainType, err := ch.getChainTypeFromRPCReqCustomMethods(rpcReqs)
	if err != nil {
		return nil, err
	}
	if chainType == "" {
		return nil, nil
	}

	return ch.ChainTypeToMethodBuilder[chainType].GetCustomMethodsMap(rpcReqs)
}

func (ch *CustomMethodHolder) ChangeCustomMethods(rpcReqs []*models.RPCReq) (map[string]string, error) {
	chainType, err := ch.getChainTypeFromRPCReqCustomMethods(rpcReqs)
	if err != nil {
		return nil, err
	}
	if chainType == "" {
		return nil, nil
	}

	return ch.ChainTypeToMethodBuilder[chainType].ChangeCustomMethods(rpcReqs)
}

func (ch *CustomMethodHolder) AddGetterMethodsIfNeeded(rpcReqs []*models.RPCReq, customMethodsMap map[string][]GetterTypesHolder) ([]*models.RPCReq, map[string]string, error) {
	chainType, err := ch.getChainTypeFromRPCReqOriginalMethods(rpcReqs)
	if err != nil {
		return nil, nil, err
	}
	if chainType == "" || customMethodsMap == nil {
		return rpcReqs, nil, nil
	}

	return ch.ChainTypeToMethodBuilder[chainType].AddGetterMethodsIfNeeded(rpcReqs, customMethodsMap)
}

func (ch *CustomMethodHolder) ChangeCustomMethodsResponses(responses []*models.RPCResJSON, changedMethods, idsHolder map[string]string, customMethodsMap map[string][]GetterTypesHolder) ([]*models.RPCResJSON, error) {
	chainType, err := ch.getChainTypeFromCustomMethodsMaps(changedMethods)
	if err != nil {
		return nil, err
	}
	if chainType == "" || customMethodsMap == nil {
		return responses, nil
	}

	return ch.ChainTypeToMethodBuilder[chainType].ChangeCustomMethodsResponses(responses, changedMethods, idsHolder, customMethodsMap)
}
