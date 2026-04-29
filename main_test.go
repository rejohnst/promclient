package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestClientOptionsPromIPSupportsIPv6(t *testing.T) {
	opts := clientOptions{
		promIP:         "::1",
		timeoutSeconds: 5,
	}

	address, timeout, err := opts.addressAndTimeout()
	if err != nil {
		t.Fatalf("addressAndTimeout returned error: %v", err)
	}
	if got, want := address, "http://[::1]:9090"; got != want {
		t.Fatalf("unexpected address: got %q want %q", got, want)
	}
	if got, want := timeout, 5*time.Second; got != want {
		t.Fatalf("unexpected timeout: got %s want %s", got, want)
	}
}

func TestRunRejectsStepWithoutRangeLength(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"query", "-query=up", "-step=30s"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("unexpected exit code: got %d want 2", code)
	}
	if got := stderr.String(); !strings.Contains(got, "query: -step requires -len") {
		t.Fatalf("unexpected stderr: %q", got)
	}
}

func TestRunScopesCommandSpecificFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"metrics", "-severity=critical"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("unexpected exit code: got %d want 2", code)
	}
	if got := stderr.String(); !strings.Contains(got, "flag provided but not defined: -severity") {
		t.Fatalf("unexpected stderr: %q", got)
	}
}
