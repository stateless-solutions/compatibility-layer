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
	ErrInternalSlotResultNotFloat = &models.RPCErr{
		Code:          JSONRPCErrorInternal - 25,
		Message:       "slot response is not a float",
		HTTPErrorCode: 500,
	}
)

type ContextConv struct {
	contextToRegular   map[string]string
	contextMethodToPos map[string][]int
}

func NewContextConv(configs []MethodsConfig) *ContextConv {
	cc := &ContextConv{
		contextToRegular:   map[string]string{},
		contextMethodToPos: map[string][]int{},
	}

	for _, config := range configs {
		for _, method := range config.Methods {
			cc.contextToRegular[method.CustomMethod] = method.OriginalMethod
			cc.contextMethodToPos[method.CustomMethod] = method.PositionsGetterParam
		}
	}

	return cc
}

func extractCommitmentType(current interface{}) (solanaRPC.CommitmentType, error) {
	cMap, ok := current.(map[string]interface{})
	if !ok {
		return solanaRPC.CommitmentFinalized, nil // params can be a string in some cases, in which default commitment should be used
	}

	cTypeRaw, ok := cMap["commitment"]
	if !ok {
		return solanaRPC.CommitmentFinalized, nil // this just means no commitment was input and should return default
	}

	cTypeString, ok := cTypeRaw.(string)
	if !ok {
		return "", ErrParseErr
	}

	return solanaRPC.CommitmentType(cTypeString), nil
}

func (b *ContextConv) getCommitmentTypes(req *models.RPCReq) ([]solanaRPC.CommitmentType, error) {
	// in case params are empty
	// default commitment is used: finalized
	if req.Params == nil {
		return []solanaRPC.CommitmentType{solanaRPC.CommitmentFinalized}, nil
	}

	_, ok := b.contextToRegular[req.Method]
	if ok {
		var p []interface{}
		err := json.Unmarshal(req.Params, &p)
		if err != nil {
			return nil, err
		}

		poss := b.contextMethodToPos[req.Method]

		// in case no param position specified
		// default commitment is used: finalized
		if len(poss) == 0 {
			return []solanaRPC.CommitmentType{solanaRPC.CommitmentFinalized}, nil
		}

		var cs []solanaRPC.CommitmentType
		for _, pos := range poss {
			if len(p) <= pos {
				return []solanaRPC.CommitmentType{solanaRPC.CommitmentFinalized}, nil // TODO: this doesn't work with range, for generic it should work
			}

			cType, err := extractCommitmentType(p[pos])
			if err != nil {
				return nil, ErrParseErr
			}

			cs = append(cs, cType)
		}

		return cs, nil
	}

	return nil, nil
}

func (b *ContextConv) GetCustomMethodsMap(rpcReqs []*models.RPCReq) (map[string][]solanaRPC.CommitmentType, error) {
	contextMethodsCommitmentType := make(map[string][]solanaRPC.CommitmentType, len(rpcReqs))

	for _, req := range rpcReqs {
		ct, err := b.getCommitmentTypes(req)
		if err != nil {
			return nil, err
		}
		if ct != nil {
			contextMethodsCommitmentType[string(req.ID)] = ct
		}
	}

	return contextMethodsCommitmentType, nil
}

func buildGetSlotReq(id string, cType solanaRPC.CommitmentType) *models.RPCReq {
	return &models.RPCReq{
		JSONRPC: "2.0",
		Method:  "getSlot",
		ID:      json.RawMessage(id),
		Params:  json.RawMessage(fmt.Sprintf(`[{"commitment":"%s"}]`, cType)),
	}
}

var validCommitmentTypes = map[solanaRPC.CommitmentType]bool{
	solanaRPC.CommitmentConfirmed: true,
	solanaRPC.CommitmentFinalized: true,
	solanaRPC.CommitmentProcessed: true,
}

