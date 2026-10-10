package mainsite

// siteTemplates holds every Go template for the company website. Templates are
// parsed once at startup and executed per request. Bilingual values are
// resolved through the t helper, so both languages always share one layout.

const siteTemplates = `{{define "site"}}<!doctype html>
<html lang="{{.HTMLang}}" dir="{{.Dir}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{t .Page.Title .Lang}} · {{.Brand}}</title>
<meta name="description" content="{{t .Page.Meta .Lang}}">
{{if .Page.Robots}}<meta name="robots" content="{{.Page.Robots}}">{{else}}<meta name="robots" content="index,follow">{{end}}
<meta name="theme-color" content="#0f8f95">
<meta property="og:type" content="website">
<meta property="og:site_name" content="{{.Brand}}">
<meta property="og:title" content="{{t .Page.Title .Lang}} · {{.Brand}}">
<meta property="og:description" content="{{t .Page.Meta .Lang}}">
<meta property="og:url" content="{{.Canonical}}">
<meta property="og:locale" content="{{if eq .HTMLang "ar"}}ar_EG{{else}}en_US{{end}}">
<meta name="twitter:card" content="summary_large_image">
<link rel="canonical" href="{{.Canonical}}">
<link rel="alternate" hreflang="en" href="{{.EnHref}}">
<link rel="alternate" hreflang="ar" href="{{.ArHref}}">
<link rel="alternate" hreflang="x-default" href="{{.EnHref}}">
<link rel="icon" href="/favicon.svg" type="image/svg+xml">
<link rel="apple-touch-icon" href="/favicon.svg">
<style>{{.CSS}}</style>
<script type="application/ld+json">{{.OrgJSON}}</script>
</head>
<body>
{{template "header" .}}
<main>
{{template "hero" .}}
{{range .Page.Sections}}{{template "sec" (dict "S" . "Lang" $.Lang)}}{{end}}
{{if .Page.FAQ}}{{template "faq" (dict "Items" .Page.FAQ "Lang" .Lang)}}{{end}}
{{if .Page.Final.Kind}}{{template "sec" (dict "S" .Page.Final "Lang" .Lang)}}{{end}}
</main>
{{template "footer" .}}
<a class="wa-float" href="{{.Whatsapp}}" target="_blank" rel="noopener" aria-label="WhatsApp">💬</a>
</body>
</html>
{{end}}

{{define "header"}}<header class="site-header"><div class="container"><nav class="nav">
<a class="brand" href="{{langURL .Lang "/"}}"><span class="brand-mark">X</span><span>{{.Brand}}<span class="brand-sub">{{t .BrandSub .Lang}}</span></span></a>
<input type="checkbox" id="nav-toggle" aria-label="Menu">
<label class="nav-burger" for="nav-toggle"><span></span><span></span><span></span></label>
<div class="nav-links">
{{range .Nav}}{{if .Items}}
<div class="dropdown"><button type="button">{{t .Label $.Lang}}</button>
<div class="dropdown-menu">{{range .Items}}<a href="{{langURL $.Lang .Href}}">{{t .Label $.Lang}}</a>{{end}}</div></div>
{{else}}
<a href="{{langURL $.Lang .Href}}"{{if eq .Key $.Active}} class="active"{{end}}>{{t .Label $.Lang}}</a>
{{end}}{{end}}
<a class="btn btn-primary nav-cta" href="{{langURL .Lang "/request-demo"}}">{{t .DemoCTA .Lang}}</a>
<a class="nav-links-lang" href="{{.AltHref}}" title="{{t .AltLabel .Lang}}">{{t .AltLabel .Lang}}</a>
</div>
</nav></div></header>{{end}}

{{define "hero"}}<section class="hero"><div class="container">
{{if .Page.Eyebrow}}<span class="eyebrow">{{t .Page.Eyebrow .Lang}}</span>{{end}}
<h1>{{t .Page.H1 .Lang}}</h1>
<p class="lead">{{t .Page.Lead .Lang}}</p>
{{if .Page.HeroCTAs}}<div class="hero-ctas">{{range .Page.HeroCTAs}}{{template "btn" (dict "L" $.Lang "B" .)}}{{end}}</div>{{end}}
{{if .Page.HeroTiles}}<div class="hero-tiles">{{range .Page.HeroTiles}}<span>{{.Icon}} {{t .Title $.Lang}}</span>{{end}}</div>{{end}}
{{if .Page.HeroNote}}<p class="hero-note">{{t .Page.HeroNote $.Lang}}</p>{{end}}
</div></section>{{end}}

{{define "btn"}}<a class="btn {{if eq .B.Kind "primary"}}btn-primary{{else if eq .B.Kind "wa"}}btn-wa{{else}}btn-ghost{{end}}" href="{{.B.Href}}"{{if .B.Ext}} target="_blank" rel="noopener"{{end}}>{{t .B.Label .L}}</a>{{end}}

{{define "sec-head"}}{{if .S.Label}}<span class="sec-label">{{t .S.Label .Lang}}</span>{{end}}{{if .S.Title}}<div class="sec-head"><h2>{{t .S.Title .Lang}}</h2>{{if .S.Lead}}<p>{{t .S.Lead .Lang}}</p>{{end}}</div>{{end}}{{end}}

{{define "sec"}}{{with .S}}<section class="section">
<div class="container">
{{if eq (printf "%s" .Kind) "stats"}}{{template "sec_stats" (dict "S" . "Lang" $.Lang)}}{{else}}{{template "sec-head" (dict "S" . "Lang" $.Lang)}}{{end}}
{{if eq (printf "%s" .Kind) "features"}}{{template "sec_features" (dict "S" . "Lang" $.Lang)}}{{end}}
{{if eq (printf "%s" .Kind) "cards"}}{{template "sec_cards" (dict "S" . "Lang" $.Lang)}}{{end}}
{{if eq (printf "%s" .Kind) "steps"}}{{template "sec_steps" (dict "S" . "Lang" $.Lang)}}{{end}}
{{if eq (printf "%s" .Kind) "plans"}}{{template "sec_plans" (dict "S" . "Lang" $.Lang)}}{{end}}
{{if eq (printf "%s" .Kind) "prose"}}{{template "sec_prose" (dict "S" . "Lang" $.Lang)}}{{end}}
{{if eq (printf "%s" .Kind) "split"}}{{template "sec_split" (dict "S" . "Lang" $.Lang)}}{{end}}
{{if eq (printf "%s" .Kind) "faq"}}{{template "sec_faq" (dict "S" . "Lang" $.Lang)}}{{end}}
{{if eq (printf "%s" .Kind) "cta"}}{{template "sec_cta" (dict "S" . "Lang" $.Lang)}}{{end}}
{{if .Links}}<div class="section-ctas">{{range .Links}}{{template "btn" (dict "L" $.Lang "B" .)}}{{end}}</div>{{end}}
</div></section>{{end}}{{end}}

{{define "sec_features"}}<div class="grid grid-3">{{range .S.Features}}<div class="feature">
<div class="ico">{{.Icon}}</div><h3>{{t .Title $.Lang}}</h3>{{if .Body}}<p>{{t .Body $.Lang}}</p>{{end}}
</div>{{end}}</div>{{end}}

{{define "sec_cards"}}<div class="grid grid-3">{{range .S.Cards}}<div class="card">
{{with .Badge}}<span class="badge {{if eq .En "Free"}}free{{else if eq .En "Best seller"}}hot{{end}}">{{t . $.Lang}}</span>{{end}}
<div class="ico">{{.Icon}}</div>
{{if .Eyebrow}}<p class="eyebrow">{{t .Eyebrow $.Lang}}</p>{{end}}
<h3>{{t .Title $.Lang}}</h3>
{{if .Body}}<p>{{t .Body $.Lang}}</p>{{end}}
{{if .Meta}}<div class="meta">{{range .Meta}}<span>{{t . $.Lang}}</span>{{end}}</div>{{end}}
{{if .Price.En}}<div class="price-row"><span class="price">{{t .Price $.Lang}}</span>{{if .Rating}}<span class="rating">★ {{.Rating}}</span>{{end}}</div>{{end}}
{{if .Href}}<a class="card-read" href="{{.Href}}">{{if .CTA}}{{t .CTA $.Lang}}{{else}}{{t (moreText) $.Lang}}{{end}}</a>{{end}}
</div>{{end}}</div>{{end}}

{{define "sec_steps"}}<div class="steps">{{range .S.Steps}}<div class="step"><div class="num"></div><div>
<h3>{{t .Title $.Lang}}</h3><p>{{t .Body $.Lang}}</p></div></div>{{end}}</div>{{end}}

{{define "sec_plans"}}<div class="plans">{{range .S.Plans}}<div class="plan{{if .Featured}} featured{{end}}">
{{with .Badge}}<span class="badge">{{t . $.Lang}}</span>{{end}}
<h3>{{t .Name $.Lang}}</h3>
<p class="desc">{{t .Desc $.Lang}}</p>
<div class="price-row"><span class="price">{{t .Price $.Lang}}</span>{{if .Currency}}<span class="per">{{.Currency}}{{if .Period}} · {{t .Period $.Lang}}{{end}}</span>{{end}}</div>
<ul>{{range .Features}}<li>{{t . $.Lang}}</li>{{end}}</ul>
{{if .Href}}<a class="btn {{if .Featured}}btn-primary{{else}}btn-ghost{{end}}" href="{{.Href}}">{{t .CTA $.Lang}}</a>{{end}}
</div>{{end}}</div>{{end}}

{{define "sec_stats"}}<div class="stats">{{range .S.Stats}}<div class="stat"><div class="v">{{t .Value $.Lang}}</div><div class="l">{{t .Label $.Lang}}</div></div>{{end}}</div>{{end}}

{{define "sec_prose"}}<div class="prose">{{range .S.Blocks}}{{if .Heading}}<h3>{{t .Heading $.Lang}}</h3>{{end}}{{range .Body}}<p>{{t . $.Lang}}</p>{{end}}{{if .List}}<ul>{{range .List}}<li>{{t . $.Lang}}</li>{{end}}</ul>{{end}}{{end}}</div>{{end}}

{{define "sec_split"}}<div class="split">{{range .S.Blocks}}<div class="split-box">{{if .Heading}}<h3>{{t .Heading $.Lang}}</h3>{{end}}{{range .Body}}<p>{{t . $.Lang}}</p>{{end}}</div>{{end}}</div>{{end}}

{{define "sec_faq"}}<div class="faq">{{range .S.FAQ}}<details><summary>{{t .Q $.Lang}}</summary><div class="a"><p>{{t .A $.Lang}}</p></div></details>{{end}}</div>{{end}}

{{define "sec_cta"}}<section class="section"><div class="container"><div class="cta">
{{if .S.Title}}<h2>{{t .S.Title .Lang}}</h2>{{end}}
{{if .S.Lead}}<p>{{t .S.Lead .Lang}}</p>{{end}}
{{if .S.Links}}<div class="hero-ctas">{{range .S.Links}}{{template "btn" (dict "L" $.Lang "B" .)}}{{end}}</div>{{end}}
</div></div></section>{{end}}

{{define "faq"}}<section class="section"><div class="container">{{template "sec_faq" (dict "S" (dict "FAQ" .Items) "Lang" .Lang)}}</div></section>{{end}}

{{define "footer"}}<footer class="site-footer">
<div class="footer-top"><div class="container"><div class="footer-grid">
<div class="footer-brand">
<a class="brand" href="{{langURL .Lang "/"}}"><span class="brand-mark">X</span><span>{{.Brand}}<span class="brand-sub">{{t .BrandSub .Lang}}</span></span></a>
<p>{{t .BrandTag .Lang}}</p>
<div class="socials">
<a href="{{.Whatsapp}}" target="_blank" rel="noopener" aria-label="WhatsApp">💬</a>
<a href="mailto:{{.Email}}" aria-label="Email">✉️</a>
<a href="{{.Github}}" target="_blank" rel="noopener" aria-label="GitHub">🐙</a>
<a href="{{.OdooApps}}" target="_blank" rel="noopener" aria-label="Odoo Apps">🛒</a>
</div>
</div>
{{range .Footer}}<div class="footer-col"><h4>{{t .Title $.Lang}}</h4>{{range .Links}}<a href="{{langURL $.Lang .Href}}"{{if .Ext}} target="_blank" rel="noopener"{{end}}>{{t .Label $.Lang}}</a>{{end}}</div>{{end}}
</div></div></div>
<div class="footer-bottom"><div class="container">
<span>© {{.Year}} {{.Brand}}. {{t .Rights .Lang}}</span>
<div class="legals"><a href="{{langURL .Lang "/privacy-policy"}}">{{t .Privacy .Lang}}</a><a href="{{langURL .Lang "/terms"}}">{{t .Terms .Lang}}</a></div>
</div></div>
</footer>{{end}}
`
