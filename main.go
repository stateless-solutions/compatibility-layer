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
	chainURL        = environment.MustGetString("CHAIN_URL")
	keyFile         = environment.MustGetString("KEY_FILE")
	keyFilePassword = environment.GetString("KEY_FILE_PASSWORD", "")
	identity        = environment.MustGetString("IDENTITY")
	httpPort        = environment.GetString("HTTP_PORT", "8080")
	configFile      = environment.GetString("CONFIG_FILE", "supported-chains/ethereum.json")
)

func main() {
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

	bn := blocknumber.NewBlockNumberConv(configFile)

	context := &rpccontext.RPCContext{
		SigningKey:      signer,
		Identity:        identity,
		ChainURL:        chainURL,
		HTTPPort:        httpPort,
		BlockNumberConv: bn,
	}

	// Start the server on the specified port
	http.HandleFunc("/", context.Handler)
	log.Printf("Starting server on :%s...\n", context.HTTPPort)
	log.Fatal(http.ListenAndServe(":"+context.HTTPPort, nil))
}
