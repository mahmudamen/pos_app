package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Public brand, pricing and privacy pages served at the API root. nginx on
// api.xamltech.com proxies everything except /admin/ (the React admin SPA) and
// /.well-known/ to this app, so GET / is the branded landing visitors see when
// they open the domain, GET /pricing lists the subscription plans, and
// GET /private is the privacy policy linked from the Android app.
//
// The visual system is the POS.Go "Nocturne" brand (deep night canvas + soft
// light wells + hairline-divided sections) with emerald kept as the single
// interactive accent (CTAs, links, focus). Every page is bilingual
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

func renderSite(c *gin.Context, page string, plans []planCard) {
	lang := pickSiteLang(c)
	dir := "ltr"
	if lang == "ar" {
		dir = "rtl"
	}
	data := siteTemplateData{
		Page: page, Lang: lang, Dir: dir,
		Year: time.Now().Year(), T: siteStrings[lang], Plans: plans,
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
func registerSitePages(engine *gin.Engine, pool *pgxpool.Pool) {
	engine.GET("/", func(c *gin.Context) {
		renderSite(c, "index", loadPlanCards(c.Request.Context(), pool, pickSiteLang(c)))
	})
	engine.GET("/pricing", func(c *gin.Context) {
		renderSite(c, "pricing", loadPlanCards(c.Request.Context(), pool, pickSiteLang(c)))
	})
	engine.GET("/private", func(c *gin.Context) {
		renderSite(c, "privacy", nil)
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
