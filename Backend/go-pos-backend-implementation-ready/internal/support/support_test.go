package support

import (
	"strings"
	"testing"
)

func TestParseMessage_PlainText(t *testing.T) {
	raw := "From: Alice <alice@example.com>\r\n" +
		"To: support@xamltech.com\r\n" +
		"Subject: Need help with a refund\r\n" +
		"Message-ID: <abc123@example.com>\r\n" +
		"Date: Mon, 06 Oct 2026 10:00:00 +0000\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" +
		"My refund has not arrived yet.\r\n"

	got, err := ParseMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if got.FromEmail != "alice@example.com" || got.FromName != "Alice" {
		t.Fatalf("from = %q <%q>", got.FromName, got.FromEmail)
	}
	if got.ToEmail != "support@xamltech.com" {
		t.Fatalf("to = %q", got.ToEmail)
	}
	if got.Subject != "Need help with a refund" {
		t.Fatalf("subject = %q", got.Subject)
	}
	if got.MessageID != "abc123@example.com" {
		t.Fatalf("message-id = %q", got.MessageID)
	}
	if !strings.Contains(got.BodyText, "refund has not arrived") {
		t.Fatalf("body = %q", got.BodyText)
	}
	if got.BodyHTML != "" {
		t.Fatalf("html should be empty, got %q", got.BodyHTML)
	}
}

func TestParseMessage_MultipartAlternative(t *testing.T) {
	raw := "From: Bob <bob@example.com>\r\n" +
		"To: support@xamltech.com\r\n" +
		"Subject: Invoice question\r\n" +
		"In-Reply-To: <prev@xamltech.com>\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/alternative; boundary=\"BOUND\"\r\n" +
		"\r\n" +
		"--BOUND\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" +
		"Please check my invoice.\r\n" +
		"--BOUND\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"\r\n" +
		"<p>Please <b>check</b> my invoice.</p>\r\n" +
		"--BOUND--\r\n"

	got, err := ParseMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if got.BodyText != "Please check my invoice." {
		t.Fatalf("text = %q", got.BodyText)
	}
	if !strings.Contains(got.BodyHTML, "<b>check</b>") {
		t.Fatalf("html = %q", got.BodyHTML)
	}
	if got.InReplyTo != "<prev@xamltech.com>" {
		t.Fatalf("in-reply-to = %q", got.InReplyTo)
	}
}

func TestParseMessage_HTMLOnlyFallsBackToText(t *testing.T) {
	raw := "From: C <c@example.com>\r\n" +
		"Subject: hi\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"\r\n" +
		"<p>Hello&nbsp;<b>there</b></p>\r\n"

	got, err := ParseMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if got.BodyText != "Hello there" {
		t.Fatalf("fallback text = %q", got.BodyText)
	}
}

func TestThreadKey(t *testing.T) {
	cases := map[string]string{
		"Re: Need help":                 "need help",
		"RE: Fwd:  Need   Help":         "need help",
		"Sv: Re: Need help":             "need help",
		"Aw: [3] Need help":             "need help",
		"Need help":                     "need help",
		"  Re:Re:   spaced   subject  ": "spaced subject",
		"":                              "",
	}
	for in, want := range cases {
		if got := ThreadKey(in); got != want {
			t.Errorf("ThreadKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStripHTMLToText(t *testing.T) {
	got := stripHTMLToText("<div><p>Hello&nbsp;<b>World</b> &amp; friends</p></div>")
	if got != "Hello World & friends" {
		t.Fatalf("stripHTMLToText = %q", got)
	}
	if stripHTMLToText("") != "" {
		t.Fatal("empty html should yield empty text")
	}
}

func TestNewThreadID_UniqueV4(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		id := NewThreadID()
		if len(id) != 36 {
			t.Fatalf("id %q length = %d", id, len(id))
		}
		if id[14] != '4' {
			t.Fatalf("id %q is not version 4", id)
		}
		if c := id[19]; c != '8' && c != '9' && c != 'a' && c != 'b' {
			t.Fatalf("id %q has bad variant nibble %q", id, c)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}
