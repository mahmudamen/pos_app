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

// Android Digital Asset Links. Android fetches
// /.well-known/assetlinks.json on the domain root over HTTPS, parses it as JSON
// and compares the listed SHA-256 certificate fingerprints against the signing
// certificate of the installed package, so the body is embedded and served
// verbatim — a template or a re-serialised struct would still parse, but the
// file exists precisely so that a byte-for-byte edit is reviewable in a diff.
//
// Two fingerprints are listed, and both are needed:
//
//   - 56:91:A7:… is the Play app-signing certificate. Once Play App Signing is
//     enrolled, Google re-signs the uploaded bundle with a key the developer
//     never holds, and that is the certificate every Play-installed copy
//     presents.
//   - 6B:3C:FE:…:EB is the upload key in this repository
//     (Flutter/pos_go_app signing config, .secrets/play/pos-go-release.jks,
//     alias pos-go-upload). A copy installed by sideloading a release APK
//     presents this one instead, and an assetlinks.json listing only Google's
//     certificate leaves those installs unverified.
//
// It answers on every public surface, matching the Search Console token above:
// Digital Asset Links is fetched per domain root, so whichever host the app
// claims in its intent filter needs the file to resolve, and all of them are
// served by this binary behind the same nginx.
var (
	//go:embed assets/assetlinks.json
	assetLinks []byte
)

// assetLinksPath is the exact path the Android package manager requests.
const assetLinksPath = "/.well-known/assetlinks.json"

// Brand mark. The site used to draw a placeholder letter "P" as an inline SVG
// (header) and as a data-URI favicon; both are replaced by the real POS.Go app
// icon — the mark that ships on Play — so the marketing site, the favicon, the
// social preview and the server-rendered auth pages all carry one identity.
// The 192/32 sizes are pre-scaled so a cold load does not downscale the 512px
// source in the browser, and the 1200x630 card is the Open Graph preview.
var (
	//go:embed assets/brand-512.png
	brandMark512 []byte
	//go:embed assets/brand-192.png
	brandMark192 []byte
	//go:embed assets/brand-32.png
	brandMark32 []byte
	//go:embed assets/og-1200x630.png
	brandOgImage []byte
)
