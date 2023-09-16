package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/MadAppGang/httplog"
	"github.com/marvinruder/hetzner-dyndns/internal/handler"
)

func main() {
	if os.Getenv("COLOR") == "true" {
		httplog.ForceConsoleColor()
	}
	if os.Getenv("COLOR") == "false" {
		httplog.DisableConsoleColor()
	}
	http.Handle("/nic/update", httplog.LoggerWithConfig(httplog.LoggerConfig{
		CaptureBody:    true,
		Formatter:      httplog.FullFormatterWithRequestAndResponseHeadersAndBody,
		HideHeaderKeys: []string{"Authorization"},
		ProxyHandler:   httplog.NewProxy(),
		RouterName:     "DynDns Request",
	})(http.Handler(&handler.DynDnsHandler{DynDnsRequest: handler.DynDnsRequest})))
	http.Handle("/", httplog.LoggerWithConfig(httplog.LoggerConfig{
		CaptureBody:  true,
		Formatter:    httplog.FullFormatterWithRequestAndResponseHeadersAndBody,
		ProxyHandler: httplog.NewProxy(),
		RouterName:   "Other Request",
	})(http.NotFoundHandler()))

	if os.Getenv("ZONE") == "" {
		fmt.Println("Zone not configured, will accept requests for all zones")
	} else {
		fmt.Println("Will accept requests for zone " + os.Getenv("ZONE"))
	}
	if os.Getenv("TOKEN") == "" {
		fmt.Println("Token not configured, will accept requests with any token")
	} else {
		fmt.Println("Will accept requests with correct token only")
	}

	fmt.Println("Starting server on port 8245")
	err := http.ListenAndServe(":8245", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Println("Server closed")
	} else if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
		os.Exit(1)
	}
}
