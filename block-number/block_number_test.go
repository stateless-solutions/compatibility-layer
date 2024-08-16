package blocknumber

import (
	"encoding/json"
	"testing"

	"github.com/stateless-solutions/stateless-compatibility-layer/models"
)

func TestBlockNumber(t *testing.T) {
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
			name: "Call and block number latest tag",
			req: []*models.RPCReq{{
				Method: "eth_callAndBlockNumber",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["","latest"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_call",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa",
			}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"number": "0x21"}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockNumberResult{
					Data:        "aaa",
					BlockNumber: "0x21",
				},
			},
			contentsToRewrite: []string{"latest"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Balance and block number input block number",
			req: []*models.RPCReq{{
				Method: "eth_getBalanceAndBlockNumber",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["","0x23"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getBalance",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 1,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "1",
			}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockNumberResult{
					Data:        "1",
					BlockNumber: "0x23",
				},
			},
		},
		{
			name: "Storage at and block number input block hash",
			req: []*models.RPCReq{{
				Method: "eth_getStorageAtAndBlockNumber",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["","",{"blockHash":"0x3f07a9c83155594c000642e7d60e8a8a00038d03e9849171a05ed0e2d47acbb3"}]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getStorageAt",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa"}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"number": "0x21"}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockNumberResult{
					Data:        "aaa",
					BlockNumber: "0x21",
				},
			},
			contentsToRewrite: []string{"0x3f07a9c83155594c000642e7d60e8a8a00038d03e9849171a05ed0e2d47acbb3"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Transaction count and block number pending tag",
			req: []*models.RPCReq{{
				Method: "eth_getTransactionCountAndBlockNumber",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["","pending"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getTransactionCount",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "1",
			}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"number": "0x21"}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockNumberResult{
					Data:        "1",
					BlockNumber: "0x21",
				},
			},
			contentsToRewrite: []string{"pending"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Code and block number earliest tag",
			req: []*models.RPCReq{{
				Method: "eth_getCodeAndBlockNumber",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["","earliest"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getCode",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "1",
			}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"number": "0x21"}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockNumberResult{
					Data:        "1",
					BlockNumber: "0x21",
				},
			},
			contentsToRewrite: []string{"earliest"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Code and block number finalized tag",
			req: []*models.RPCReq{{
				Method: "eth_getCodeAndBlockNumber",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["","finalized"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getCode",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "1",
			}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"number": "0x21"}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockNumberResult{
					Data:        "1",
					BlockNumber: "0x21",
				},
			},
			contentsToRewrite: []string{"finalized"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Code and block number safe tag",
			req: []*models.RPCReq{{
				Method: "eth_getCodeAndBlockNumber",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["","safe"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getCode",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "1",
			}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"number": "0x21"}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockNumberResult{
					Data:        "1",
					BlockNumber: "0x21",
				},
			},
			contentsToRewrite: []string{"safe"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "No block number method",
			req: []*models.RPCReq{{
				Method: "eth_getCode",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["","0x23"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getCode",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 1,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "1",
			}},
			expectedRes: &models.RPCResJSON{
				ID:     json.RawMessage("21"),
				Result: "1",
			},
		},
		{
			name: "Block number response not map",
			req: []*models.RPCReq{{
				Method: "eth_callAndBlockNumber",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["","latest"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_call",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa",
			}, {ID: json.RawMessage("22"),
				Result: "a"}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockNumberResult{
					Data:        "aaa",
					BlockNumber: "0x21",
				},
			},
			contentsToRewrite: []string{"latest"},
			idsToRewrite:      []string{"22"},
			expectedErr:       ErrInternalBlockNumberMethodNotMap,
		},
		{
			name: "Call and block number latest tag",
			req: []*models.RPCReq{{
				Method: "eth_callAndBlockNumber",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["","latest"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_call",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa",
			}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"block": "0x21"}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockNumberResult{
					Data:        "aaa",
					BlockNumber: "0x21",
				},
			},
			contentsToRewrite: []string{"latest"},
			idsToRewrite:      []string{"22"},
			expectedErr:       ErrInternalBlockNumberMethodNotNumberEntry,
		},
		{
			name: "Block transaction count and block number by number latest tag",
			req: []*models.RPCReq{{
				Method: "eth_getBlockTransactionCountAndBlockNumberByNumber",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["latest"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getBlockTransactionCountByNumber",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa",
			}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"number": "0x21"}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockNumberResult{
					Data:        "aaa",
					BlockNumber: "0x21",
				},
			},
			contentsToRewrite: []string{"latest"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Raw transaction and block number by number and index latest tag",
			req: []*models.RPCReq{{
				Method: "eth_getRawTransactionAndBlockNumberByBlockNumberAndIndex",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["latest"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getRawTransactionByBlockNumberAndIndex",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa",
			}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"number": "0x21"}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockNumberResult{
					Data:        "aaa",
					BlockNumber: "0x21",
				},
			},
			contentsToRewrite: []string{"latest"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Uncle count and block number by number latest tag",
			req: []*models.RPCReq{{
				Method: "eth_getUncleCountAndBlockNumberByBlockNumber",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["latest"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getUncleCountByBlockNumber",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa",
			}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"number": "0x21"}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockNumberResult{
					Data:        "aaa",
					BlockNumber: "0x21",
				},
			},
			contentsToRewrite: []string{"latest"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Log and block range no input",
			req: []*models.RPCReq{{
				Method: "eth_getLogsAndBlockRange",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`[{}]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getLogs",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 3,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa",
			}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"number": "0x21"}},
				{ID: json.RawMessage("23"),
					Result: map[string]interface{}{"number": "0x22"}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockRangeResult{
					Data:          "aaa",
					StartingBlock: "0x21",
					EndingBlock:   "0x22",
				},
			},
			contentsToRewrite: []string{"earliest", "latest"},
			idsToRewrite:      []string{"22", "23"},
		},
		{
			name: "Log and block range block number input",
			req: []*models.RPCReq{{
				Method: "eth_getLogsAndBlockRange",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`[{"fromBlock": "0x21", "toBlock": "0x22"}]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getLogs",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 1,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa",
			}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockRangeResult{
					Data:          "aaa",
					StartingBlock: "0x21",
					EndingBlock:   "0x22",
				},
			},
		},
		{
			name: "Log and block range block hash input",
			req: []*models.RPCReq{{
				Method: "eth_getLogsAndBlockRange",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`[{"blockHash":"0x3f07a9c83155594c000642e7d60e8a8a00038d03e9849171a05ed0e2d47acbb3"}]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getLogs",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa",
			}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"number": "0x21"}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockRangeResult{
					Data:          "aaa",
					StartingBlock: "0x21",
					EndingBlock:   "0x21",
				},
			},
			contentsToRewrite: []string{"0x3f07a9c83155594c000642e7d60e8a8a00038d03e9849171a05ed0e2d47acbb3"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Log and block range tag input",
			req: []*models.RPCReq{{
				Method: "eth_getLogsAndBlockRange",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`[{"fromBlock": "safe", "toBlock": "pending"}]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getLogs",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 3,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa",
			}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"number": "0x21"}},
				{ID: json.RawMessage("23"),
					Result: map[string]interface{}{"number": "0x22"}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Result: blockRangeResult{
					Data:          "aaa",
					StartingBlock: "0x21",
					EndingBlock:   "0x22",
				},
			},
			contentsToRewrite: []string{"safe", "pending"},
			idsToRewrite:      []string{"22", "23"},
		},
		{
			name: "Error on data request",
			req: []*models.RPCReq{{
				Method: "eth_callAndBlockNumber",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["","latest"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_call",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID: json.RawMessage("21"),
				Error: &models.RPCErr{
					Code:    21,
					Message: "call not good",
				},
			}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"number": "0x21"}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Error: &models.RPCErr{
					Code:    21,
					Message: "call not good",
				},
			},
			contentsToRewrite: []string{"latest"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Error on number request",
			req: []*models.RPCReq{{
				Method: "eth_callAndBlockNumber",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["","latest"]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_call",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa",
			}, {ID: json.RawMessage("22"),
				Error: &models.RPCErr{
					Code:    21,
					Message: "block number not good",
				}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Error: &models.RPCErr{
					Code:    21,
					Message: "block number not good",
				},
			},
			contentsToRewrite: []string{"latest"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Error on number with hash request",
			req: []*models.RPCReq{{
				Method: "eth_getStorageAtAndBlockNumber",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`["","",{"blockHash":"0x3f07a9c83155594c000642e7d60e8a8a00038d03e9849171a05ed0e2d47acbb3"}]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getStorageAt",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 2,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa"}, {ID: json.RawMessage("22"),
				Error: &models.RPCErr{
					Code:    21,
					Message: "block number not good",
				}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Error: &models.RPCErr{
					Code:    21,
					Message: "block number not good",
				},
			},
			contentsToRewrite: []string{"0x3f07a9c83155594c000642e7d60e8a8a00038d03e9849171a05ed0e2d47acbb3"},
			idsToRewrite:      []string{"22"},
		},
		{
			name: "Error on block number request with block range",
			req: []*models.RPCReq{{
				Method: "eth_getLogsAndBlockRange",
				ID:     json.RawMessage("21"),
				Params: json.RawMessage(`[{}]`),
			}},
			expectedReq: &models.RPCReq{
				Method: "eth_getLogs",
				ID:     json.RawMessage("21"),
			},
			expectedReqLength: 3,
			res: []*models.RPCResJSON{{
				ID:     json.RawMessage("21"),
				Result: "aaa",
			}, {ID: json.RawMessage("22"),
				Result: map[string]interface{}{"number": "0x21"}},
				{ID: json.RawMessage("23"),
					Error: &models.RPCErr{
						Code:    21,
						Message: "block number not good",
					}}},
			expectedRes: &models.RPCResJSON{
				ID: json.RawMessage("21"),
				Error: &models.RPCErr{
					Code:    21,
					Message: "block number not good",
				},
			},
			contentsToRewrite: []string{"earliest", "latest"},
			idsToRewrite:      []string{"22", "23"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBlockNumberConv("../supported-chains/ethereum.json")

			blockMap, err := b.GetBlockNumberMap(tt.req)
			if err != nil {
				t.Fatalf("Test case %s: Error not expected, got %v", tt.name, err)
			}
			changedMethods := b.ChangeBlockNumberMethods(tt.req)

			if tt.req[0].Method != tt.expectedReq.Method {
				t.Errorf("Test case %s: Expected method %s, got %s", tt.name, tt.expectedReq.Method, tt.req[0].Method)
			}

			var idsHolder map[string]string
			tt.req, idsHolder, _ = AddBlockNumberMethodsIfNeeded(tt.req, blockMap)

			if tt.idsToRewrite != nil && tt.contentsToRewrite != nil {
				for i := 0; i < len(tt.idsToRewrite); i++ {
					idsHolder[tt.contentsToRewrite[i]] = tt.idsToRewrite[i]
				}
			}

			if len(tt.req) != tt.expectedReqLength {
				t.Errorf("Test case %s: Expected req length %d, got %d", tt.name, tt.expectedReqLength, len(tt.req))
			}

			ress, err := b.ChangeBlockNumberResponses(tt.res, changedMethods, idsHolder, blockMap)
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
