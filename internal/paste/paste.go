package paste

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"

	"golang.org/x/term"
)

var ErrAborted = errors.New("paste aborted")

var (
	pasteStart = []byte("\x1b[200~")
	pasteEnd   = []byte("\x1b[201~")
)

// Read consumes pasted text until the bracketed-paste terminator, Ctrl-D, or EOF.
func Read(r *bufio.Reader) ([]byte, error) {
	var content []byte
	var inPaste bool
	finish := func() []byte {
		content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
		return bytes.ReplaceAll(content, []byte("\r"), []byte("\n"))
	}

	for {
		b, err := r.ReadByte()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return nil, err
			}
			if len(content) == 0 {
				return nil, io.EOF
			}
			return finish(), nil
		}

		if !inPaste {
			switch b {
			case 0x03:
				return nil, ErrAborted
			case 0x04:
				return finish(), nil
			}
		}

		if b == 0x1b {
			if marker, err := r.Peek(5); err == nil {
				switch {
				case bytes.Equal(marker, pasteStart[1:]):
					if _, err := r.Discard(5); err != nil {
						return nil, err
					}
					inPaste = true
					continue
				case bytes.Equal(marker, pasteEnd[1:]):
					if _, err := r.Discard(5); err != nil {
						return nil, err
					}
					return finish(), nil
				}
			}
		}
		content = append(content, b)
	}
}

// ReadTerminal enables raw terminal input and bracketed paste when f is a tty.
func ReadTerminal(f *os.File, r *bufio.Reader, w io.Writer) (data []byte, err error) {
	if !term.IsTerminal(int(f.Fd())) {
		return Read(r)
	}

	fd := int(f.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	defer func() {
		if _, writeErr := io.WriteString(w, "\x1b[?2004l"); err == nil && writeErr != nil {
			err = writeErr
		}
		if restoreErr := term.Restore(fd, state); err == nil && restoreErr != nil {
			err = restoreErr
		}
	}()

	if _, err = io.WriteString(w, "\x1b[?2004h"); err != nil {
		return nil, err
	}
	return Read(r)
}
