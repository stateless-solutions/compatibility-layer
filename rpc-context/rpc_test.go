package rpccontext

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	customrpcmethods "github.com/stateless-solutions/stateless-compatibility-layer/custom-rpc-methods"
)

func TestAttestorHandler(t *testing.T) {
	tests := []struct {
		name                 string
		handler              http.HandlerFunc
		keyFile              string
		reqBody              string
		expectedCode         int
		expectedBody         string
		expectedPanicMessage string
		setChainURLInHeader  bool
		useAttestation       bool
	}{
		{
			name: "Success Case One Request",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jsonrpc":"2.0","result": "success", "id": "1"}`))
			}),
			keyFile:        "test-data/.mock_key.pem",
			useAttestation: true,
			reqBody:        `{"jsonrpc":"2.0","method":"eth_getBalance","id":1,"params":["0x20f33ce90a13a4b5e7697e3544c3083b8f8a51d4", "latest"]}`,
			expectedCode:   http.StatusOK,
			expectedBody:   `{"jsonrpc":"2.0","id":"1","result":"success","attestation":{"signatureFormat":"ssh-rsa","hashAlgo":"sha256","identity":"mock_identity","msg":"68e7a69974a641064a6a5ae8b1a00997939a325ec585a49e9fe82b386a21726a","signature":"8e71bb7db5b3b719e12a36219c05308dff130740f2714ed3bbf04f69cb6e95a691792cc7492c14c13c74b76cdbc939169b1f6ba53ff4f82b8c89875d9f49b8db6d83ef4924f18931e975bd27de9e6e734ed5c930330f14c2f36e6002b577a37de27adf57a4b17bcee8816d757c989f5119807c4cd85212712eecc042dc6e917a"}}`,
		},
		{
			name: "Success Case One Request No Attestation",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jsonrpc":"2.0","result": "success", "id": "1"}`))
			}),
			keyFile:      "test-data/.mock_key.pem",
			reqBody:      `{"jsonrpc":"2.0","method":"eth_getBalance","id":1,"params":["0x20f33ce90a13a4b5e7697e3544c3083b8f8a51d4", "latest"]}`,
			expectedCode: http.StatusOK,
			expectedBody: `{"jsonrpc":"2.0","result":"success","id":"1"}`,
		},
		{
			name: "Success Case One Request set Chain URL in Header",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jsonrpc":"2.0","result": "success", "id": "1"}`))
			}),
			setChainURLInHeader: true,
			keyFile:             "test-data/.mock_key.pem",
			useAttestation:      true,
			reqBody:             `{"jsonrpc":"2.0","method":"eth_getBalance","id":1,"params":["0x20f33ce90a13a4b5e7697e3544c3083b8f8a51d4", "latest"]}`,
			expectedCode:        http.StatusOK,
			expectedBody:        `{"jsonrpc":"2.0","id":"1","result":"success","attestation":{"signatureFormat":"ssh-rsa","hashAlgo":"sha256","identity":"mock_identity","msg":"68e7a69974a641064a6a5ae8b1a00997939a325ec585a49e9fe82b386a21726a","signature":"8e71bb7db5b3b719e12a36219c05308dff130740f2714ed3bbf04f69cb6e95a691792cc7492c14c13c74b76cdbc939169b1f6ba53ff4f82b8c89875d9f49b8db6d83ef4924f18931e975bd27de9e6e734ed5c930330f14c2f36e6002b577a37de27adf57a4b17bcee8816d757c989f5119807c4cd85212712eecc042dc6e917a"}}`,
		},
		{
			name: "Success Case Batch Request",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[{"jsonrpc":"2.0","result": "success","id": "1"},{"jsonrpc":"2.0","result": "success","id": "2"}]`))
			}),
			keyFile:        "test-data/.mock_key.pem",
			useAttestation: true,
			reqBody:        `[{"jsonrpc":"2.0","method":"eth_getBalance","id":1,"params":["0x20f33ce90a13a4b5e7697e3544c3083b8f8a51d4", "latest"]},{"jsonrpc":"2.0","method":"eth_getBalance","id":2,"params":["0x20f33ce90a13a4b5e7697e3544c3083b8f8a51d4", "latest"]}]`,
			expectedCode:   http.StatusOK,
			expectedBody:   `[{"jsonrpc":"2.0","id":"1","result":"success","attestation":{"signatureFormat":"ssh-rsa","hashAlgo":"sha256","identity":"mock_identity","msg":"68e7a69974a641064a6a5ae8b1a00997939a325ec585a49e9fe82b386a21726a","signature":"8e71bb7db5b3b719e12a36219c05308dff130740f2714ed3bbf04f69cb6e95a691792cc7492c14c13c74b76cdbc939169b1f6ba53ff4f82b8c89875d9f49b8db6d83ef4924f18931e975bd27de9e6e734ed5c930330f14c2f36e6002b577a37de27adf57a4b17bcee8816d757c989f5119807c4cd85212712eecc042dc6e917a"}},{"jsonrpc":"2.0","id":"2","result":"success","attestation":{"msg":"68e7a69974a641064a6a5ae8b1a00997939a325ec585a49e9fe82b386a21726a","signature":"8e71bb7db5b3b719e12a36219c05308dff130740f2714ed3bbf04f69cb6e95a691792cc7492c14c13c74b76cdbc939169b1f6ba53ff4f82b8c89875d9f49b8db6d83ef4924f18931e975bd27de9e6e734ed5c930330f14c2f36e6002b577a37de27adf57a4b17bcee8816d757c989f5119807c4cd85212712eecc042dc6e917a"}}]`,
		},
		{
			name: "Failure Case Invalid Req Body",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jsonrpc":"2.0","method":"eth_getBalance","id":1,"params":["0x20f33ce90a13a4b5e7697e3544c3083b8f8a51d4", "latest"]}`))
			}),
			keyFile:        "test-data/.mock_key.pem",
			useAttestation: true,
			reqBody:        `{"jsonrpc": 1}`,
			expectedCode:   http.StatusBadRequest,
			expectedBody:   "Invalid request format\n",
		},
		{
			name: "Failure Case Invalid Res Body",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jsonrpc": 1}`))
			}),
			keyFile:        "test-data/.mock_key.pem",
			useAttestation: true,
			reqBody:        `{"jsonrpc":"2.0","method":"eth_getBalance","id":1,"params":["0x20f33ce90a13a4b5e7697e3544c3083b8f8a51d4", "latest"]}`,
			expectedCode:   http.StatusBadRequest,
			expectedBody:   "Invalid response format\n",
		},
	}

	// Mock identity and password
	identity := "mock_identity"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock server
			mockServer := httptest.NewServer(tt.handler)
			defer mockServer.Close()

			// Mock request
			body, _ := json.Marshal(json.RawMessage(tt.reqBody))
			req, err := http.NewRequest("POST", "/", bytes.NewBuffer(body))
			if err != nil {
				t.Fatalf("Error creating mock request: %v", err)
			}

			// Mock response recorder
			rec := httptest.NewRecorder()

			ch := customrpcmethods.NewCustomMethodHolder("../supported-chains/ethereum.json")

			context := &RPCContext{
				Identity:           identity,
				CustomMethodHolder: ch,
			}

			if tt.useAttestation {
				context.EnableAttestation(tt.keyFile, "", identity)
			}

			if tt.setChainURLInHeader {
				req.Header.Set("Stateless-Chain-URL", mockServer.URL)
			} else {
				context.DefaultChainURL = mockServer.URL
			}

			// Create a handler using AttestorHandler
			handler := http.HandlerFunc(context.Handler)

			// Serve the request using the protected handler
			handler.ServeHTTP(rec, req)

			// Check the response code
			if rec.Code != tt.expectedCode {
				t.Errorf("Test case %s: Expected status code %d, got %d", tt.name, tt.expectedCode, rec.Code)
			}

			// Compare JSON bodies
			if rec.Body.String() != tt.expectedBody {
				t.Errorf("Test case %s: Expected body %s, got %s", tt.name, tt.expectedBody, rec.Body)
			}
		})
	}
}
