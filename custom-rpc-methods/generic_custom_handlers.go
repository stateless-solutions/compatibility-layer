package customrpcmethods

import (
	"fmt"
	"reflect"

	"github.com/stateless-solutions/compatibility-layer/models"
)

// CustomHandlerHolder is a generic of structs that hold the methods for custom handlers
type CustomHandlerHolder interface {
	evmCustomHandlersHolder
}

// this validates if all custom handlers have the correct structure and saves unto a map
// reflect was used because it was the easiest form to iterate over the methods of a structure
// this must be called on init for chain types that suport custom handlers
func SaveCustomHandlersToMap[T GetterTypes, CH CustomHandlerHolder](chInst CH, chMap map[string]func(*models.RPCReq) ([]T, error)) {
	expectedType := reflect.TypeOf(func(CH, *models.RPCReq) ([]T, error) { return nil, nil })
	holderType := reflect.TypeOf(chInst)

	for i := 0; i < holderType.NumMethod(); i++ {
		method := holderType.Method(i)

		if method.Type != expectedType {
			panic(fmt.Sprintf("method %s has an incorrect signature: expected %v, got %v", method.Name, expectedType, method.Type))
		}

		handler := reflect.ValueOf(chInst).MethodByName(method.Name)
		handlerFunc := handler.Interface().(func(*models.RPCReq) ([]T, error))
		chMap[method.Name] = handlerFunc
	}
}
