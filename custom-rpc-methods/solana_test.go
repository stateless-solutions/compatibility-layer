package customrpcmethods

import (
	"encoding/json"
	"testing"

	"github.com/stateless-solutions/compatibility-layer/models"
)

func TestSolana(t *testing.T) {
	tests := []testCaseGenericConv{
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
			expectedErr:       ErrInternalSlotResultNotExpectedType,
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

	runTests(t, "../supported-chains/solana.json", tests)
}
