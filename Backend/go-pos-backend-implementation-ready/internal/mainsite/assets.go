package mainsite

// faviconSVG is the site favicon: the brand "X" on the teal→indigo gradient.
// Embedded as a string so no file reads (or embed directives) are needed at
// runtime — the binary is fully self-contained.
const faviconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
<stop offset="0" stop-color="#14a6a6"/><stop offset="1" stop-color="#4451c7"/>
</linearGradient></defs>
<rect width="64" height="64" rx="16" fill="url(#g)"/>
<path d="M19 19 L45 45 M45 19 L19 45" stroke="#ffffff" stroke-width="7" stroke-linecap="round"/>
</svg>`
