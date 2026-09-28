package main

import (
	"errors"
	"fmt"
	"os"

	"remotehq-simulator/internal/paste"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if errors.Is(err, paste.ErrAborted) {
			os.Exit(130)
		}
		os.Exit(1)
	}
}
