package integrationtests

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stateless-solutions/stateless-compatibility-layer/attestation"
	blocknumber "github.com/stateless-solutions/stateless-compatibility-layer/block-number"
	"github.com/stateless-solutions/stateless-compatibility-layer/models"
	rpccontext "github.com/stateless-solutions/stateless-compatibility-layer/rpc-context"
)

var urlFlag string
var integration bool
var keyFile string
var configFileTest string
var integrationTestFile string

type IntegrationTestCases struct {
	Name    string        `json:"name"`
	ReqBody models.RPCReq `json:"reqBody"`
}

type IntegrationTestConfig struct {
	Cases []IntegrationTestCases `json:"cases"`
}

func init() {
	flag.StringVar(&urlFlag, "url", "", "The URL of the server to test against")
	flag.BoolVar(&integration, "integration", false, "Bool to run the integration tests")
	flag.StringVar(&keyFile, "keyfile", "", "Path of key file for attestions")
	flag.StringVar(&configFileTest, "configFile", "", "Path of config file")
	flag.StringVar(&integrationTestFile, "integrationFile", "", "Path of integration tests config file")
}

func TestMain(m *testing.M) {
	flag.Parse()
	if integration {
		if urlFlag == "" {
			panic("URL must be provided with the -url flag")
		}
		if keyFile == "" {
			panic("Keyfile must be provided with the -keyfile flag")
		}
		if configFileTest == "" {
			panic("Config file must be provided with the -configFile flag")
		}
		if integrationTestFile == "" {
			panic("Integration file must be provided with the -integrationFile flag")
		}
	}
	m.Run()
}

func TestIntegration(t *testing.T) {
	if !integration {
		t.Skip("just run unit test")
	}

	file, err := os.Open(integrationTestFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	var config IntegrationTestConfig
	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		panic(err)
	}

	tests := config.Cases

	// Mock identity
	identity := "mock_identity"

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Mock request
			body, _ := json.Marshal(tt.ReqBody)
			req, err := http.NewRequest("POST", "/", bytes.NewBuffer(body))
			if err != nil {
				t.Fatalf("Error creating mock request: %v", err)
			}

			// Mock response recorder
			rec := httptest.NewRecorder()

			// var signer ssh.Signer
			signer, err := attestation.GetSigningKeyFromKeyFile(keyFile)
			if err != nil {
				panic(err)
			}

			bn := blocknumber.NewBlockNumberConv(configFileTest)

			context := &rpccontext.RPCContext{
				SigningKey:      signer,
				Identity:        identity,
				ChainURL:        urlFlag,
				BlockNumberConv: bn,
			}

			// Create a handler using AttestorHandler
			handler := http.HandlerFunc(context.Handler)

			// Serve the request using the protected handler
			handler.ServeHTTP(rec, req)

			// Check the response code
			if rec.Code != http.StatusOK {
				t.Errorf("Test case %s: Expected status code %d, got %d", tt.Name, http.StatusOK, rec.Code)
			}

			var rpcRes models.RPCResJSON
			err = json.Unmarshal(rec.Body.Bytes(), &rpcRes)
			if err != nil {
				panic(err)
			}

			// Check for rpc error
			if rpcRes.Error != nil {
				t.Errorf("Test case %s: Expected no error, got %s", tt.Name, rpcRes.Error.Error())
			}
		})
	}
}