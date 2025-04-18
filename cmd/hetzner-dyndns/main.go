package main

import (
	"errors"
	"net/http"
	"os"

	"github.com/MadAppGang/httplog"
	"github.com/marvinruder/hetzner-dyndns/internal/handler"
	"github.com/marvinruder/hetzner-dyndns/internal/logger"
)

func main() {
	if os.Getenv("COLOR") == "true" {
		httplog.ForceConsoleColor()
	}
	if os.Getenv("COLOR") == "false" {
		httplog.DisableConsoleColor()
	}

	http.Handle("/nic/update", httplog.LoggerWithConfig(httplog.LoggerConfig{
		CaptureBody: true,
		Formatter: httplog.ChainLogFormatter(
			httplog.DefaultLogFormatter,
			httplog.RequestHeaderLogFormatter,
			httplog.ResponseBodyLogFormatter,
		),
		HideHeaderKeys: []string{"Authorization"},
		ProxyHandler:   httplog.NewProxy(),
		RouterName:     "DynDns Request",
	})(http.Handler(&handler.DynDnsHandler{DynDnsRequest: handler.DynDnsRequest})))
	http.Handle("/", httplog.LoggerWithConfig(httplog.LoggerConfig{
		Formatter:    httplog.DefaultLogFormatter,
		ProxyHandler: httplog.NewProxy(),
		RouterName:   "Other Request",
	})(http.NotFoundHandler()))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	if os.Getenv("ZONE") == "" {
		logger.Info("Zone not configured, will accept requests for all zones")
	} else {
		logger.Info("Will accept requests for configured zone only", "zone", os.Getenv("ZONE"))
	}
	if os.Getenv("TOKEN") == "" {
		logger.Info("Token not configured, will accept requests with any token")
	} else {
		logger.Info("Will accept requests with correct token only")
	}

	logger.Info("Starting server", "port", 8245)
	err := http.ListenAndServe(":8245", nil)
	if errors.Is(err, http.ErrServerClosed) {
		logger.Info("Server closed")
	} else if err != nil {
		logger.Error("Error starting server", "error", err)
		os.Exit(1)
	}
}
