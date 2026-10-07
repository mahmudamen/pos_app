package email

import (
	"html"
	"regexp"
	"strings"
)

// shell wraps an Arabic-first body block in the branded POS.Go card. The body
// is inserted as-is (it must already be escaped HTML). dir is "rtl" for the
// default Arabic presentation, "ltr" for English-only mail.
func shell(title, subtitle, body, footer string) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="ar" dir="rtl"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>POS.Go</title></head>
<body style="margin:0;padding:0;background:#f4f6fb;font-family:'Segoe UI',Tahoma,Arial,sans-serif;color:#1f2937">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f4f6fb;padding:24px 0"><tr><td align="center">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%;background:#ffffff;border-radius:14px;overflow:hidden;box-shadow:0 6px 24px rgba(15,23,42,0.08)">
<tr><td style="background:linear-gradient(135deg,#0ea5e9,#2563eb);padding:28px 32px;text-align:right">
<div style="color:#ffffff;font-size:22px;font-weight:700">POS.Go</div>`)
	if subtitle != "" {
		b.WriteString(`<div style="color:rgba(255,255,255,0.85);font-size:13px;margin-top:4px">`)
		b.WriteString(html.EscapeString(subtitle))
		b.WriteString(`</div>`)
	}
	b.WriteString(`</td></tr>
<tr><td style="padding:32px;text-align:right">`)
	if title != "" {
		b.WriteString(`<h1 style="margin:0 0 14px;font-size:22px;color:#0f172a">`)
		b.WriteString(html.EscapeString(title))
		b.WriteString(`</h1>`)
	}
	b.WriteString(body)
	b.WriteString(`</td></tr>`)
	if footer != "" {
		b.WriteString(`<tr><td style="background:#f8fafc;padding:18px 32px;text-align:center;font-size:12px;color:#94a3b8">`)
		b.WriteString(footer)
		b.WriteString(`</td></tr>`)
	}
	b.WriteString(`
</table>
</td></tr></table>
</body></html>`)
	return b.String()
}

// footerLine is the standard branded footer: privacy link + support address.
func footerLine() string {
	return `POS.Go · <a href="https://api.xamltech.com/private" style="color:#94a3b8">سياسة الخصوصية</a> · ` + SupportEmail
}

// supportLine is the standard "need help?" line pointing at the support inbox.
func supportLine() string {
	return `تحتاج مساعدة؟ راسلنا على <a href="mailto:` + SupportEmail + `" style="color:#2563eb">` + SupportEmail + `</a>`
}

// button renders a call-to-action anchor in the brand style.
func button(href, label string) string {
	return `<a href="` + html.EscapeString(href) + `" style="display:inline-block;background:#2563eb;color:#ffffff;text-decoration:none;font-weight:600;padding:11px 22px;border-radius:9px;font-size:14px">` + html.EscapeString(label) + `</a>`
}

// esc is the shorthand for HTML-escaping interpolated user data.
func esc(s string) string { return html.EscapeString(s) }

// data returns the trimmed value for a key, or fallback when empty/absent.
func data(d map[string]string, key, fallback string) string {
	if v, ok := d[key]; ok {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return fallback
}

// subjectTag matches an HTML tag so it can be stripped from subjects (which
// travel as plain text, never as markup).
var subjectTag = regexp.MustCompile(`<[^>]*>`)

// sanitizeSubject prepares a workflow-supplied title for an email Subject:
// markup is dropped and whitespace is collapsed, so an operator-typed
// "<script>…</script>" or a multi-line title never reaches the header as-is.
// The HTML body still receives the full (escaped) title.
func sanitizeSubject(s string) string {
	s = subjectTag.ReplaceAllString(s, " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}
