package ocr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

// ErrUnavailable is returned by Engine.OCR when the engine is disabled via
// config or the Tesseract binary is not on PATH.
var ErrUnavailable = errors.New("ocr engine unavailable")

// Engine wraps a local Tesseract binary ("ara+eng" by default) so Arabic-
// labelled Egyptian invoices can be read without any cloud service. The
// process contract is `tesseract stdin stdout -l <langs> --psm <psm>`: the
// image is piped in on stdin (leptonica decodes PNG/JPEG straight from
// FILE*), and the plain-text result is read on stdout. Keeps the main binary
// CGO-free — only the runtime container needs the tesseract packages.
type Engine struct {
	Enabled bool
	Bin     string
	Langs   string
	PSM     int
}

// NewEngine builds the runner from configuration.
func NewEngine(enabled bool, bin, langs string, psm int) Engine {
	if bin == "" {
		bin = "tesseract"
	}
	if langs == "" {
		langs = "ara+eng"
	}
	if psm < 0 {
		psm = 0
	}
	return Engine{Enabled: enabled, Bin: bin, Langs: langs, PSM: psm}
}

// Available reports whether an OCR round-trip can succeed: enabled by config
// and the binary resolvable on PATH. Handlers use it to answer 503 cleanly
// instead of failing mid-request.
func (e Engine) Available() bool {
	if !e.Enabled {
		return false
	}
	path, err := exec.LookPath(e.Bin)
	return err == nil && path != ""
}

// OCR runs Tesseract over image bytes and returns the decoded text. It honors
// the context for timeouts; a cancelled context surfaces a context error.
func (e Engine) OCR(ctx context.Context, image []byte) (string, error) {
	if !e.Available() {
		return "", ErrUnavailable
	}
	// #nosec G204 -- the binary and args are operator-controlled configuration,
	// never user input; image bytes arrive on stdin.
	cmd := exec.CommandContext(ctx, e.Bin, "stdin", "stdout",
		"-l", e.Langs, "--psm", fmt.Sprintf("%d", e.PSM))
	cmd.Stdin = bytes.NewReader(image)
	var out, errB bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errB
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("ocr timeout: %w", ctx.Err())
		}
		msg := errB.String()
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		return "", fmt.Errorf("tesseract failed: %w (stderr: %s)", err, msg)
	}
	return out.String(), nil
}
