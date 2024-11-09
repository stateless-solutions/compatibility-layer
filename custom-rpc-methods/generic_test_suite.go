package customrpcmethods

import (
	"testing"

	"github.com/stateless-solutions/compatibility-layer/models"
)

type testCaseGenericConv struct {
	name              string
	req               []*models.RPCReq
	expectedReq       *models.RPCReq
	expectedReqLength int
	gatewayMode       bool
	res               []*models.RPCResJSON
	idsToRewrite      []string // needed to be able to assest bc in the real code the id is random
	contentsToRewrite []string
	expectedRes       *models.RPCResJSON
	expectedErr       error
}

func runTests(t *testing.T, configFile string, tests []testCaseGenericConv) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			ch := NewCustomMethodHolder(tt.gatewayMode, configFile)

			tt.req, err = ch.HandleGatewayMode(tt.req)
			if err != nil {
				t.Fatalf("Test case %s: Error not expected, got %v", tt.name, err)
			}

			context, err := ch.GetCustomMethodsMap(tt.req)
			if err != nil {
				t.Fatalf("Test case %s: Error not expected, got %v", tt.name, err)
			}
			changedMethods, _ := ch.ChangeCustomMethods(tt.req)

			if tt.req[0].Method != tt.expectedReq.Method {
				t.Errorf("Test case %s: Expected method %s, got %s", tt.name, tt.expectedReq.Method, tt.req[0].Method)
			}

			var idsHolder map[string]string
			tt.req, idsHolder, _ = ch.AddGetterMethodsIfNeeded(tt.req, context)

			if tt.idsToRewrite != nil && tt.contentsToRewrite != nil {
				for i := 0; i < len(tt.idsToRewrite); i++ {
					idsHolder[tt.contentsToRewrite[i]] = tt.idsToRewrite[i]
				}
			}

			if len(tt.req) != tt.expectedReqLength {
				t.Errorf("Test case %s: Expected req length %d, got %d", tt.name, tt.expectedReqLength, len(tt.req))
			}

			ress, err := ch.ChangeCustomMethodsResponses(tt.res, changedMethods, idsHolder, context)
			if err != tt.expectedErr {
				t.Fatalf("Test case %s: Expected error %v, got %v", tt.name, tt.expectedErr, err)
			}

			if err == nil {
				if tt.expectedRes.Error != nil {
					if ress[0].Error.Message != tt.expectedRes.Error.Message {
						t.Errorf("Test case %s: Expected rpc error %s, got %s", tt.name, tt.expectedRes.Error.Message, tt.res[0].Error.Message)
					}
				}

				if ress[0].Result != tt.expectedRes.Result {
					t.Errorf("Test case %s: Expected response %s, got %s", tt.name, tt.expectedRes.Result, tt.res[0].Result)
				}
			}
		})
	}
}
