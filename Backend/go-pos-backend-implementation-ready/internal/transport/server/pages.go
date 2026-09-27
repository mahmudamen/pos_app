package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/pos-api/internal/config"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Public brand, pricing and privacy pages served at the API root. The reverse
// proxy in front of the app fans the three public domains in, and this package
// decides what each one shows:
//
//   - xamltech.com        → the XAMLtech company home (GET /)
//   - posgo.xamltech.com  → the POS.Go product landing, pricing and privacy
//   - api.xamltech.com    → the SaaS console (GET / redirects to /admin/)
//
// The visual system is the POS.Go light brand (clean light canvas + soft
// cards + hairline-divided sections) with emerald kept as the single
// interactive accent (CTAs, links, focus), and the gallery frames the real
// app screenshots in phone mockups. Every page is bilingual
// (Arabic / English) and direction-aware: the language is chosen from ?lang=,
// then a cookie, then Accept-Language, and the page renders with the matching
// `lang`/`dir` attributes and logical CSS. The pages need no auth; /pricing
// reads the platform plans when a pool is available and falls back to the
// seeded catalog otherwise (so the OpenAPI generator, which registers routes
// with a nil pool, still works offline).

// siteTemplateData is the render context for every public page.
type siteTemplateData struct {
	Page         string
	Lang         string
	Dir          string
	Year         int
	T            map[string]string
	Plans        []planCard
	Compare      []compareRow
	ComparePlans []string
	// POSURL and SaaSURL are absolute origins for the two product domains, so
	// the company home (served on xamltech.com) can link out to them without
	// inheriting its own host.
	POSURL  string
	SaaSURL string
	// Gallery holds the captioned screenshots for the landing gallery. Images
	// are served from /screenshots/:name; Src is a plain path so the HTML stays
	// light instead of carrying inlined base64 data URIs.
	Gallery []siteGalleryItem
	// Modules is the localized "Everything included" grid.
	Modules []siteModule
}

// siteGalleryItem is one captioned screenshot in the landing gallery.
type siteGalleryItem struct {
	Src     string
	Caption string
	Width   int
	Height  int
}

// siteModule is one card of the "Everything included" section: a title, a
// lead-in sentence and three proof bullets. Icon is a complete inline SVG.
type siteModule struct {
	Title  string
	Detail string
	Points []string
	Icon   template.HTML
}

// moduleIcons are the small inline icons for the module cards.
var moduleIcons = map[string]template.HTML{
	"sell":      `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M6 2.5h12l2 5.5v11.5a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V8z"/><path d="M2.6 8h18.8"/><path d="M9 11.5h6"/></svg>`,
	"payments":  `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="2.5" y="6" width="19" height="12" rx="2"/><circle cx="12" cy="12" r="2.6"/><path d="M6 9.5h.01M18 14.5h.01"/></svg>`,
	"registers": `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3.5 6h11v13.5a2 2 0 0 1-2 2H5.5a2 2 0 0 1-2-2z"/><path d="M14.5 9h3l4 3v5.5h-7"/><path d="M8 12.5h.01M8 16h.01M11 12.5h.01M11 16h.01"/></svg>`,
	"inventory": `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2.5l8.5 4.5v9.5L12 21l-8.5-4.5V7z"/><path d="M12 12l8.6-4.6M12 12L3.4 7.4M12 12v9"/></svg>`,
	"lots":      `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2.5l9.5 5-9.5 5-9.5-5z"/><path d="M2.5 12.5l9.5 5 9.5-5"/><path d="M2.5 17.5l9.5 5 9.5-5"/></svg>`,
	"discount":  `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2.5l2.2 4.6 5 .8-3.6 3.5.9 5-4.5-2.4L7.5 16.4l.9-5-3.6-3.5 5-.8z"/><path d="M19 19l.5 1.5L21 21l-1.5.5L19 23l-.5-1.5L17 21l1.5-.5z"/></svg>`,
	"reports":   `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 20V10"/><path d="M10 20V4"/><path d="M16 20v-7"/><path d="M22 20H2"/></svg>`,
	"loyalty":   `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20.5S4.5 15.6 3 11.2a5.1 5.1 0 0 1 9-4.1 5.1 5.1 0 0 1 9 4.1c-1.5 4.4-9 9.3-9 9.3z"/></svg>`,
	"receipts":  `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M7 3l2 1 2-1 2 1 2-1 2 1v17l-2-1-2 1-2-1-2 1-2-1z"/><path d="M9.5 8h5M9.5 12h5M9.5 16h3"/></svg>`,
	"sync":      `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M20.5 5.5v5h-5"/><path d="M3.5 18.5v-5h5"/><path d="M19.6 10a7.5 7.5 0 0 0-13.2-3.2L3.5 10M4.4 14a7.5 7.5 0 0 0 13.2 3.2L20.5 14"/></svg>`,
	"team":      `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="8" r="3.2"/><path d="M3 20a6 6 0 0 1 12 0"/><circle cx="17" cy="9" r="2.5"/><path d="M16.5 13.6A4.5 4.5 0 0 1 21 19"/></svg>`,
	"panel":     `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2.2"/><path d="M3 9h18M9 9v12"/></svg>`,
}

