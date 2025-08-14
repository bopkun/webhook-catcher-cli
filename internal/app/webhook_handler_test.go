package app

import (
    "errors"
    "io"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestWebhookHandler_JSON(t *testing.T) {
    body := `{"hello":"world"}`
    r := httptest.NewRequest("POST", "/test", strings.NewReader(body))
    r.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    out := captureStdout(func() { WebhookHandler(w, r) })

    resp := w.Result()
    if resp.StatusCode != 200 {
        t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
    }
    b, _ := io.ReadAll(resp.Body)
    _ = resp.Body.Close()
    if string(b) != "ok" {
        t.Fatalf("expected response body 'ok', got %q", string(b))
    }

    if !strings.Contains(out, "WEBHOOK RECEIVED") && !strings.Contains(out, "WEBHOOK") {
        t.Fatalf("expected console output to mention WEBHOOK, got: %s", out)
    }
    if !strings.Contains(out, "Method:") {
        t.Fatalf("expected console output to include Method, got: %s", out)
    }
    if !strings.Contains(out, "Headers:") {
        t.Fatalf("expected console output to include Headers, got: %s", out)
    }
    if !strings.Contains(out, "\"hello\": \"world\"") {
        t.Fatalf("expected pretty JSON in console output, got: %s", out)
    }
}

func TestWebhookHandler_EmptyBody(t *testing.T) {
    r := httptest.NewRequest("GET", "/empty", nil)
    r.Header.Set("X-Empty", "1")
    w := httptest.NewRecorder()

    out := captureStdout(func() { WebhookHandler(w, r) })

    resp := w.Result()
    if resp.StatusCode != 200 {
        t.Fatalf("expected 200, got %d", resp.StatusCode)
    }
    b, _ := io.ReadAll(resp.Body)
    _ = resp.Body.Close()
    if string(b) != "ok" {
        t.Fatalf("expected ok body, got %q", string(b))
    }
    if !strings.Contains(out, "<empty>") {
        t.Fatalf("expected <empty> in output, got: %s", out)
    }
    if !strings.Contains(out, "Method:") || !strings.Contains(out, "Headers:") || !strings.Contains(out, "Body:") {
        t.Fatalf("expected sections in output, got: %s", out)
    }
}

func TestWebhookHandler_TextBody(t *testing.T) {
    r := httptest.NewRequest("POST", "/text", strings.NewReader("hello world"))
    r.Header.Set("Content-Type", "text/plain")
    w := httptest.NewRecorder()

    out := captureStdout(func() { WebhookHandler(w, r) })

    resp := w.Result()
    if resp.StatusCode != 200 {
        t.Fatalf("expected 200, got %d", resp.StatusCode)
    }
    b, _ := io.ReadAll(resp.Body)
    _ = resp.Body.Close()
    if string(b) != "ok" {
        t.Fatalf("expected ok body, got %q", string(b))
    }
    if !strings.Contains(out, "hello world") {
        t.Fatalf("expected plain body in output, got: %s", out)
    }
}

func TestWebhookHandler_MultiValueHeaders(t *testing.T) {
    r := httptest.NewRequest("POST", "/headers", strings.NewReader("{}"))
    r.Header["X-TEST"] = []string{"a", "b"}
    w := httptest.NewRecorder()

    out := captureStdout(func() { WebhookHandler(w, r) })
    out = stripANSI(out)
    if !strings.Contains(out, "X-TEST: a, b") {
        t.Fatalf("expected multi-value header joined with comma, got: %s", out)
    }
}

// errorBody simulates a request body that fails on read
type errorBody struct{}

func (errorBody) Read(p []byte) (int, error) { return 0, errors.New("read fail") }
func (errorBody) Close() error { return nil }

func TestWebhookHandler_ReadBodyError(t *testing.T) {
    r := httptest.NewRequest("POST", "/err", nil)
    // Replace body with a reader that errors
    r.Body = errorBody{}
    w := httptest.NewRecorder()

    WebhookHandler(w, r)
    resp := w.Result()
    if resp.StatusCode != 400 {
        t.Fatalf("expected 400 on read error, got %d", resp.StatusCode)
    }
    b, _ := io.ReadAll(resp.Body)
    _ = resp.Body.Close()
    if !strings.Contains(string(b), "failed to read body") {
        t.Fatalf("expected failure message in body, got %q", string(b))
    }
}
