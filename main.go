package main

import (
	"log"
	"net/http"

	blocknumber "github.com/stateless-solutions/stateless-compatibility-layer/block-number"
	"github.com/stateless-solutions/stateless-compatibility-layer/environment"
	rpccontext "github.com/stateless-solutions/stateless-compatibility-layer/rpc-context"
)

var (
	defaultChainURL = environment.GetString("DEFAULT_CHAIN_URL", "")
	useAttestation  = environment.GetBool("USE_ATTESTATION", false)
	keyFile         = environment.GetString("KEY_FILE", "")
	keyFilePassword = environment.GetString("KEY_FILE_PASSWORD", "")
	identity        = environment.GetString("IDENTITY", "")
	httpPort        = environment.GetString("HTTP_PORT", "8080")
	configFiles     = environment.GetString("CONFIG_FILES", "supported-chains/ethereum.json")
)

func main() {
	bn := blocknumber.NewBlockNumberConv(configFiles)

	context := &rpccontext.RPCContext{
		Identity:        identity,
		DefaultChainURL: defaultChainURL,
		HTTPPort:        httpPort,
		BlockNumberConv: bn,
	}

	if useAttestation {
		context.EnableAttestation(keyFile, keyFilePassword, identity)
	}

	// Start the server on the specified port
	http.HandleFunc("/", context.Handler)
	log.Printf("Starting server on :%s...\n", context.HTTPPort)
	log.Fatal(http.ListenAndServe(":"+context.HTTPPort, nil))
}
