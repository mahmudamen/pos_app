package mainsite

import "html/template"

// faviconSVG is the site favicon: the brand asterisk on the violet accent.
// Embedded as a string so no file reads (or embed directives) are needed at
// runtime — the binary is fully self-contained.
const faviconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
<stop offset="0" stop-color="#4D36D0"/><stop offset="1" stop-color="#937AFF"/>
</linearGradient></defs>
<rect width="64" height="64" rx="16" fill="url(#g)"/>
<g stroke="#ffffff" stroke-width="5" stroke-linecap="round">
<path d="M32 15 L32 49"/>
<path d="M17.3 23.5 L46.7 40.5"/>
<path d="M46.7 23.5 L17.3 40.5"/>
</g>
</svg>`

// brandMarkSVG is the header/footer logo mark — a geometric six-spoke asterisk
// on the violet gradient, matching the favicon so the identity is consistent.
const brandMarkSVG = `<svg class="mark" viewBox="0 0 40 40" role="img" aria-label="XamlTech">` +
	`<defs><linearGradient id="xm" x1="0" y1="0" x2="1" y2="1">` +
	`<stop offset="0" stop-color="#4D36D0"/><stop offset="1" stop-color="#937AFF"/></linearGradient></defs>` +
	`<rect width="40" height="40" rx="10" fill="url(#xm)"/>` +
	`<g stroke="#ffffff" stroke-width="3.1" stroke-linecap="round">` +
	`<path d="M20 10 L20 30"/>` +
	`<path d="M11.3 15 L28.7 25"/>` +
	`<path d="M28.7 15 L11.3 25"/>` +
	`</g></svg>`

// siteIcons maps a symbolic name to an inline SVG so the templates never fall
// back to emoji tiles. All icons inherit the surrounding text colour.
var siteIcons = map[string]string{
	"whatsapp": `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 5.5A2.5 2.5 0 0 1 5.5 3h13A2.5 2.5 0 0 1 21 5.5v8a2.5 2.5 0 0 1-2.5 2.5H9.3l-4.4 3.7A.9.9 0 0 1 3 18.9V5.5Z" fill="currentColor"/></svg>`,
	"mail":     `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 6.5A2.5 2.5 0 0 1 5.5 4h13A2.5 2.5 0 0 1 21 6.5v11A2.5 2.5 0 0 1 18.5 20h-13A2.5 2.5 0 0 1 3 17.5v-11Z" fill="none" stroke="currentColor" stroke-width="1.6"/><path d="m4 7.5 8 5.5 8-5.5" fill="none" stroke="currentColor" stroke-width="1.6"/></svg>`,
	"github":   `<svg viewBox="0 0 16 16" aria-hidden="true"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82a7.4 7.4 0 0 1 2-.27c.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z" fill="currentColor"/></svg>`,
	"apps":     `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 2 3 7v10l9 5 9-5V7l-9-5Z" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/><path d="M3 7l9 5 9-5M12 12v10" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/></svg>`,
}

// iconHTML renders a named site icon, or an empty string when unknown.
func iconHTML(name string) template.HTML {
	return template.HTML(siteIcons[name]) //nolint:gosec // static, compile-time trusted markup
}

// brandMarkHTML renders the logo mark.
func brandMarkHTML() template.HTML { return template.HTML(brandMarkSVG) } //nolint:gosec // static trusted markup
