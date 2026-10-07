package identity

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// SMTPMailer delivers email over SMTP with opportunistic STARTTLS. It is safe
// for concurrent use; each Send opens its own connection so a slow or failing
// message cannot poison later ones. The configured from/fromName is the default
// identity, overridable per message via EmailMessage.From.
type SMTPMailer struct {
	host     string
	port     int
	username string
	password string
	from     string
	fromName string
}

// Send connects, starts TLS when the server advertises it, authenticates when a
// username is configured, and delivers a multipart message (plain text plus
// HTML when provided).
func (m *SMTPMailer) Send(ctx context.Context, msg EmailMessage) error {
	from := msg.From
	if strings.TrimSpace(from) == "" {
		from = m.from
	}
	if strings.TrimSpace(from) == "" {
		return fmt.Errorf("smtp: no sender address configured")
	}
	if strings.TrimSpace(msg.To) == "" {
		return fmt.Errorf("smtp: no recipient address")
	}
	msg.From = from

	addr := net.JoinHostPort(m.host, strconv.Itoa(m.port))
	conn, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp handshake: %w", err)
	}
	defer func() { _ = client.Quit() }()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if m.username != "" {
		if err := client.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write(buildMIMEMessage(m.fromName, msg)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	return nil
}

// buildMIMEMessage assembles an RFC 5322 message. When msg.HTML is set the body
// is multipart/alternative (text first, HTML second); otherwise a single
// text/plain part is sent. Bodies are base64-encoded so non-ASCII (Arabic)
// content and long lines are always legal.
func buildMIMEMessage(fromName string, msg EmailMessage) []byte {
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(formatAddress(fromName, msg.From))
	b.WriteString("\r\n")
	b.WriteString("To: ")
	b.WriteString(sanitizeHeader(msg.To))
	b.WriteString("\r\n")
	b.WriteString("Subject: ")
	b.WriteString(encodeHeader(msg.Subject))
	b.WriteString("\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")

	text := base64.StdEncoding.EncodeToString([]byte(msg.Text))
	if strings.TrimSpace(msg.HTML) == "" || strings.TrimSpace(msg.Text) == "" {
		b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		if strings.TrimSpace(msg.Text) == "" {
			text = base64.StdEncoding.EncodeToString([]byte(htmlToText(msg.HTML)))
		}
		writeBase64Body(&b, text)
		return []byte(b.String())
	}

	const boundary = "posgo-alt-boundary-7f3c1a"
	b.WriteString("Content-Type: multipart/alternative; boundary=\"")
	b.WriteString(boundary)
	b.WriteString("\"\r\n\r\n")
	b.WriteString("--")
	b.WriteString(boundary)
	b.WriteString("\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	writeBase64Body(&b, text)
	b.WriteString("\r\n--")
	b.WriteString(boundary)
	b.WriteString("\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	writeBase64Body(&b, base64.StdEncoding.EncodeToString([]byte(msg.HTML)))
	b.WriteString("\r\n--")
	b.WriteString(boundary)
	b.WriteString("--\r\n")
	return []byte(b.String())
}

// writeBase64Body wraps a base64 blob at 76 columns, as MIME requires.
func writeBase64Body(b *strings.Builder, encoded string) {
	for len(encoded) > 76 {
		b.WriteString(encoded[:76])
		b.WriteString("\r\n")
		encoded = encoded[76:]
	}
	b.WriteString(encoded)
}

// formatAddress renders a display-name + address header, quoting the name when
// it contains characters that require it.
func formatAddress(name, address string) string {
	address = sanitizeHeader(address)
	if strings.TrimSpace(name) == "" {
		return address
	}
	return mime.QEncoding.Encode("UTF-8", sanitizeHeader(name)) + " <" + address + ">"
}

// encodeHeader RFC 2047-encodes a header value when it is not plain ASCII.
func encodeHeader(value string) string {
	value = sanitizeHeader(value)
	if isASCII(value) {
		return value
	}
	return mime.QEncoding.Encode("UTF-8", value)
}

// sanitizeHeader strips CR/LF and NUL so a value cannot inject extra headers.
func sanitizeHeader(value string) string {
	return strings.NewReplacer("\r", " ", "\n", " ", "\x00", "").Replace(value)
}

func isASCII(value string) bool {
	for _, r := range value {
		if r > 127 {
			return false
		}
	}
	return true
}

// htmlToText is a last-resort fallback that strips tags when only HTML was
// supplied. It is intentionally naive; callers normally provide Text too.
func htmlToText(html string) string {
	var out strings.Builder
	inTag := false
	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			out.WriteRune(r)
		}
	}
	return strings.TrimSpace(out.String())
}