func (b *ContextConv) AddGetterMethodsIfNeeded(rpcReqs []*models.RPCReq, ctMethodsCommitmentType map[string][]solanaRPC.CommitmentType) ([]*models.RPCReq, map[string]string, error) {
	idsHolder := make(map[string]string, len(ctMethodsCommitmentType))

	for _, cts := range ctMethodsCommitmentType {
		for _, ct := range cts {
			if !validCommitmentTypes[ct] {
				return nil, nil, ErrParseErr
			}

			_, ok := idsHolder[string(ct)]
			if !ok {
				id, err := generateRandomNumberStringWithRetries(rpcReqs)
				if err != nil {
					return nil, nil, err
				}
				idsHolder[string(ct)] = id
				rpcReqs = append(rpcReqs, buildGetSlotReq(id, ct))
			}
		}
	}

	return rpcReqs, idsHolder, nil
}

func (b *ContextConv) ChangeCustomMethods(rpcReqs []*models.RPCReq) (map[string]string, error) {
	changedMethods := make(map[string]string, len(rpcReqs))

	for _, rpcReq := range rpcReqs {
		regMethod, ok := b.contextToRegular[rpcReq.Method]
		if !ok {
			continue
		}

		changedMethods[string(rpcReq.ID)] = rpcReq.Method
		rpcReq.Method = regMethod
	}

	return changedMethods, nil
}

func getCommitmentHolder(responses []*models.RPCResJSON, idsHolder map[string]string) (map[string]*models.RPCResJSON, []*models.RPCResJSON, error) {
	cHolder := make(map[string]*models.RPCResJSON, len(idsHolder))
	var responsesWithoutCT []*models.RPCResJSON

	for _, res := range responses {
		var ctMethod bool
		for content, id := range idsHolder {
			if string(res.ID) == id {
				cHolder[content] = res
				ctMethod = true
			}
		}
		if !ctMethod {
			responsesWithoutCT = append(responsesWithoutCT, res)
		}
	}

	return cHolder, responsesWithoutCT, nil
}

func (b *ContextConv) ChangeCustomMethodsResponses(responses []*models.RPCResJSON, changedMethods, idsHolder map[string]string, ctMethodsCommitmentType map[string][]solanaRPC.CommitmentType) ([]*models.RPCResJSON, error) {
	cHolder, cleanRes, err := getCommitmentHolder(responses, idsHolder)
	if err != nil {
		return nil, err
	}

	for _, res := range cleanRes {
		originalMethod, ok := changedMethods[string(res.ID)]
		if !ok {
			continue
		}

		err := b.changeResultToContextStruct(res, cHolder, ctMethodsCommitmentType, originalMethod)
		if err != nil {
			return nil, err
		}
	}

	return cleanRes, nil
}

func getCommitment(res *models.RPCResJSON, ctHolder map[string]*models.RPCResJSON, ctMethodsCommitmentType map[string][]solanaRPC.CommitmentType) ([]int, *models.RPCErr, error) {
	cts := ctMethodsCommitmentType[string(res.ID)]

	var commitments []int
	var ctError *models.RPCErr
	for _, ct := range cts {
		cth := ctHolder[string(ct)]
		if cth.Error != nil {
			ctError = cth.Error
			break // if there was an error the rest of the response is invalid
		}

		ctFloat, ok := cth.Result.(float64)
		if !ok {
			return nil, nil, ErrInternalSlotResultNotFloat
		}

		floatStr := strconv.FormatFloat(ctFloat, 'f', -1, 64)
		floatStrWithoutDot := strings.Replace(floatStr, ".", "", 1)

		ctInt, err := strconv.Atoi(floatStrWithoutDot)
		if err != nil {
			return nil, nil, err
		}

		commitments = append(commitments, ctInt)
	}

	return commitments, ctError, nil
}

func (b *ContextConv) changeResultToContextStruct(res *models.RPCResJSON, cHolder map[string]*models.RPCResJSON, ctMethodsCommitmentType map[string][]solanaRPC.CommitmentType, originalMethod string) error {
	if res.Error != nil {
		return nil // if there is an error the rest of the response is invalid
	}
	slot, ctError, err := getCommitment(res, cHolder, ctMethodsCommitmentType)
	if err != nil {
		return err
	}
	if ctError != nil {
		res.Result = nil // if there is an error the rest of the response is invalid
		res.Error = ctError
		return nil
	}

	res.Result = contextResult{
		Value: res.Result,
		Context: context{
			Slot: slot[0],
		},
	}

	return nil
}