// buildModules assembles the localized "Everything included" cards from the
// per-language site strings. 12 modules cover the full shipped feature set.
func buildModules(lang string) []siteModule {
	t := siteStrings[lang]
	defs := []struct {
		Icon string
		Key  string
	}{
		{"sell", "all1"}, {"payments", "all2"}, {"registers", "all3"},
		{"inventory", "all4"}, {"lots", "all5"}, {"discount", "all6"},
		{"reports", "all7"}, {"loyalty", "all8"}, {"receipts", "all9"},
		{"sync", "all10"}, {"team", "all11"}, {"panel", "all12"},
	}
	mods := make([]siteModule, 0, len(defs))
	for _, d := range defs {
		mods = append(mods, siteModule{
			Title:  t[d.Key+"t"],
			Detail: t[d.Key+"d"],
			Icon:   moduleIcons[d.Icon],
			Points: []string{t[d.Key+"a"], t[d.Key+"b"], t[d.Key+"c"]},
		})
	}
	return mods
}

// planCard is a display-ready subscription plan, localized for the pricing page.
type planCard struct {
	Code        string
	Name        string
	Description string
	Price       string
	Currency    string
	Period      string
	Features    []string
	Keys        []string
	Featured    bool
}

// compareRow is one feature row of the /pricing comparison table: a localized
// label and one included/excluded flag per plan (aligned to ComparePlans).
type compareRow struct {
	Key    string
	Label  string
	Values []bool
}

func pickSiteLang(c *gin.Context) string {
	lang := strings.ToLower(strings.TrimSpace(c.Query("lang")))
	if lang == "ar" || lang == "en" {
		c.SetCookie("pos_lang", lang, 60*60*24*365, "/", "", false, false)
		return lang
	}
	if cookie, err := c.Cookie("pos_lang"); err == nil && (cookie == "ar" || cookie == "en") {
		return cookie
	}
	accept := strings.ToLower(c.GetHeader("Accept-Language"))
	if strings.HasPrefix(accept, "ar") || strings.Contains(accept, ",ar") {
		return "ar"
	}
	return "en"
}

// siteOrigins holds the absolute product origins the pages link to. They follow
// the configured POS_DOMAIN / SAAS_DOMAIN so a staging deployment links to its
// own hosts, and fall back to the production defaults.
type siteOrigins struct{ POS, SaaS string }

func siteOrigin(domain, fallback string) string {
	host := config.NormalizeHost(domain)
	if host == "" {
		host = config.NormalizeHost(fallback)
	}
	return "https://" + host
}

