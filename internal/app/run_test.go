package app

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

func TestRun_Local_OK(t *testing.T) {
	orig := ServeLocalFunc
	defer func() { ServeLocalFunc = orig }()
	// stub server to succeed
	ServeLocalFunc = func(addr string, mux http.Handler) error { return nil }

	err := Run(Options{Host: "127.0.0.1", Port: 8080, Tunnel: false})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestRun_Local_Error(t *testing.T) {
	orig := ServeLocalFunc
	defer func() { ServeLocalFunc = orig }()
	ServeLocalFunc = func(addr string, mux http.Handler) error { return errors.New("boom") }

	err := Run(Options{Host: "127.0.0.1", Port: 8080, Tunnel: false})
	if err == nil || !strings.Contains(err.Error(), "server error") {
		t.Fatalf("expected wrapped server error, got %v", err)
	}
}

func TestRun_Ngrok_WithToken_OK(t *testing.T) {
	orig := ServeNgrokFunc
	defer func() { ServeNgrokFunc = orig }()
	called := false
	ServeNgrokFunc = func(ctx context.Context, epOpts []config.HTTPEndpointOption, connectOpts []ngrok.ConnectOption, mux http.Handler) error {
		called = true
		return nil
	}
	// Ensure env var is ignored when explicit token provided
	_ = os.Unsetenv("NGROK_AUTHTOKEN")

	err := Run(Options{Tunnel: true, NgrokToken: "abc", NgrokRegion: "us", NgrokDomain: "example.ngrok.app"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !called {
		t.Fatalf("expected serveNgrokFunc to be called")
	}
}

func TestRun_Ngrok_Prompt_SaveToken_OK(t *testing.T) {
	// run within temp dir so .env writes are isolated
	origWD, _ := os.Getwd()
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	defer func() { _ = os.Chdir(origWD) }()

	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.Write([]byte("tok123\n"))
	_ = w.Close()
	_ = os.Unsetenv("NGROK_AUTHTOKEN")

	orig := ServeNgrokFunc
	defer func() { ServeNgrokFunc = orig }()
	ServeNgrokFunc = func(ctx context.Context, epOpts []config.HTTPEndpointOption, connectOpts []ngrok.ConnectOption, mux http.Handler) error {
		return nil
	}

	err := Run(Options{Tunnel: true, NgrokToken: ""})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// token should be set in env and saved to .env
	if os.Getenv("NGROK_AUTHTOKEN") != "tok123" {
		t.Fatalf("expected NGROK_AUTHTOKEN=tok123 in env, got %q", os.Getenv("NGROK_AUTHTOKEN"))
	}
	b, err2 := os.ReadFile(filepath.Join(tmp, ".env"))
	if err2 != nil || !strings.Contains(string(b), "NGROK_AUTHTOKEN=tok123") {
		t.Fatalf("expected .env to contain token, err=%v content=%q", err2, string(b))
	}
}

func TestRun_Ngrok_Error(t *testing.T) {
	orig := ServeNgrokFunc
	defer func() { ServeNgrokFunc = orig }()
	ServeNgrokFunc = func(ctx context.Context, epOpts []config.HTTPEndpointOption, connectOpts []ngrok.ConnectOption, mux http.Handler) error {
		return errors.New("fail")
	}

	err := Run(Options{Tunnel: true, NgrokToken: "abc"})
	if err == nil || !strings.Contains(err.Error(), "ngrok listen error") {
		t.Fatalf("expected ngrok listen error, got %v", err)
	}
}
