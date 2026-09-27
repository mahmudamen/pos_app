package server

import _ "embed"

// Real on-device product screenshots, downscaled from the Flutter screenshot
// tour (1080x2400 -> 600px wide, lossless PNG) and embedded into the binary.
// The landing gallery renders all of them side by side — framed in phone
// mockups — so visitors see the actual app screens, not a drawing. The images
// are served from the /screenshots/:name route (long-lived cache headers) so
// the HTML stays light instead of inlining the PNGs as base64 data URIs.
var (
	//go:embed assets/gal1.png
	gal1Png []byte
	//go:embed assets/gal2.png
	gal2Png []byte
	//go:embed assets/gal3.png
	gal3Png []byte
	//go:embed assets/gal4.png
	gal4Png []byte
	//go:embed assets/gal5.png
	gal5Png []byte
	//go:embed assets/gal6.png
	gal6Png []byte
	//go:embed assets/gal7.png
	gal7Png []byte
	//go:embed assets/gal8.png
	gal8Png []byte
)

// galleryAssets lists every embedded screenshot in gallery order. The caption
// key maps into the per-language site strings (site_strings.go / _ar); Name is
// the URL segment served by /screenshots/:name.
var galleryAssets = []struct {
	Key    string
	Name   string
	Png    []byte
	Width  int
	Height int
}{
	{"galCheckout", "checkout", gal1Png, 600, 1334},
	{"galCart", "cart", gal2Png, 600, 1334},
	{"galDash", "dashboard", gal3Png, 600, 1334},
	{"galSales", "sales", gal4Png, 600, 1334},
	{"galCustomers", "customers", gal5Png, 600, 1334},
	{"galSessions", "sessions", gal6Png, 600, 1334},
	{"galCommunity", "community", gal7Png, 600, 1334},
	{"galNational", "national", gal8Png, 600, 1334},
}

// screenshotByName returns the embedded PNG for a /screenshots/:name segment.
func screenshotByName(name string) ([]byte, bool) {
	for _, a := range galleryAssets {
		if a.Name == name {
			return a.Png, true
		}
	}
	return nil, false
}

// Google Search Console HTML-file ownership verification. Google fetches
// /google79d5199d984f63af.html over HTTP(S) and matches the body byte for byte,
// so the file is embedded and served verbatim — no template, no added markup
// and no trailing newline. It answers on every public surface because the
// property can be verified against the POS domain, the console domain or the
// company domain, and which one is submitted is decided in the Search Console
// UI rather than here.
var (
	//go:embed assets/google79d5199d984f63af.html
	googleSiteVerification []byte
)

// googleSiteVerificationPath is the exact path Search Console requests.
const googleSiteVerificationPath = "/google79d5199d984f63af.html"