func renderSite(c *gin.Context, page string, plans []planCard, origins siteOrigins) {
	lang := pickSiteLang(c)
	dir := "ltr"
	if lang == "ar" {
		dir = "rtl"
	}
	data := siteTemplateData{
		Page: page, Lang: lang, Dir: dir,
		Year: time.Now().Year(), T: siteStrings[lang], Plans: plans,
		POSURL: origins.POS, SaaSURL: origins.SaaS,
	}
	if page == "index" {
		stringsT := siteStrings[lang]
		for _, a := range galleryAssets {
			data.Gallery = append(data.Gallery, siteGalleryItem{
				Src:     "/screenshots/" + a.Name,
				Caption: stringsT[a.Key],
				Width:   a.Width,
				Height:  a.Height,
			})
		}
		data.Modules = buildModules(lang)
	}
	if page == "pricing" {
		data.Compare, data.ComparePlans = buildCompare(lang, plans)
	}
	c.Header("Vary", "Accept-Language")
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := siteTemplates.ExecuteTemplate(c.Writer, page, data); err != nil {
		c.String(http.StatusInternalServerError, "render error")
	}
}

// registerSitePages mounts the public brand, pricing and privacy pages. They
// need no auth; pricing and the landing preview render from the platform plans
// when the pool is available and fall back to the seeded catalog otherwise.
//
// The root page is surface-aware: xamltech.com serves the XAMLtech company
// home, posgo.xamltech.com serves the POS.Go product landing, and
// api.xamltech.com sends visitors straight to the console at /admin/. With
// surface routing disabled every host keeps rendering the POS.Go landing.
func registerSitePages(engine *gin.Engine, pool *pgxpool.Pool, routing config.SurfaceRouting) {
	origins := siteOrigins{
		POS:  siteOrigin(routing.POSDomain, config.DefaultPOSDomain),
		SaaS: siteOrigin(routing.SaaSDomain, config.DefaultSaaSDomain),
	}
	engine.GET("/", func(c *gin.Context) {
		switch httptransport.Surface(c) {
		case config.SurfaceCompany:
			renderSite(c, "company", nil, origins)
			return
		case config.SurfaceSaaS:
			c.Redirect(http.StatusFound, "/admin/")
			return
		}
		renderSite(c, "index", loadPlanCards(c.Request.Context(), pool, pickSiteLang(c)), origins)
	})
	engine.GET("/pricing", func(c *gin.Context) {
		renderSite(c, "pricing", loadPlanCards(c.Request.Context(), pool, pickSiteLang(c)), origins)
	})
	engine.GET("/private", func(c *gin.Context) {
		renderSite(c, "privacy", nil, origins)
	})
	// Public data-deletion request page. Google Play requires a self-service
	// deletion path for apps that can create an account, and "Create store" on
	// the sign-in screen provisions a trial tenant.
	engine.GET("/delete-account", func(c *gin.Context) {
		renderSite(c, "delete", nil, origins)
	})
	// The landing gallery screenshots, served as static PNGs with long-lived
	// cache headers so the HTML stays light and repeat visits are instant.
	engine.GET("/screenshots/:name", func(c *gin.Context) {
		png, ok := screenshotByName(c.Param("name"))
		if !ok {
			c.String(http.StatusNotFound, "not found")
			return
		}
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("Content-Type", "image/png")
		_, _ = c.Writer.Write(png)
	})
}

// loadPlanCards reads the active plans from the platform table. It degrades
// gracefully (nil pool, query error, or empty catalog) to the seeded defaults
// so the page — and the offline OpenAPI generator — always renders.
func loadPlanCards(ctx context.Context, pool *pgxpool.Pool, lang string) []planCard {
	if pool == nil {
		return fallbackPlanCards(lang)
	}
	qctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := pool.Query(qctx, `SELECT code, name, description, price_minor, billing_period, features, max_users, max_products
		FROM plans WHERE is_active ORDER BY price_minor, code`)
	if err != nil {
		return fallbackPlanCards(lang)
	}
	defer rows.Close()
	cards := make([]planCard, 0)
	for rows.Next() {
		var code, name, desc, period string
		var priceMinor int64
		var features []byte
		var maxUsers, maxProducts int
		if err := rows.Scan(&code, &name, &desc, &priceMinor, &period, &features, &maxUsers, &maxProducts); err != nil {
			return fallbackPlanCards(lang)
		}
		var keys []string
		_ = json.Unmarshal(features, &keys)
		cards = append(cards, buildPlanCard(lang, code, name, desc, priceMinor, period, keys, maxUsers, maxProducts))
	}
	if rows.Err() != nil || len(cards) == 0 {
		return fallbackPlanCards(lang)
	}
	return cards
}

