package customrpcmethods

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/stateless-solutions/stateless-compatibility-layer/models"
)

type GenericConv[T GetterTypes, R GetterReturns, S GetterStructs, SR GetterRangeStructs] struct {
	chainType                   ChainType
	defaultGetter               T
	defaultGetterRange          []T // always first one should from and second one to
	getterExtractor             func(interface{}) (T, error)
	getterReqBuilder            func(string, T) (*models.RPCReq, error)
	idHolderIndexExtractor      func(T) (string, error)
	getterReturnExtractor       func(interface{}) (R, error)
	getterStructBuilder         func(interface{}, R) S
	getterRangeStructBuilder    func(interface{}, R, R) SR
	customToRegular             map[string]string
	customMethodToPos           map[string][]int
	customMethodToIsRange       map[string]bool
	customMethodToCustomHandler map[string]func(*models.RPCReq) ([]T, error)
}

func NewGenericConv[T GetterTypes, R GetterReturns, S GetterStructs, SR GetterRangeStructs](configs []MethodsConfig,
	chainType ChainType, defaultGetter T, defaultGetterRange []T, getterExtractor func(interface{}) (T, error),
	getterReqBuilder func(string, T) (*models.RPCReq, error), idHolderIndexExtractor func(T) (string, error),
	getterReturnExtractor func(interface{}) (R, error), getterStructBuilder func(interface{}, R) S, getterRangeStructBuilder func(interface{}, R, R) SR) *GenericConv[T, R, S, SR] {

	gc := &GenericConv[T, R, S, SR]{
		chainType:                   chainType,
		defaultGetter:               defaultGetter,
		defaultGetterRange:          defaultGetterRange,
		getterExtractor:             getterExtractor,
		getterReqBuilder:            getterReqBuilder,
		idHolderIndexExtractor:      idHolderIndexExtractor,
		getterReturnExtractor:       getterReturnExtractor,
		getterStructBuilder:         getterStructBuilder,
		getterRangeStructBuilder:    getterRangeStructBuilder,
		customToRegular:             map[string]string{},
		customMethodToPos:           map[string][]int{},
		customMethodToIsRange:       map[string]bool{},
		customMethodToCustomHandler: map[string]func(*models.RPCReq) ([]T, error){},
	}

	for _, config := range configs {
		for _, method := range config.Methods {
			gc.customToRegular[method.CustomMethod] = method.OriginalMethod
			if len(method.PositionsGetterParam) > 2 {
				panic(fmt.Sprintf("positions getter param length for method %s is %d and the max allowed is 2", method.CustomMethod, len(method.PositionsGetterParam)))
			}
			gc.customMethodToPos[method.CustomMethod] = method.PositionsGetterParam
			if method.IsRange && !rangeSupportChainTypes[chainType] {
				panic(fmt.Sprintf("is range is true for method %s of chain type %s that doesn't support it", method.CustomMethod, chainType))
			}
			gc.customMethodToIsRange[method.CustomMethod] = method.IsRange
			if method.CustomHandler != "" {
				handler := reflect.ValueOf(chainTypeToCustomHandlerHolder[chainType]).MethodByName(method.CustomHandler)
				if !handler.IsValid() {
					panic(fmt.Sprintf("custom handler %s for method %s is not implemented", method.CustomHandler, method.CustomMethod))
				}
				// on init it already validates all the methods on custom handler are of the expected signature
				handlerFunc := handler.Interface().(func(*models.RPCReq) ([]T, error))
				gc.customMethodToCustomHandler[method.CustomMethod] = handlerFunc
			}
		}
	}

	return gc
}

func (g *GenericConv[T, R, S, SR]) returnDefaultGetters(req *models.RPCReq) []T {
	if g.customMethodToIsRange[req.Method] {
		return g.defaultGetterRange
	} else {
		return []T{g.defaultGetter}
	}
}

func (g *GenericConv[T, R, S, SR]) getGetters(req *models.RPCReq) ([]T, error) {
	// in case params are empty
	// default getters are used
	if req.Params == nil {
		return g.returnDefaultGetters(req), nil
	}

	_, ok := g.customToRegular[req.Method]
	if ok {
		customHandler, ok := g.customMethodToCustomHandler[req.Method]
		if ok {
			return customHandler(req)
		}

		var p []interface{}
		err := json.Unmarshal(req.Params, &p)
		if err != nil {
			return nil, err
		}

		// in case no param position or params are empty
		// default getters are used
		if len(g.customMethodToPos[req.Method]) == 0 || len(p) == 0 {
			return g.returnDefaultGetters(req), nil
		}

		var gts []T
		var defaultFromUsed bool
		for i, pos := range g.customMethodToPos[req.Method] {
			if len(p) <= pos {
				// in case params of the position are not present
				// default getters are used
				if i == 0 {
					if g.customMethodToIsRange[req.Method] {
						gts = append(gts, g.defaultGetterRange[0])
						defaultFromUsed = true
						continue
					} else {
						return []T{g.defaultGetter}, nil
					}
				}
				if i == 1 && g.customMethodToIsRange[req.Method] {
					if defaultFromUsed {
						gts = append(gts, g.defaultGetterRange[1])
					}
					// if to getter param is not present in range and from was
					// it will be assumed a range of one will be done
					continue
				}
				return nil, ErrParseErr
			}

			gt, err := g.getterExtractor(p[pos])
			if err != nil {
				return nil, err
			}

			gts = append(gts, gt)
		}

		return gts, nil
	}

	return nil, nil
}

