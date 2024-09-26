package main

import (
	"log"
	"net/http"

	"github.com/stateless-solutions/stateless-compatibility-layer/attestation"
	blocknumber "github.com/stateless-solutions/stateless-compatibility-layer/block-number"
	"github.com/stateless-solutions/stateless-compatibility-layer/environment"
	rpccontext "github.com/stateless-solutions/stateless-compatibility-layer/rpc-context"
	"golang.org/x/crypto/ssh"
)

var (
	defaultChainURL = environment.GetString("DEFAULT_CHAIN_URL", "")
	useAttestation  = environment.GetBool("USE_ATTESTION", false)
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
		if keyFile == "" {
			panic("KEY_FILE env must be set if USE_ATTESTION is true")
		}
		if identity == "" {
			panic("IDENTITY env must be set if USE_ATTESTION is true")
		}

		var signer ssh.Signer
		var err error
		if keyFilePassword != "" {
			signer, err = attestation.GetSigningKeyFromKeyFileWithPassphrase(keyFile, keyFilePassword)
			if err != nil {
				panic(err)
			}
		} else {
			signer, err = attestation.GetSigningKeyFromKeyFile(keyFile)
			if err != nil {
				panic(err)
			}
		}

		context.SigningKey = signer
		context.UseAttestion = true
	}

	// Start the server on the specified port
	http.HandleFunc("/", context.Handler)
	log.Printf("Starting server on :%s...\n", context.HTTPPort)
	log.Fatal(http.ListenAndServe(":"+context.HTTPPort, nil))
}
