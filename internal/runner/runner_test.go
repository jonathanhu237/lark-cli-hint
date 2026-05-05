package runner

import (
	"bytes"
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestRunStreamsCapturesAndPreservesExitCode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	buffer := NewBoundedBuffer(1024)
	result, err := Run(context.Background(), []string{"sh", "-c", "printf out; printf err >&2; exit 7"}, Streams{
		Stdout: &stdout,
		Stderr: &stderr,
		Buffer: buffer,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.ExitCode != 7 {
		t.Fatalf("ExitCode = %d, want 7", result.ExitCode)
	}
	if stdout.String() != "out" || stderr.String() != "err" {
		t.Fatalf("streamed stdout/stderr = %q/%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(result.Output, "out") || !strings.Contains(result.Output, "err") {
		t.Fatalf("captured output = %q", result.Output)
	}
}

func TestRunSuccessExitCode(t *testing.T) {
	result, err := Run(context.Background(), []string{"sh", "-c", "exit 0"}, Streams{Buffer: NewBoundedBuffer(32)})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestRunPreservesSignalExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal exit status is Unix-specific")
	}
	result, err := Run(context.Background(), []string{"sh", "-c", "kill -TERM $$"}, Streams{Buffer: NewBoundedBuffer(32)})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.ExitCode != 143 {
		t.Fatalf("ExitCode = %d, want 143 for SIGTERM", result.ExitCode)
	}
}

func TestRunPassesStdinToWrappedCommand(t *testing.T) {
	var stdout bytes.Buffer
	result, err := Run(context.Background(), []string{"sh", "-c", "cat"}, Streams{
		Stdin:  strings.NewReader("from stdin\n"),
		Stdout: &stdout,
		Buffer: NewBoundedBuffer(1024),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	if stdout.String() != "from stdin\n" {
		t.Fatalf("stdout = %q, want stdin passthrough", stdout.String())
	}
	if result.Output != "from stdin\n" {
		t.Fatalf("captured output = %q, want stdin passthrough output", result.Output)
	}
}

func TestRunTapSeesOutputOutsideBoundedBuffer(t *testing.T) {
	var tap bytes.Buffer
	result, err := Run(context.Background(), []string{"sh", "-c", "printf '%120s' '' | tr ' ' x; printf middle; printf '%120s' '' | tr ' ' y; exit 1"}, Streams{
		Buffer: NewBoundedBuffer(80),
		Tap:    &tap,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Fatalf("ExitCode = %d, want 1", result.ExitCode)
	}
	if strings.Contains(result.Output, "middle") {
		t.Fatalf("bounded output unexpectedly retained middle signal: %q", result.Output)
	}
	if !strings.Contains(tap.String(), "middle") {
		t.Fatalf("tap did not see full streamed output: %q", tap.String())
	}
}

func TestBoundedBufferKeepsHeadAndTailWhenTruncated(t *testing.T) {
	buffer := NewBoundedBuffer(160)
	early := "missing required scope: docx:document:read\n"
	_, _ = buffer.Write([]byte(early))
	_, _ = buffer.Write([]byte(strings.Repeat("x", 300)))
	_, _ = buffer.Write([]byte("\ntail end"))
	captured := buffer.String()
	if !strings.Contains(captured, "missing required scope: docx:document:read") {
		t.Fatalf("buffer lost early error signal: %q", captured)
	}
	if !strings.Contains(captured, "tail end") {
		t.Fatalf("buffer lost tail output: %q", captured)
	}
	if !strings.Contains(captured, "truncated") {
		t.Fatalf("buffer missing truncation marker: %q", captured)
	}
	if len(captured) > 160 {
		t.Fatalf("buffer length = %d, want <= 160", len(captured))
	}
}

func TestBoundedBufferKeepsHeadAndTailForSmallLimit(t *testing.T) {
	buffer := NewBoundedBuffer(4)
	_, _ = buffer.Write([]byte("abcdef"))
	if buffer.String() != "abef" {
		t.Fatalf("buffer = %q, want head and tail", buffer.String())
	}
}
