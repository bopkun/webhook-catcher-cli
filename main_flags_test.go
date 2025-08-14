package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"testing"

	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

func TestMain_Flags_Local_NoPrompt(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"webhook-catcher-cli", "-host", "127.0.0.1", "-port", "8081"}

	called := false
	origLocal := serveLocalFunc
	defer func() { serveLocalFunc = origLocal }()
	serveLocalFunc = func(addr string, mux http.Handler) error {
		called = true
		return nil
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	main()
	if !called {
		t.Fatalf("expected local server to be started when flags provided")
	}
}

func TestMain_Flags_Tunnel_NoPrompt(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"webhook-catcher-cli", "-tunnel", "true", "-ngrok-authtoken", "tok", "-ngrok-region", "us", "-ngrok-domain", "x.ngrok.app"}

	called := false
	origNgrok := serveNgrokFunc
	defer func() { serveNgrokFunc = origNgrok }()
	serveNgrokFunc = func(ctx context.Context, epOpts []config.HTTPEndpointOption, connectOpts []ngrok.ConnectOption, mux http.Handler) error {
		called = true
		return nil
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	main()
	if !called {
		t.Fatalf("expected ngrok to be started when tunnel flags provided")
	}
}
