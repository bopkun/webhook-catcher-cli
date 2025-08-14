//go:build integration
// +build integration

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

func TestMain_InteractiveLocal(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"webhook-catcher-cli"}

	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.Write([]byte("1\n"))
	_ = w.Close()

	called := false
	origLocal := serveLocalFunc
	defer func() { serveLocalFunc = origLocal }()
	serveLocalFunc = func(addr string, mux http.Handler) error { called = true; return nil }

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	main()
	if !called {
		t.Fatalf("expected local server to be started via interactive choice 1")
	}
}

func TestMain_InteractiveTunnel(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"webhook-catcher-cli"}

	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.Write([]byte("2\n"))
	_ = w.Close()

	// ensure token available so run() won't prompt again
	_ = os.Setenv("NGROK_AUTHTOKEN", "tok")
	defer os.Unsetenv("NGROK_AUTHTOKEN")

	called := false
	origNgrok := serveNgrokFunc
	defer func() { serveNgrokFunc = origNgrok }()
	serveNgrokFunc = func(ctx context.Context, epOpts []config.HTTPEndpointOption, connectOpts []ngrok.ConnectOption, mux http.Handler) error {
		called = true
		return nil
	}

	main()
	if !called {
		t.Fatalf("expected ngrok tunnel to be started via interactive choice 2")
	}
}
