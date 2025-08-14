package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

const (
	colorReset   = "\033[0m"
	colorGreen   = "\033[32m"
	colorCyan    = "\033[36m"
	colorYellow  = "\033[33m"
	colorMagenta = "\033[35m"
	colorBlue    = "\033[34m"
	colorRed     = "\033[31m"
	colorBold    = "\033[1m"
)

var printMu sync.Mutex

// Indirections for easier testing (do not change runtime behavior)
var HTTPListenAndServe = http.ListenAndServe
var HTTPServe = http.Serve

type ListenerWithURL interface {
	net.Listener
	URL() string
}

var NgrokListen = func(ctx context.Context, epOpts []config.HTTPEndpointOption, connectOpts []ngrok.ConnectOption) (ListenerWithURL, error) {
	return ngrok.Listen(ctx, config.HTTPEndpoint(epOpts...), connectOpts...)
}

// serveLocalFunc starts a local HTTP server. Overridable in tests.
var ServeLocalFunc = func(addr string, mux http.Handler) error {
	return HTTPListenAndServe(addr, mux)
}

// serveNgrokFunc starts an ngrok listener and serves HTTP on it. Overridable in tests.
var ServeNgrokFunc = func(ctx context.Context, epOpts []config.HTTPEndpointOption, connectOpts []ngrok.ConnectOption, mux http.Handler) error {
	ln, err := NgrokListen(ctx, epOpts, connectOpts)
	if err != nil {
		return err
	}
	log.Printf("%s[INFO]%s Public URL: %s", colorGreen, colorReset, ln.URL())
	log.Printf("%s[INFO]%s All incoming requests will be printed below. Press Ctrl+C to stop.", colorGreen, colorReset)
	return HTTPServe(ln, mux)
}

// Options contains runtime options for Run.
type Options struct {
	Host        string
	Port        int
	Tunnel      bool
	NgrokToken  string
	NgrokRegion string
	NgrokDomain string
}

