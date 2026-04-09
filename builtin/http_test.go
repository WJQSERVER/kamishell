package builtin

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestHTTPBuiltinGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("hello from http builtin"))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := HTTP([]string{server.URL}, &rmMockEnv{store: map[string]interface{}{}}, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "hello from http builtin" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestHTTPBuiltinJSONAutoUsesPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("expected JSON content type, got %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		if string(body) != `{"name":"kami"}` {
			t.Fatalf("unexpected request body: %q", string(body))
		}
		_, _ = w.Write([]byte("json-ok"))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := HTTP([]string{server.URL, "--json", `{"name":"kami"}`}, &rmMockEnv{store: map[string]interface{}{}}, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "json-ok" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestHTTPBuiltinFormBodyAndAcceptHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Fatalf("unexpected content type: %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("unexpected accept: %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		payload := string(body)
		if payload != "lang=zh&name=kami" && payload != "name=kami&lang=zh" {
			t.Fatalf("unexpected request body: %q", payload)
		}
		_, _ = w.Write([]byte("form-ok"))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	args := []string{server.URL, "--form", "name=kami", "--form", "lang=zh", "--accept", "application/json"}

	code := HTTP(args, &rmMockEnv{store: map[string]interface{}{}}, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "form-ok" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestHTTPBuiltinIncludeWritesMetadataAndBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Reply", "ok")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := HTTP([]string{"--include", server.URL}, &rmMockEnv{store: map[string]interface{}{}}, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "201 Created") {
		t.Fatalf("expected response status in stdout, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "X-Reply: ok") {
		t.Fatalf("expected response header in stdout, got %q", stdout.String())
	}
	if !strings.HasSuffix(stdout.String(), "created") {
		t.Fatalf("expected response body suffix, got %q", stdout.String())
	}
}

func TestHTTPBuiltinHeadersModeSkipsBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Reply", "ok")
		_, _ = w.Write([]byte("body-should-not-appear"))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := HTTP([]string{"--headers", server.URL}, &rmMockEnv{store: map[string]interface{}{}}, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "body-should-not-appear") {
		t.Fatalf("body unexpectedly printed: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "X-Reply: ok") {
		t.Fatalf("expected header output, got %q", stdout.String())
	}
}

func TestHTTPBuiltinReadsBodyFromStdinAndWritesFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		if string(body) != "stdin payload" {
			t.Fatalf("unexpected request body: %q", string(body))
		}
		_, _ = w.Write([]byte("saved"))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	stdin := bytes.NewBufferString("stdin payload")
	outputPath := filepath.Join(t.TempDir(), "response.txt")

	code := HTTP([]string{"--method", "POST", "--data", "-", "--output", outputPath, server.URL}, &rmMockEnv{store: map[string]interface{}{}}, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "saved" {
		t.Fatalf("unexpected output file contents: %q", string(data))
	}
}

func TestHTTPBuiltinAppliesAuthAndCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "kami" || pass != "secret" {
			t.Fatalf("unexpected basic auth: %q %q %v", user, pass, ok)
		}
		if got := r.Header.Values("X-Test"); len(got) != 2 || got[0] != "one" || got[1] != "two" {
			t.Fatalf("unexpected X-Test values: %#v", got)
		}
		_, _ = w.Write([]byte("auth-ok"))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	args := []string{server.URL, "--auth", "kami:secret", "--header", "X-Test: one", "--header", "X-Test: two"}

	code := HTTP(args, &rmMockEnv{store: map[string]interface{}{}}, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "auth-ok" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestHTTPBuiltinRetriesOnConfiguredStatuses(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("retry"))
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	args := []string{"--retries", "2", "--retry-status", "502", server.URL}

	code := HTTP(args, &rmMockEnv{store: map[string]interface{}{}}, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "ok" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestHTTPBuiltinReturnsFailureOnHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("missing"))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := HTTP([]string{server.URL}, &rmMockEnv{store: map[string]interface{}{}}, nil, stdout, stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit code, got %d", code)
	}
	if strings.TrimSpace(stdout.String()) != "missing" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unexpected status 404 Not Found") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestHTTPBuiltinRejectsConflictingBodyModes(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := HTTP([]string{"https://example.com", "--json", "{}", "--form", "name=kami"}, &rmMockEnv{store: map[string]interface{}{}}, nil, stdout, stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code for conflicting body modes")
	}
	if !strings.Contains(stderr.String(), "only one body mode") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestHTTPBuiltinRejectsConflictingOutputModes(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := HTTP([]string{"https://example.com", "--include", "--headers"}, &rmMockEnv{store: map[string]interface{}{}}, nil, stdout, stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code for conflicting output modes")
	}
	if !strings.Contains(stderr.String(), "only one of --include, --headers or --status") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}
