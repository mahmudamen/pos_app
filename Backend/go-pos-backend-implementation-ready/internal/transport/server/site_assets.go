package server

import (
	_ "embed"
	"encoding/base64"
)

// Real on-device product screenshots, downscaled from the Flutter screenshot
// tour (1080x2400 -> 600px wide, lossless PNG) and embedded into the binary.
// The landing gallery renders all of them side by side so visitors see the
// actual app screens, not a drawing. Inlining as data URIs keeps the pages
// offline by construction (no extra routes for the OpenAPI generator, nothing
// for nginx to proxy).
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
// key maps into the per-language site strings (site_strings.go / _ar).
var galleryAssets = []struct {
	Key    string
	Png    []byte
	Width  int
	Height int
}{
	{"galCheckout", gal1Png, 600, 1334},
	{"galCart", gal2Png, 600, 1334},
	{"galDash", gal3Png, 600, 1334},
	{"galSales", gal4Png, 600, 1334},
	{"galCustomers", gal5Png, 600, 1334},
	{"galSessions", gal6Png, 600, 1334},
	{"galCommunity", gal7Png, 600, 1334},
	{"galNational", gal8Png, 600, 1334},
}

// assetPNGDataURI base64-encodes a PNG for inline <img src="data:image/png;base64,...">.
func assetPNGDataURI(png []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}