func (g *GenericConv[T, R, S, SR]) GetCustomMethodsMap(rpcReqs []*models.RPCReq) (map[string][]T, error) {
	customMethodsGetter := make(map[string][]T, len(rpcReqs))

	for _, req := range rpcReqs {
		cm, err := g.getGetters(req)
		if err != nil {
			return nil, err
		}
		if cm != nil {
			customMethodsGetter[string(req.ID)] = cm
		}
	}

	return customMethodsGetter, nil
}

func (g *GenericConv[T, R, S, SR]) AddGetterMethodsIfNeeded(rpcReqs []*models.RPCReq, cMethodsGetter map[string][]T) ([]*models.RPCReq, map[string]string, error) {
	idsHolder := make(map[string]string, len(cMethodsGetter))

	for _, cs := range cMethodsGetter {
		for _, c := range cs {
			id, err := generateRandomNumberStringWithRetries(rpcReqs)
			if err != nil {
				return nil, nil, err
			}
			index, err := g.idHolderIndexExtractor(c)
			if err != nil {
				return nil, nil, err
			}
			req, err := g.getterReqBuilder(id, c)
			if err != nil {
				return nil, nil, err
			}
			idsHolder[index] = id
			rpcReqs = append(rpcReqs, req)
		}
	}

	return rpcReqs, idsHolder, nil
}

func (b *GenericConv[T, R, S, SR]) ChangeCustomMethods(rpcReqs []*models.RPCReq) (map[string]string, error) {
	changedMethods := make(map[string]string, len(rpcReqs))

	for _, rpcReq := range rpcReqs {
		regMethod, ok := b.customToRegular[rpcReq.Method]
		if !ok {
			continue
		}

		changedMethods[string(rpcReq.ID)] = rpcReq.Method
		rpcReq.Method = regMethod
	}

	return changedMethods, nil
}

func getGetterHolder(responses []*models.RPCResJSON, idsHolder map[string]string) (map[string]*models.RPCResJSON, []*models.RPCResJSON, error) {
	gHolder := make(map[string]*models.RPCResJSON, len(idsHolder))
	var responsesWithoutG []*models.RPCResJSON

	for _, res := range responses {
		var gMethod bool
		for content, id := range idsHolder {
			if string(res.ID) == id {
				gHolder[content] = res
				gMethod = true
			}
		}
		if !gMethod {
			responsesWithoutG = append(responsesWithoutG, res)
		}
	}

	return gHolder, responsesWithoutG, nil
}

func (b *GenericConv[T, R, S, SR]) ChangeCustomMethodsResponses(responses []*models.RPCResJSON, changedMethods, idsHolder map[string]string, cMethodsGetter map[string][]T) ([]*models.RPCResJSON, error) {
	gHolder, cleanRes, err := getGetterHolder(responses, idsHolder)
	if err != nil {
		return nil, err
	}

	for _, res := range cleanRes {
		originalMethod, ok := changedMethods[string(res.ID)]
		if !ok {
			continue
		}

		err := b.changeResultToGetterStruct(res, gHolder, cMethodsGetter, originalMethod)
		if err != nil {
			return nil, err
		}
	}

	return cleanRes, nil
}

func (g *GenericConv[T, R, S, SR]) getGetterReturn(res *models.RPCResJSON, gtHolder map[string]*models.RPCResJSON, cMethodsGetter map[string][]T) ([]R, *models.RPCErr, error) {
	gts := cMethodsGetter[string(res.ID)]

	var getterReturns []R
	var gtError *models.RPCErr
	for _, gt := range gts {
		index, err := g.idHolderIndexExtractor(gt)
		if err != nil {
			return nil, nil, err
		}

		gth := gtHolder[index]
		if gth.Error != nil {
			gtError = gth.Error
			break // if there was an error the rest of the response is invalid
		}

		gtr, err := g.getterReturnExtractor(gth.Result)
		if err != nil {
			return nil, nil, err
		}

		getterReturns = append(getterReturns, gtr)
	}

	return getterReturns, gtError, nil
}

func (b *GenericConv[T, R, S, SR]) changeResultToGetterStruct(res *models.RPCResJSON, gtHolder map[string]*models.RPCResJSON, cMethodsGetter map[string][]T, originalMethod string) error {
	if res.Error != nil {
		return nil // if there is an error the rest of the response is invalid
	}
	getterReturn, gtError, err := b.getGetterReturn(res, gtHolder, cMethodsGetter)
	if err != nil {
		return err
	}
	if gtError != nil {
		res.Result = nil // if there is an error the rest of the response is invalid
		res.Error = gtError
		return nil
	}

	if b.customMethodToIsRange[originalMethod] {
		from := getterReturn[0]
		to := getterReturn[0]
		if len(getterReturn) > 1 {
			to = getterReturn[1]
		}
		res.Result = b.getterRangeStructBuilder(res.Result, from, to)
	} else {
		res.Result = b.getterStructBuilder(res.Result, getterReturn[0])
	}

	return nil
}