// Run contains the main logic, extracted for testability.
func Run(opts Options) error {
	// Unify log output to stdout to reduce mixed streams with fmt prints
	log.SetOutput(os.Stdout)

	// Load .env if present to populate environment (for NGROK_AUTHTOKEN etc.)
	LoadDotEnv(".env")

	// Handler mux
	mux := http.NewServeMux()
	mux.HandleFunc("/", WebhookHandler)

	if opts.Tunnel {
		// Start ngrok tunnel
		ctx := context.Background()
		// Resolve authtoken (flag -> env -> prompt)
		token := strings.TrimSpace(opts.NgrokToken)
		if token == "" {
			token = strings.TrimSpace(os.Getenv("NGROK_AUTHTOKEN"))
		}
		if token == "" {
			fmt.Printf("%s[INFO]%s Welcome to Webhook Catcher!\n", colorGreen, colorReset)
			fmt.Printf("%s[INFO]%s It looks like this is your first time enabling tunneling.\n", colorGreen, colorReset)
			fmt.Println("[PROMPT] Open https://dashboard.ngrok.com/get-started/your-authtoken")
			fmt.Print("[PROMPT] Paste your ngrok Authtoken here: ")
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			token = strings.TrimSpace(line)
			if token == "" {
				return fmt.Errorf("missing ngrok Authtoken. Get one at https://dashboard.ngrok.com/get-started/your-authtoken")
			}
			// Persist to environment for this process and future runs
			_ = os.Setenv("NGROK_AUTHTOKEN", token)
			if err := SaveEnvVar(".env", "NGROK_AUTHTOKEN", token); err != nil {
				log.Printf("[WARN] failed to persist token to .env: %v", err)
			} else {
				log.Printf("%s[INFO]%s Saved NGROK_AUTHTOKEN to .env for future runs.", colorGreen, colorReset)
			}
		}

		// Build connect options
		var connectOpts []ngrok.ConnectOption
		if token != "" {
			connectOpts = append(connectOpts, ngrok.WithAuthtoken(token))
		} else {
			connectOpts = append(connectOpts, ngrok.WithAuthtokenFromEnv())
		}
		if opts.NgrokRegion != "" {
			connectOpts = append(connectOpts, ngrok.WithRegion(opts.NgrokRegion))
		}
		// Endpoint options
		var epOpts []config.HTTPEndpointOption
		if opts.NgrokDomain != "" {
			epOpts = append(epOpts, config.WithDomain(opts.NgrokDomain))
		}

		log.Printf("%s[INFO]%s Webhook Catcher is running!", colorGreen, colorReset)
		log.Printf("%s[INFO]%s Starting ngrok tunnel...", colorGreen, colorReset)

		if err := ServeNgrokFunc(ctx, epOpts, connectOpts, mux); err != nil {
			return fmt.Errorf("ngrok listen error: %w\n[HINT] Ensure your ngrok Authtoken is valid: https://dashboard.ngrok.com/get-started/your-authtoken", err)
		}
		return nil
	}

	// Local listener mode
	addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
	log.Printf("%s[INFO]%s Webhook Catcher is running!", colorGreen, colorReset)
	log.Printf("%s[INFO]%s Listening on http://%s", colorGreen, colorReset, addr)
	log.Printf("%s[INFO]%s All incoming requests will be printed below. Press Ctrl+C to stop.", colorGreen, colorReset)
	if err := ServeLocalFunc(addr, mux); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	// Always respond 200 OK quickly to not block senders
	defer r.Body.Close()

	// Read body (limit to avoid excessive memory use)
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10 MB
	if err != nil {
		log.Printf("failed to read request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("failed to read body"))
		return
	}

	// Respond first, then print to console
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))

	// Build output atomically to avoid interleaving
	printMu.Lock()
	defer printMu.Unlock()

	var out bytes.Buffer

	// Timestamp
	ts := time.Now().Format("2006-01-02 15:04:05")

	fmt.Fprintf(&out, "\n%s--- WEBHOOK RECEIVED (%s) ---%s\n\n", colorBold, ts, colorReset)

	// Method and path
	mColored := ColorMethod(r.Method)
	fmt.Fprintf(&out, "%sMethod:%s %s %s%s%s\n\n", colorCyan, colorReset, mColored, colorYellow, r.URL.Path, colorReset)

	// Headers
	out.WriteString("Headers:\n")
	keys := make([]string, 0, len(r.Header))
	for k := range r.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&out, "  %s%s%s: %s\n", colorBlue, k, colorReset, strings.Join(r.Header[k], ", "))
	}
	out.WriteString("\n")

	// Body
	out.WriteString("Body:\n")
	if pretty, ok := TryPrettyJSON(body); ok {
		fmt.Fprintf(&out, "%s%s%s\n", colorGreen, pretty, colorReset)
	} else if len(body) > 0 {
		out.WriteString(string(body) + "\n")
	} else {
		out.WriteString("<empty>\n")
	}

	out.WriteString(strings.Repeat("-", 50) + "\n")

	// Single print to stdout
	fmt.Print(out.String())
}

func TryPrettyJSON(b []byte) (string, bool) {
	// Trim to increase chance of valid JSON detection
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) == 0 {
		return "", false
	}
	if !(trimmed[0] == '{' || trimmed[0] == '[') {
		return "", false
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, trimmed, "", "  "); err != nil {
		return "", false
	}
	return buf.String(), true
}

func ColorMethod(m string) string {
	switch strings.ToUpper(m) {
	case "GET":
		return colorGreen + m + colorReset
	case "POST":
		return colorYellow + m + colorReset
	case "PUT":
		return colorMagenta + m + colorReset
	case "DELETE":
		return colorRed + m + colorReset
	default:
		return colorCyan + m + colorReset
	}
}

func LoadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	// Normalize newlines and parse
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if eq := strings.Index(line, "="); eq > 0 {
			key := strings.TrimSpace(line[:eq])
			val := strings.TrimSpace(line[eq+1:])
			val = strings.Trim(val, "\"'")
			_ = os.Setenv(key, val)
		}
	}
}

// saveEnvVar upserts a KEY=VALUE line into the given .env file.
func SaveEnvVar(path, key, value string) error {
	var lines []string
	if b, err := os.ReadFile(path); err == nil {
		lines = strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
		updated := false
		for i, raw := range lines {
			s := strings.TrimSpace(raw)
			if s == "" || strings.HasPrefix(s, "#") {
				continue
			}
			if idx := strings.Index(s, "="); idx > 0 {
				k := strings.TrimSpace(s[:idx])
				if k == key {
					lines[i] = fmt.Sprintf("%s=%s", key, value)
					updated = true
				}
			}
		}
		if !updated {
			lines = append(lines, fmt.Sprintf("%s=%s", key, value))
		}
	} else {
		lines = []string{fmt.Sprintf("%s=%s", key, value)}
	}
	content := strings.Join(lines, "\n")
	return os.WriteFile(path, []byte(content), 0600)
}
