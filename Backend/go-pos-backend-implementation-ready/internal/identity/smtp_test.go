package identity

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuildMIMEMessageMultipartAlternative(t *testing.T) {
	msg := EmailMessage{
		To:      "owner@example.com",
		From:    SupportEmail,
		Subject: "مرحبًا بك في POS.Go — Welcome to POS.Go",
		Text:    "Welcome to POS.Go",
		HTML:    "<html><body>Welcome</body></html>",
	}
	raw := string(buildMIMEMessage("POS.Go", msg))

	if !strings.Contains(raw, "multipart/alternative") {
		t.Fatal("expected multipart/alternative when HTML is present")
	}
	if !strings.Contains(raw, `From: `) || !strings.Contains(raw, `<`+SupportEmail+`>`) {
		t.Fatalf("missing from header: %s", raw)
	}
	if !strings.Contains(raw, "Content-Type: text/plain; charset=\"UTF-8\"") ||
		!strings.Contains(raw, "Content-Type: text/html; charset=\"UTF-8\"") {
		t.Fatal("both text and html parts expected")
	}
	if !strings.Contains(raw, "Subject: =?") {
		// Arabic subject must be RFC 2047 encoded, not raw UTF-8.
		t.Fatal("expected an encoded subject")
	}
	// The base64 text part must decode back to the text body.
	if !strings.Contains(decodeFirstBase64Part(t, raw), "Welcome to POS.Go") {
		t.Fatal("text part did not decode to the message body")
	}
}

func TestBuildMIMEMessageTextOnly(t *testing.T) {
	raw := string(buildMIMEMessage("POS.Go", EmailMessage{To: "a@b.com", From: "s@x.com", Subject: "Hi", Text: "Plain body"}))
	if strings.Contains(raw, "multipart/alternative") {
		t.Fatal("text-only message must not be multipart")
	}
	if !strings.Contains(decodeFirstBase64Part(t, raw), "Plain body") {
		t.Fatal("body did not decode")
	}
}

func TestBuildMIMEMessageStripsHeaderInjection(t *testing.T) {
	raw := string(buildMIMEMessage("POS.Go", EmailMessage{
		To:      "victim@example.com\r\nBcc: attacker@evil.com",
		From:    "s@x.com",
		Subject: "ok\r\nX-Evil: 1",
		Text:    "hi",
	}))
	if strings.Contains(raw, "\r\nBcc:") || strings.Contains(raw, "\r\nX-Evil:") {
		t.Fatalf("header injection not stripped:\n%s", raw)
	}
}

// decodeFirstBase64Part returns the first base64 body block decoded to text.
func decodeFirstBase64Part(t *testing.T, raw string) string {
	t.Helper()
	idx := strings.Index(raw, "base64\r\n\r\n")
	if idx < 0 {
		t.Fatalf("no base64 body separator in:\n%s", raw)
	}
	body := raw[idx+len("base64\r\n\r\n"):]
	if end := strings.Index(body, "\r\n--"); end >= 0 {
		body = body[:end]
	}
	compact := strings.NewReplacer("\r", "", "\n", "").Replace(body)
	decoded, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		t.Fatalf("base64 decode: %v (body=%q)", err, body)
	}
	return string(decoded)
}
