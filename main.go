package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	customrpcmethods "github.com/stateless-solutions/stateless-compatibility-layer/custom-rpc-methods"
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
	logLevel        = slog.Level(environment.GetInt64("LOG_LEVEL", int64(slog.LevelInfo)))
	gatewayMode     = environment.GetBool("GATEWAY_MODE", false)
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	ch := customrpcmethods.NewCustomMethodHolder(gatewayMode, configFiles)

	rpcContext := &rpccontext.RPCContext{
		Identity:           identity,
		DefaultChainURL:    defaultChainURL,
		HTTPPort:           httpPort,
		CustomMethodHolder: ch,
		Logger:             logger,
	}

	if useAttestation {
		rpcContext.EnableAttestation(keyFile, keyFilePassword, identity)
	}

	srv := &http.Server{
		Addr:    ":" + rpcContext.HTTPPort,
		Handler: http.DefaultServeMux,
	}

	// Start the server on the specified port
	http.HandleFunc("/rpc", rpcContext.Handler)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	go func() {
		logger.Info(fmt.Sprintf("Starting server on :%s", rpcContext.HTTPPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Listen and serve failed", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit

	logger.Info("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt to gracefully shutdown the server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", slog.String("error", err.Error()))
	}

	logger.Info("Server exiting")
}
