package server

import (
	_ "embed"
	"encoding/base64"
)

// Real on-device product screenshots, optimized from the Flutter screenshot
// tour (1080x2400 -> 440px wide, palette-quantized) and embedded into the
// binary. The landing hero renders them inside a phone mockup with an
// interactive tab switcher so visitors see the actual app, not a drawing.
// Inlining as data URIs keeps the pages offline by construction (no extra
// routes for the OpenAPI generator, nothing for nginx to proxy).

var (
	//go:embed assets/shot-pos.png
	shotPosPng []byte
	//go:embed assets/shot-dashboard.png
	shotDashPng []byte
	//go:embed assets/shot-sales.png
	shotSalesPng []byte
)

// assetPNGDataURI base64-encodes a PNG for inline <img src="data:image/png;base64,...">.
func assetPNGDataURI(png []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}
