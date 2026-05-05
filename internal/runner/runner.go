package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
)

type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Buffer *BoundedBuffer
	Tap    io.Writer
}

type Result struct {
	Command  []string
	ExitCode int
	Output   string
}

func Run(ctx context.Context, command []string, streams Streams) (Result, error) {
	if len(command) == 0 {
		return Result{}, errors.New("missing command")
	}
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)

	buffer := streams.Buffer
	if buffer == nil {
		buffer = NewBoundedBuffer(256 * 1024)
	}
	stdout := streams.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	stderr := streams.Stderr
	if stderr == nil {
		stderr = io.Discard
	}
	if streams.Stdin != nil {
		cmd.Stdin = streams.Stdin
	}
	cmd.Stdout = multiWriter(stdout, buffer, streams.Tap)
	cmd.Stderr = multiWriter(stderr, buffer, streams.Tap)

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitCodeFrom(exitErr)
		} else {
			return Result{}, err
		}
	}

	return Result{
		Command:  append([]string{}, command...),
		ExitCode: exitCode,
		Output:   buffer.String(),
	}, nil
}

func multiWriter(writers ...io.Writer) io.Writer {
	var out []io.Writer
	for _, writer := range writers {
		if writer != nil {
			out = append(out, writer)
		}
	}
	if len(out) == 0 {
		return io.Discard
	}
	return io.MultiWriter(out...)
}

func exitCodeFrom(exitErr *exec.ExitError) int {
	if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	return exitErr.ExitCode()
}

type BoundedBuffer struct {
	mu        sync.Mutex
	limit     int
	buf       []byte
	head      []byte
	tail      []byte
	truncated bool
}

const truncationMarker = "\n...[lark-cue output truncated]...\n"

func NewBoundedBuffer(limit int) *BoundedBuffer {
	if limit <= 0 {
		limit = 1
	}
	return &BoundedBuffer{limit: limit}
}

func (b *BoundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.truncated && len(b.buf)+len(p) <= b.limit {
		b.buf = append(b.buf, p...)
		return len(p), nil
	}

	headLimit, tailLimit, _ := b.budgets()
	if !b.truncated {
		data := append(append([]byte(nil), b.buf...), p...)
		b.head = append([]byte(nil), data[:min(len(data), headLimit)]...)
		b.tail = append([]byte(nil), data[max(0, len(data)-tailLimit):]...)
		b.buf = nil
		b.truncated = true
		return len(p), nil
	}
	b.tail = append(b.tail, p...)
	if len(b.tail) > tailLimit {
		b.tail = append([]byte(nil), b.tail[len(b.tail)-tailLimit:]...)
	}
	return len(p), nil
}

func (b *BoundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.truncated {
		_, _, marker := b.budgets()
		out := append([]byte(nil), b.head...)
		out = append(out, marker...)
		out = append(out, b.tail...)
		return string(out)
	}
	return string(append([]byte(nil), b.buf...))
}

func (b *BoundedBuffer) budgets() (int, int, []byte) {
	if b.limit <= len(truncationMarker)+2 {
		headLimit := b.limit / 2
		return headLimit, b.limit - headLimit, nil
	}
	remaining := b.limit - len(truncationMarker)
	headLimit := remaining / 2
	return headLimit, remaining - headLimit, []byte(truncationMarker)
}

func CommandString(command []string) string {
	var out bytes.Buffer
	for i, part := range command {
		if i > 0 {
			out.WriteByte(' ')
		}
		if part == "" || needsQuoting(part) {
			out.WriteString(fmt.Sprintf("%q", part))
		} else {
			out.WriteString(part)
		}
	}
	return out.String()
}

func needsQuoting(value string) bool {
	for _, r := range value {
		if r == ' ' || r == '\t' || r == '\n' || r == '"' || r == '\'' {
			return true
		}
	}
	return false
}
