package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	app "github.com/0xReLogic/webhook-catcher-cli/internal/app"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

// All core logic lives in internal/app. main.go only parses flags and delegates to app.Run.

func main() {
	// CLI flags
	host := flag.String("host", "127.0.0.1", "host/interface to bind (e.g., 0.0.0.0)")
	port := flag.Int("port", 3000, "port to listen on")
	tunnel := flag.Bool("tunnel", false, "enable ngrok tunnel and print public URL")
	ngrokToken := flag.String("ngrok-authtoken", "", "ngrok authtoken (optional; defaults to NGROK_AUTHTOKEN env var)")
	ngrokRegion := flag.String("ngrok-region", "", "ngrok region, e.g. us, eu, ap (optional)")
	ngrokDomain := flag.String("ngrok-domain", "", "reserved ngrok domain to use (optional)")
	flag.Parse()

	// If launched without any arguments (e.g., double-click), offer a simple mode chooser.
	if len(os.Args) == 1 {
		fmt.Println("Select mode: [1] Local (default)  [2] Tunnel (ngrok)")
		fmt.Print("Enter 1 or 2 (default 1): ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(line)
		if choice == "2" {
			*tunnel = true
		} else {
			*tunnel = false
		}
	}

	if err := run(appOptions{
		host:        *host,
		port:        *port,
		tunnel:      *tunnel,
		ngrokToken:  *ngrokToken,
		ngrokRegion: *ngrokRegion,
		ngrokDomain: *ngrokDomain,
	}); err != nil {
		log.Fatal(err)
	}
}

// Legacy test aliases to maintain backward compatibility for root package tests.
// These forward to internal/app exported symbols without changing runtime behavior.

// Indirection aliases that can be overridden by tests in the main package.
// Default implementations forward to internal/app variables, but tests may replace these.
var httpListenAndServe = func(addr string, handler http.Handler) error { return http.ListenAndServe(addr, handler) }
var httpServe = func(l net.Listener, h http.Handler) error { return http.Serve(l, h) }

type listenerWithURL = app.ListenerWithURL

var ngrokListen = func(ctx context.Context, epOpts []config.HTTPEndpointOption, connectOpts []ngrok.ConnectOption) (listenerWithURL, error) {
	return ngrok.Listen(ctx, config.HTTPEndpoint(epOpts...), connectOpts...)
}

var serveLocalFunc = func(addr string, mux http.Handler) error { return httpListenAndServe(addr, mux) }
var serveNgrokFunc = func(ctx context.Context, epOpts []config.HTTPEndpointOption, connectOpts []ngrok.ConnectOption, mux http.Handler) error {
	ln, err := ngrokListen(ctx, epOpts, connectOpts)
	if err != nil {
		return err
	}
	// Print public URL for visibility similar to app implementation
	fmt.Printf("Public URL: %s\n", ln.URL())
	return httpServe(ln, mux)
}

// Run wrapper and options struct for tests expecting run/appOptions in main package.
// This preserves old field names used by existing root tests while delegating to internal/app.
type appOptions struct {
	host        string
	port        int
	tunnel      bool
	ngrokToken  string
	ngrokRegion string
	ngrokDomain string
}

func run(opts appOptions) error {
	// Sync local overrides into internal/app so app.Run uses test stubs when provided.
	app.ServeLocalFunc = serveLocalFunc
	app.ServeNgrokFunc = serveNgrokFunc

	return app.Run(app.Options{
		Host:        opts.host,
		Port:        opts.port,
		Tunnel:      opts.tunnel,
		NgrokToken:  opts.ngrokToken,
		NgrokRegion: opts.ngrokRegion,
		NgrokDomain: opts.ngrokDomain,
	})
}