type fallbackPlan struct {
	Code, Name, Desc string
	PriceMinor       int64
	Period           string
	Features         []string
	MaxUsers         int
	MaxProducts      int
}

var fallbackPlans = []fallbackPlan{
	{"starter", "Starter", "Solo store getting started", 0, "monthly",
		[]string{"pos.basic", "inventory.basic", "dashboard.basic"}, 3, 100},
	{"business", "Business", "Growing multi-cashier store", 29900, "monthly",
		[]string{"pos.basic", "inventory.advanced", "dashboard.advanced", "restaurant", "loyalty"}, 10, 1000},
	{"enterprise", "Enterprise", "Unlimited stores and verticals", 99900, "monthly",
		[]string{"pos.basic", "inventory.advanced", "dashboard.advanced", "restaurant", "pharmacy", "textile", "loyalty", "sync.multi_device"}, 0, 0},
}

func fallbackPlanCards(lang string) []planCard {
	cards := make([]planCard, 0, len(fallbackPlans))
	for i, p := range fallbackPlans {
		card := buildPlanCard(lang, p.Code, p.Name, p.Desc, p.PriceMinor, p.Period, p.Features, p.MaxUsers, p.MaxProducts)
		card.Featured = i == 1
		cards = append(cards, card)
	}
	return cards
}

func buildPlanCard(lang, code, name, desc string, priceMinor int64, period string, features []string, maxUsers, maxProducts int) planCard {
	labelName, labelDesc := name, desc
	if meta, ok := planTranslations[lang][code]; ok {
		if meta.Name != "" {
			labelName = meta.Name
		}
		if meta.Desc != "" {
			labelDesc = meta.Desc
		}
	}
	labels := make([]string, 0, len(features)+2)
	for _, key := range features {
		labels = append(labels, featureLabel(lang, key))
	}
	labels = append(labels, limitLabel(lang, "users", maxUsers), limitLabel(lang, "products", maxProducts))
	return planCard{
		Code:        code,
		Name:        labelName,
		Description: labelDesc,
		Price:       formatPlanPrice(priceMinor),
		Currency:    siteStrings[lang]["currency"],
		Period:      periodLabel(lang, period),
		Features:    labels,
		Keys:        features,
	}
}

// featureLabel localizes a plan feature key, falling back to the raw key.
func featureLabel(lang, key string) string {
	if label, ok := featureTranslations[lang][key]; ok {
		return label
	}
	return key
}

// buildCompare derives the /pricing comparison table from the feature keys the
// plans actually expose: one row per distinct feature (in first-encountered
// order), a boolean per plan aligned to the given plan order, and the plan
// display names as column headers.
func buildCompare(lang string, plans []planCard) ([]compareRow, []string) {
	names := make([]string, 0, len(plans))
	rows := make([]compareRow, 0)
	seen := make(map[string]struct{})
	for _, p := range plans {
		names = append(names, p.Name)
		for _, k := range p.Keys {
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			rows = append(rows, compareRow{Key: k, Label: featureLabel(lang, k), Values: make([]bool, len(plans))})
		}
	}
	for i, p := range plans {
		for _, k := range p.Keys {
			for ri := range rows {
				if rows[ri].Key == k {
					rows[ri].Values[i] = true
				}
			}
		}
	}
	return rows, names
}

func formatPlanPrice(minor int64) string {
	if minor%100 == 0 {
		return strconv.FormatInt(minor/100, 10)
	}
	return fmt.Sprintf("%.2f", float64(minor)/100)
}

func periodLabel(lang, period string) string {
	if period == "yearly" {
		if lang == "ar" {
			return "سنة"
		}
		return "yr"
	}
	if lang == "ar" {
		return "شهر"
	}
	return "mo"
}

func limitLabel(lang, kind string, n int) string {
	if n <= 0 {
		return siteStrings[lang]["plan"+strings.Title(kind)] + ": " + siteStrings[lang]["planUnlimited"]
	}
	return fmt.Sprintf("%s: %d", siteStrings[lang]["plan"+strings.Title(kind)], n)
}
