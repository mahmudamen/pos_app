// openapi generates docs/openapi.json from the live Gin route table.
//
// It mirrors the production assembly through internal/transport/server.Register
// with a nil pool and no rate limiter, so the spec can never drift from what
// cmd/api actually serves — and it runs offline with no database. Unless a
// write endpoint documents its request body, the emitted operations carry the
// shared data/error envelope only; hand-augment schemas as the contract grows
// (see docs/07_API_CONTRACT.md).
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/infrastructure/ratelimit"
	"github.com/example/pos-api/internal/transport/server"
	"github.com/gin-gonic/gin"
)

func main() {
	out := "docs/openapi.json"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	server.Register(engine, server.Deps{
		Config: config.Config{
			JWTIssuer:          "pos-api",
			JWTAccessSecret:    "openapi-generator-access-secret-32-bytes-min",
			JWTRefreshSecret:   "openapi-generator-refresh-secret-32-bytes-min",
			JWTAccessTTL:       15 * time.Minute,
			JWTRefreshTTL:      7 * 24 * time.Hour,
			BcryptCost:         4,
			MaxSessionsPerUser: 5,
			HTTPMaxBodyBytes:   1 << 20,
			CashierDiscountPct: 5,
		},
		LoginLimiter: ratelimit.NewMemory(5, 5*time.Minute),
	})

	doc := build(engine.Routes())
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal openapi:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, data, 0o600); err != nil { // #nosec G703 -- path comes from the CLI -out flag, not untrusted input
		fmt.Fprintln(os.Stderr, "write "+out+":", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d paths)\n", out, len(doc["paths"].(map[string]map[string]any)))
}

func build(routes []gin.RouteInfo) map[string]any {
	type op struct {
		Tags        []string              `json:"tags"`
		Summary     string                `json:"summary,omitempty"`
		OperationID string                `json:"operationId"`
		Security    []map[string][]string `json:"security,omitempty"`
		Responses   map[string]any        `json:"responses"`
	}
	paths := make(map[string]map[string]any)

	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path != routes[j].Path {
			return routes[i].Path < routes[j].Path
		}
		return routes[i].Method < routes[j].Method
	})

	for _, r := range routes {
		pathKey := toOpenPath(r.Path)
		if paths[pathKey] == nil {
			paths[pathKey] = make(map[string]any)
		}
		tag := pathTag(r.Path)
		opID := operationID(r.Method, r.Path)
		security := requiredSecurity(r.Path)
		op := op{
			Tags:        []string{tag},
			OperationID: opID,
			Security:    security,
			Responses: map[string]any{
				"default": map[string]any{
					"description": "data/error envelope; see docs/07_API_CONTRACT.md",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{"$ref": "#/components/schemas/Envelope"},
						},
					},
				},
			},
		}
		if tag == "meta" || tag == "health" || tag == "metrics" {
			op.Summary = publicSummary(r)
		}
		paths[pathKey][strings.ToLower(r.Method)] = op
	}

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "pos-api",
			"version":     "0.1.0",
			"description": "Auto-generated from the Gin route table by cmd/openapi.",
		},
		"servers": []map[string]any{{"url": "http://127.0.0.1:8080"}},
		"paths":   paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
			"schemas": map[string]any{
				"Envelope": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"data": map[string]any{"type": "object", "nullable": true},
						"error": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"code":       map[string]any{"type": "string"},
								"message":    map[string]any{"type": "string"},
								"request_id": map[string]any{"type": "string"},
							},
						},
						"meta": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"request_id": map[string]any{"type": "string"},
							},
						},
					},
				},
			},
		},
	}
}

// toOpenPath converts Gin's ":id" params to OpenAPI "{id}" path templates.
func toOpenPath(p string) string {
	var b strings.Builder
	for i := 0; i < len(p); i++ {
		if p[i] == ':' {
			b.WriteByte('{')
			j := i + 1
			for j < len(p) && (p[j] == '/' || p[j] == ':') {
				j++
			}
			start := j
			for j < len(p) && p[j] != '/' {
				j++
			}
			b.WriteString(p[start:j])
			b.WriteByte('}')
			i = j - 1
			continue
		}
		b.WriteByte(p[i])
	}
	return b.String()
}

func pathTag(path string) string {
	seg := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(seg) > 1 {
		return seg[1]
	}
	return seg[0]
}

func operationID(method, path string) string {
	lmethod := strings.ToLower(method)
	var b strings.Builder
	b.WriteString(strings.ToUpper(lmethod[:1]))
	b.WriteString(lmethod[1:])
	for _, seg := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if seg == "" {
			continue
		}
		seg = strings.ReplaceAll(seg, ":", "")
		seg = strings.ReplaceAll(seg, "{", "")
		seg = strings.ReplaceAll(seg, "}", "")
		b.WriteString(strings.ToUpper(seg[:1]))
		b.WriteString(seg[1:])
	}
	return b.String()
}

// requiredSecurity marks which paths need a Bearer token. Login and the meta
// endpoints are public by design.
func requiredSecurity(path string) []map[string][]string {
	public := strings.HasPrefix(path, "/health") ||
		strings.HasPrefix(path, "/metrics") ||
		strings.HasPrefix(path, "/v1/meta/") ||
		path == "/v1/auth/login"
	if public {
		return nil
	}
	return []map[string][]string{{"bearerAuth": {}}}
}

func publicSummary(r gin.RouteInfo) string {
	return fmt.Sprintf("%s %s", r.Method, r.Path)
}
