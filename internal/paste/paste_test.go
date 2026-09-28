package paste

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"testing/iotest"
)

func TestRead(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantErr   error
		remaining string
	}{
		{
			name:      "bracketed paste strips markers",
			input:     "before\x1b[200~pasted\x1b[201~after",
			want:      "beforepasted",
			remaining: "after",
		},
		{
			name:      "does not consume after terminator",
			input:     "\x1b[200~html\x1b[201~1000\n12\n",
			want:      "html",
			remaining: "1000\n12\n",
		},
		{
			name:  "normalizes CR and CRLF",
			input: "a\rb\r\nc\n",
			want:  "a\nb\nc\n",
		},
		{
			name:      "Ctrl-D terminates outside paste",
			input:     "before\x04after",
			want:      "before",
			remaining: "after",
		},
		{
			name:      "Ctrl-C aborts outside paste",
			input:     "before\x03after",
			wantErr:   ErrAborted,
			remaining: "after",
		},
		{
			name:  "Ctrl-C is content inside paste",
			input: "\x1b[200~a\x03b\x04c\x1b[201~",
			want:  "a\x03b\x04c",
		},
		{
			name:  "partial marker prefix is content",
			input: "x\x1b[20Z",
			want:  "x\x1b[20Z",
		},
		{
			name:    "empty EOF",
			input:   "",
			wantErr: io.EOF,
		},
		{
			name:  "content followed by EOF",
			input: "content",
			want:  "content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			got, err := Read(r)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Read() error = %v, want %v", err, tt.wantErr)
			}
			if string(got) != tt.want {
				t.Fatalf("Read() = %q, want %q", got, tt.want)
			}
			remaining, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("reading remaining input: %v", err)
			}
			if string(remaining) != tt.remaining {
				t.Fatalf("remaining input = %q, want %q", remaining, tt.remaining)
			}
		})
	}
}

func TestReadOneByteReader(t *testing.T) {
	input := "prefix\x1b[200~a\r\nb\x1b[201~trailing"
	r := bufio.NewReader(iotest.OneByteReader(strings.NewReader(input)))

	got, err := Read(r)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if string(got) != "prefixa\nb" {
		t.Fatalf("Read() = %q, want %q", got, "prefixa\nb")
	}
	remaining, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading remaining input: %v", err)
	}
	if string(remaining) != "trailing" {
		t.Fatalf("remaining input = %q, want %q", remaining, "trailing")
	}
}

func TestReadTerminalNonTerminal(t *testing.T) {
	f, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	defer func() { _ = f.Close() }()

	got, err := ReadTerminal(f, bufio.NewReader(bytes.NewBufferString("pasted\x04later")), io.Discard)
	if err != nil {
		t.Fatalf("ReadTerminal() error = %v", err)
	}
	if string(got) != "pasted" {
		t.Fatalf("ReadTerminal() = %q, want %q", got, "pasted")
	}
}
