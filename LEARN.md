# LEARN: How I Built Webhook Catcher CLI

Step-by-step instructions on building this webhook catching CLI tool.

## Goal
Create a CLI tool that catches webhooks locally or via ngrok tunnel, with pretty JSON output and colorized logs.

**Tech Stack**: Go 1.21, ngrok SDK, GitHub Actions

---

## Step 1: Project Setup
```bash
mkdir webhook-catcher-cli && cd webhook-catcher-cli
go mod init github.com/0xReLogic/webhook-catcher-cli
go get golang.ngrok.com/ngrok@latest
```

Add `.gitignore`, `LICENSE` (MIT), basic project files.

---

## Step 2: Basic HTTP Server
Create `main.go` with simple webhook handler:
```go
func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        body, _ := io.ReadAll(r.Body)
        fmt.Printf("Method: %s\nURL: %s\nBody: %s\n", r.Method, r.URL, body)
        w.WriteHeader(http.StatusOK)
    })
    log.Fatal(http.ListenAndServe(":3000", nil))
}
```

---

## Step 3: Pretty Output & Colors
Extract core logic to `internal/app/` package:
- Add JSON pretty printing with `json.MarshalIndent`
- Add ANSI colors for HTTP methods
- Sort and display headers
- Add mutex for thread-safe output

---

## Step 4: ngrok Tunnel Support
Add tunnel mode with interactive setup:
- Read token from flag → env → prompt
- Save token to `.env` for future use  
- Implement ngrok tunnel with `golang.ngrok.com/ngrok`
- Handle token errors gracefully

---

## Step 5: Testing
Create comprehensive test suite:
- Unit tests for helpers, handlers, main logic
- Use stdin pipes to test interactive flows
- Add indirection functions for HTTP servers (testability)
- Achieve >90% code coverage

Key insight: Use function variables to make external dependencies testable.

---

## Step 6: CI/CD Setup
Create `.github/workflows/ci.yml`:
- Run tests on push/PR
- Upload coverage to Codecov
- Use Go 1.21.x consistently

---

## Step 7: Release Workflow
Create `.github/workflows/release.yml`:
- Multi-platform builds (Linux/macOS/Windows)
- Trigger on tag push or manual dispatch
- Upload binaries as release artifacts
- Handle different event types correctly

---

## Step 8: Bug Fixes
Fix critical issues:
- **Stack overflow**: Remove circular function assignments
- **YAML syntax**: Fix workflow triggers structure
- **Tag consistency**: Use proper versioning (v1.0.0 not v.1.0.0)

---

## Step 9: Documentation
Create comprehensive docs:
- README with usage examples
- Provider testing section
- Multiple badges (build, coverage, etc.)

---

## Step 10: Release
Release via:
```bash
git tag -a v1.0.0 -m "v1.0.0"
git push origin v1.0.0
```
Or use GitHub Actions manual dispatch.

---

## Key Learnings
1. **Testability**: Use dependency injection for external services
2. **Security**: Read secrets at runtime, never embed in binaries  
3. **CI/CD**: Matrix builds for cross-platform support
4. **UX**: Interactive onboarding for better user experience

## Final Structure
```
webhook-catcher-cli/
├── .github/workflows/   # CI and release
├── internal/app/        # Core logic and tests
├── main.go             # CLI entry point
├── go.mod              # Dependencies  
└── README.md           # Documentation
```

Result: Robust, tested CLI tool with automatic CI/CD and multi-platform releases.
