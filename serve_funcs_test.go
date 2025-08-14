package main

import (
    "context"
    "errors"
    "net"
    "net/http"
    "testing"

    "golang.ngrok.com/ngrok"
    "golang.ngrok.com/ngrok/config"
)

func TestServeLocal_Indirection(t *testing.T) {
    // save & restore
    orig := httpListenAndServe
    defer func() { httpListenAndServe = orig }()

    var gotAddr string
    var gotHandler http.Handler
    httpListenAndServe = func(addr string, handler http.Handler) error {
        gotAddr = addr
        gotHandler = handler
        return nil
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
    if err := serveLocalFunc(":1234", mux); err != nil {
        t.Fatalf("serveLocalFunc returned error: %v", err)
    }
    if gotAddr != ":1234" || gotHandler == nil {
        t.Fatalf("indirection not used or wrong args: addr=%q handlerNil=%v", gotAddr, gotHandler == nil)
    }
}

type fakeLn struct{ url string }

func (f *fakeLn) Accept() (net.Conn, error) { return nil, errors.New("no accept") }
func (f *fakeLn) Close() error               { return nil }
func (f *fakeLn) Addr() net.Addr             { return &net.TCPAddr{} }
func (f *fakeLn) URL() string                { return f.url }

func TestServeNgrok_Indirection(t *testing.T) {
    // save & restore
    origListen := ngrokListen
    origServe := httpServe
    defer func() { ngrokListen = origListen; httpServe = origServe }()

    listenCalled := false
    serveCalled := false

    ngrokListen = func(ctx context.Context, epOpts []config.HTTPEndpointOption, connectOpts []ngrok.ConnectOption) (listenerWithURL, error) {
        listenCalled = true
        return &fakeLn{url: "https://example.tunnel"}, nil
    }
    httpServe = func(l net.Listener, h http.Handler) error {
        serveCalled = true
        return nil
    }

    mux := http.NewServeMux()
    if err := serveNgrokFunc(context.Background(), nil, nil, mux); err != nil {
        t.Fatalf("serveNgrokFunc returned error: %v", err)
    }
    if !listenCalled || !serveCalled {
        t.Fatalf("expected both ngrokListen and httpServe to be called, got listen=%v serve=%v", listenCalled, serveCalled)
    }
}
