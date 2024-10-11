package customrpcmethods

import (
	"encoding/json"
	"testing"

	"github.com/stateless-solutions/stateless-compatibility-layer/models"
)

func TestContext(t *testing.T) {
	tests := []struct {
		name              string
		req               []*models.RPCReq
		expectedReq       *models.RPCReq
		expectedReqLength int
		res               []*models.RPCResJSON
		idsToRewrite      []string // needed to be able to assest bc in the real code the id is random
		contentsToRewrite []string
		expectedRes       *models.RPCResJSON
		expectedErr       error
	}{
		{
			name: "Block height processed tag",
			req: []*models.RPCReq{{
				Method: "getBlockHeightAndContext",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`[{"commitment":"processed"}]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "getBlockHeight",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: 211,
			}, {ID: json.RawMessage("22"),
				Result: 2.1}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: contextResult{
					Value: 211,
					Context: context{
						Slot: 21,
					},
				},
			},
			contentsToRewrite: []string{"processed"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "No context method",
			req: []*models.RPCReq{{
				Method: "getBlockHeight",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`[{"commitment":"processed"}]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "getBlockHeight",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 1,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: 1,
			}},
			expectedRes: &models.RPCResJSON{
				ID:     json.RawMessage("21"),
				Result: 1,
			},
		},
		{
			name: "Context response not float",
			req: []*models.RPCReq{{
				Method: "getBlockHeightAndContext",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`[{"commitment":"processed"}]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "getBlockHeight",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa",
			}, {ID: json.RawMessage("22"),
				Result: "a"}},
			contentsToRewrite: []string{"processed"},
			idsToRewrite:      []string{"22"},
			expectedErr:       ErrInternalSlotResultNotFloat,
		},
		{
			name: "Error on data request",
			req: []*models.RPCReq{{
				Method: "getBlockHeightAndContext",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`[{"commitment":"processed"}]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "getBlockHeight",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID: json.RawMessage("21"),
				Error: &models.RPCErr{
					Code:    21,
					Message: "block height not good",
				},
			}, {ID: json.RawMessage("22"),
				Result: 211}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Error: &models.RPCErr{
					Code:    21,
					Message: "block height not good",
				},
			},
			contentsToRewrite: []string{"latest"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Error on context request",
			req: []*models.RPCReq{{
				Method: "getBlockHeightAndContext",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`[{"commitment":"processed"}]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "getBlockHeight",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa",
			}, {ID: json.RawMessage("22"),
				Error: &models.RPCErr{
					Code:    21,
					Message: "slot number not good",
				}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Error: &models.RPCErr{
					Code:    21,
					Message: "slot number not good",
				},
			},
			contentsToRewrite: []string{"processed"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Block height and no param",
			req: []*models.RPCReq{{
				Method: "getBlockHeightAndContext",
				ID:     json.RawMessage("21"),
			}},
			expectedReq: &models.RPCReq{
				Method: "getBlockHeight",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: 211,
			}, {ID: json.RawMessage("22"),
				Result: 2.1}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: contextResult{
					Value: 211,
					Context: context{
						Slot: 21,
					},
				},
			},
			contentsToRewrite: []string{"finalized"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Block height and no commitment param",
			req: []*models.RPCReq{{
				Method: "getBlockHeightAndContext",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`[{"not_a_commitment":"processed"}]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "getBlockHeight",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: 211,
			}, {ID: json.RawMessage("22"),
				Result: 2.1}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: contextResult{
					Value: 211,
					Context: context{
						Slot: 21,
					},
				},
			},
			contentsToRewrite: []string{"finalized"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Get block and shorter than commitment pos param",
			req: []*models.RPCReq{{
				Method: "getBlockAndContext",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`[21]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "getBlock",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: 211,
			}, {ID: json.RawMessage("22"),
				Result: 2.1}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: contextResult{
					Value: 211,
					Context: context{
						Slot: 21,
					},
				},
			},
			contentsToRewrite: []string{"finalized"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Block height and string param",
			req: []*models.RPCReq{{
				Method: "getBlockHeightAndContext",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["json"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "getBlockHeight",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: 211,
			}, {ID: json.RawMessage("22"),
				Result: 2.1}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: contextResult{
					Value: 211,
					Context: context{
						Slot: 21,
					},
				},
			},
			contentsToRewrite: []string{"finalized"},
			idsToRewrite:      []string{"22"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := NewCustomMethodHolder("../supported-chains/solana.json")

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
