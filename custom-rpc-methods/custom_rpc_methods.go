package customrpcmethods

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stateless-solutions/stateless-compatibility-layer/models"
)

type ChainType string

var (
	ChainTypeEVM ChainType = "evm"

	validChainTypes = map[ChainType]bool{
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

type CustomRpcMethodBuilder interface {
	GetCustomMethodsMap(rpcReqs []*models.RPCReq) (map[string][]*rpc.BlockNumberOrHash, error)
	ChangeCustomMethods(rpcReqs []*models.RPCReq) (map[string]string, error)
	AddGetterMethodsIfNeeded(rpcReqs []*models.RPCReq, bnMethodsBlockNumber map[string][]*rpc.BlockNumberOrHash) ([]*models.RPCReq, map[string]string, error)
	ChangeCustomMethodsResponses(responses []*models.RPCResJSON, changedMethods, idsHolder map[string]string, bnMethodsBlockNumber map[string][]*rpc.BlockNumberOrHash) ([]*models.RPCResJSON, error)
}

type CustomMethodHolder struct {
	CustomRpcMethodBuilders   map[ChainType]CustomRpcMethodBuilder
	CustomMethodToChainType   map[string]ChainType
	OriginalMethodToChainType map[string]ChainType
}

func NewCustomMethodHolder(configFiles string) *CustomMethodHolder {
	ch := &CustomMethodHolder{
		CustomRpcMethodBuilders:   make(map[ChainType]CustomRpcMethodBuilder, len(validChainTypes)),
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
			ch.CustomRpcMethodBuilders[ChainTypeEVM] = NewBlockNumberConv(configs)
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

func (ch *CustomMethodHolder) GetCustomMethodsMap(rpcReqs []*models.RPCReq) (map[string][]*rpc.BlockNumberOrHash, error) {
	chainType, err := ch.getChainTypeFromRPCReqCustomMethods(rpcReqs)
	if err != nil {
		return nil, err
	}
	if chainType == "" {
		return nil, nil
	}

	return ch.CustomRpcMethodBuilders[chainType].GetCustomMethodsMap(rpcReqs)
}

func (ch *CustomMethodHolder) ChangeCustomMethods(rpcReqs []*models.RPCReq) (map[string]string, error) {
	chainType, err := ch.getChainTypeFromRPCReqCustomMethods(rpcReqs)
	if err != nil {
		return nil, err
	}
	if chainType == "" {
		return nil, nil
	}

	return ch.CustomRpcMethodBuilders[chainType].ChangeCustomMethods(rpcReqs)
}

func (ch *CustomMethodHolder) AddGetterMethodsIfNeeded(rpcReqs []*models.RPCReq, bnMethodsBlockNumber map[string][]*rpc.BlockNumberOrHash) ([]*models.RPCReq, map[string]string, error) {
	chainType, err := ch.getChainTypeFromRPCReqOriginalMethods(rpcReqs)
	if err != nil {
		return nil, nil, err
	}
	if chainType == "" {
		return rpcReqs, nil, nil
	}

	return ch.CustomRpcMethodBuilders[chainType].AddGetterMethodsIfNeeded(rpcReqs, bnMethodsBlockNumber)
}

func (ch *CustomMethodHolder) ChangeCustomMethodsResponses(responses []*models.RPCResJSON, changedMethods, idsHolder map[string]string, bnMethodsBlockNumber map[string][]*rpc.BlockNumberOrHash) ([]*models.RPCResJSON, error) {
	chainType, err := ch.getChainTypeFromCustomMethodsMaps(changedMethods)
	if err != nil {
		return nil, err
	}
	if chainType == "" {
		return responses, nil
	}

	return ch.CustomRpcMethodBuilders[chainType].ChangeCustomMethodsResponses(responses, changedMethods, idsHolder, bnMethodsBlockNumber)
}
