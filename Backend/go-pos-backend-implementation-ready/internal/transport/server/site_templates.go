package server

import "html/template"

// Public page templates. Each page is siteHead + a page-specific <main> body +
// siteFoot, all rendered from the same siteTemplateData. Everything (CSS,
// icons, receipt, barcode) is code-native and offline by construction — no
// external fonts, scripts or images.

// siteHead is the shared document head + brand header/nav. Title + meta react
// to .Page; the lang switch keeps the visitor on the current path.
const siteHead = `<!doctype html>
<html lang="{{.Lang}}" dir="{{.Dir}}">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<meta name="color-scheme" content="dark"/>
<meta name="theme-color" content="#0a1320"/>
<meta name="robots" content="{{if eq .Page "privacy"}}noindex{{else}}index, follow{{end}}"/>
<meta name="description" content="{{if eq .Page "pricing"}}{{.T.pricingMeta}}{{else if eq .Page "privacy"}}{{.T.privacyMeta}}{{else}}{{.T.heroLead}}{{end}}"/>
<link rel="icon" type="image/svg+xml" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 96 96'%3E%3Crect x='4' y='4' width='88' height='88' rx='22' fill='%230f2233' stroke='%2322c55e' stroke-width='4'/%3E%3Ctext x='48' y='68' text-anchor='middle' font-family='Verdana' font-size='52' font-weight='700' fill='%2322c55e'%3EP%3C/text%3E%3C/svg%3E"/>
<title>{{if eq .Page "pricing"}}{{.T.pricingTitle}} — {{.T.brand}}{{else if eq .Page "privacy"}}{{.T.privacyTitle}} — {{.T.brand}}{{else}}{{.T.brand}} — {{.T.tagline}}{{end}}</title>
<style>` + siteCSS + `</style>
</head>
<body>
<header class="site-header">
  <nav class="nav container" aria-label="{{.T.navLabel}}">
    <a class="brand" href="/">
      <svg viewBox="0 0 96 96" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <rect x="4" y="4" width="88" height="88" rx="22" fill="none" stroke="currentColor" stroke-width="6"/>
        <circle cx="69" cy="27" r="7" fill="currentColor"/>
        <text x="48" y="68" text-anchor="middle" font-family="Verdana, sans-serif" font-size="52" font-weight="700" fill="currentColor">P</text>
      </svg>
      <span><b>{{.T.brand}}</b><small>{{.T.brandSub}}</small></span>
    </a>
    <div class="nav-links">
      <a href="/#features">{{.T.navFeatures}}</a>
      <a href="/#verticals">{{.T.navVerticals}}</a>
      <a href="/pricing">{{.T.navPricing}}</a>
      <a href="/private">{{.T.navPrivacy}}</a>
      <a class="lang" href="?lang={{if eq .Lang "ar"}}en{{else}}ar{{end}}">{{.T.navLang}}</a>
      <a class="btn btn-primary btn-sm" href="/admin/">{{.T.navAdmin}}</a>
    </div>
  </nav>
</header>`

// sitePlansGrid is the shared plan-card markup (interactive cards, so cards are
// justified). Used on the landing preview and the /pricing page.
const sitePlansGrid = `{{range .Plans}}
<div class="plan{{if .Featured}} featured{{end}}">
  {{if .Featured}}<span class="plan-tag">{{$.T.planMost}}</span>{{end}}
  <h3>{{.Name}}</h3>
  <p class="desc">{{.Description}}</p>
  <div class="price"><span class="amount">{{.Price}}</span><span class="per">{{.Currency}} &nbsp;/ {{.Period}}</span></div>
  <ul>{{range .Features}}<li>{{.}}</li>{{end}}</ul>
  <div class="fill"><a class="btn {{if .Featured}}btn-primary{{else}}btn-ghost{{end}} btn-block" href="/admin/">{{$.T.ctaPlan}}</a></div>
</div>
{{end}}`

const siteIndexBody = `<main>
  <div class="container">
    <section class="hero">
      <div>
        <span class="pill">{{.T.heroPill}}</span>
        <h1 class="hero-title">{{.T.heroTitle}}</h1>
        <p class="hero-lead">{{.T.heroLead}}</p>
        <div class="hero-actions">
          <a class="btn btn-primary" href="/admin/">{{.T.ctaPrimary}}</a>
          <a class="btn btn-ghost" href="/pricing">{{.T.ctaSecondary}}</a>
        </div>
        <div class="stats">
          <span class="s"><b>{{.T.stat1}}</b><small>{{.T.stat1L}}</small></span>
          <span class="s"><b>{{.T.stat2}}</b><small>{{.T.stat2L}}</small></span>
          <span class="s"><b>{{.T.stat3}}</b><small>{{.T.stat3L}}</small></span>
          <span class="s"><b>{{.T.stat4}}</b><small>{{.T.stat4L}}</small></span>
        </div>
      </div>
      <aside class="receipt" aria-label="{{.T.recOrder}}">
        <div class="receipt-head"><span>{{.T.brand}}</span><span>{{.T.recOrder}}</span></div>
        <p class="receipt-date">{{.T.recDate}}</p>
        <span class="stamp">{{.T.recStamp}}</span>
        <hr class="receipt-rule"/>
        <div class="receipt-row"><span class="receipt-item-name">{{.T.recItem1}}</span><span>{{.T.currency}} {{.T.recVal1}}</span></div>
        <div class="receipt-row"><span class="receipt-item-name">{{.T.recItem2}}</span><span>{{.T.currency}} {{.T.recVal2}}</span></div>
        <div class="receipt-row"><span class="receipt-item-name">{{.T.recItem3}}</span><span>{{.T.currency}} {{.T.recVal3}}</span></div>
        <hr class="receipt-rule"/>
        <div class="receipt-row muted"><span>{{.T.recSub}}</span><span>{{.T.currency}} 275.00</span></div>
        <div class="receipt-row muted"><span>{{.T.recTip}}</span><span>{{.T.currency}} 15.00</span></div>
        <hr class="receipt-rule"/>
        <div class="receipt-total"><span>{{.T.recTotal}}</span><span class="money">{{.T.currency}} 290.00</span></div>
        <div class="receipt-row muted"><span>{{.T.recPaid}}</span><span>{{.T.currency}} 290.00</span></div>
        <div class="receipt-row muted"><span class="ok-line"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12.5l4.5 4.5L19 7.5"/></svg>{{.T.recPoints}}</span></div>
        <p class="receipt-thanks">{{.T.recThanks}}</p>
        <svg class="barcode" viewBox="0 0 240 28" fill="currentColor" preserveAspectRatio="none" aria-hidden="true">
          <rect x="0" y="0" width="4" height="28"/><rect x="8" y="4" width="2" height="24"/><rect x="14" y="0" width="5" height="28"/><rect x="23" y="6" width="2" height="22"/><rect x="29" y="0" width="3" height="28"/><rect x="36" y="2" width="5" height="26"/><rect x="45" y="6" width="2" height="22"/><rect x="51" y="0" width="4" height="28"/><rect x="59" y="3" width="3" height="25"/><rect x="66" y="0" width="2" height="28"/><rect x="72" y="5" width="5" height="23"/><rect x="81" y="0" width="3" height="28"/><rect x="88" y="2" width="4" height="26"/><rect x="96" y="6" width="2" height="22"/><rect x="102" y="0" width="5" height="28"/><rect x="111" y="4" width="2" height="24"/><rect x="117" y="0" width="3" height="28"/><rect x="124" y="2" width="5" height="26"/><rect x="133" y="6" width="2" height="22"/><rect x="139" y="0" width="4" height="28"/><rect x="147" y="3" width="3" height="25"/><rect x="154" y="0" width="2" height="28"/><rect x="160" y="5" width="5" height="23"/><rect x="169" y="0" width="3" height="28"/><rect x="176" y="2" width="4" height="26"/><rect x="184" y="6" width="2" height="22"/><rect x="190" y="0" width="5" height="28"/><rect x="199" y="4" width="2" height="24"/><rect x="205" y="0" width="3" height="28"/><rect x="212" y="2" width="5" height="26"/><rect x="221" y="6" width="2" height="22"/><rect x="227" y="0" width="4" height="28"/><rect x="235" y="3" width="5" height="25"/>
        </svg>
      </aside>
    </section>

    <section id="how" class="section">
      <span class="section-label">{{.T.secHow}}</span>
      <h2 class="section-title">{{.T.howTitle}}</h2>
      <p class="section-sub">{{.T.howLead}}</p>
      <ol class="steps">
        <li><span class="step-num" aria-hidden="true">1</span><div><h3>{{.T.how1t}}</h3><p>{{.T.how1d}}</p></div></li>
        <li><span class="step-num" aria-hidden="true">2</span><div><h3>{{.T.how2t}}</h3><p>{{.T.how2d}}</p></div></li>
        <li><span class="step-num" aria-hidden="true">3</span><div><h3>{{.T.how3t}}</h3><p>{{.T.how3d}}</p></div></li>
      </ol>
    </section>

    <section id="features" class="section">
      <span class="section-label">{{.T.secFeatures}}</span>
      <h2 class="section-title">{{.T.featTitle}}</h2>
      <p class="section-sub">{{.T.featLead}}</p>
      <div class="feature-grid">
        <div class="feature">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3 9a15 15 0 0 1 18 0"/><path d="M7.5 13a9 9 0 0 1 9 0"/><circle cx="12" cy="17.5" r="0.8" fill="currentColor"/><path d="M4 4l16 16"/></svg>
          <h3>{{.T.f1t}}</h3><p>{{.T.f1d}}</p>
        </div>
        <div class="feature">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3.5" y="6" width="17" height="12" rx="2"/><path d="M3.5 10h17"/><path d="M8 15h5"/></svg>
          <h3>{{.T.f2t}}</h3><p>{{.T.f2d}}</p>
        </div>
        <div class="feature">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3l8 4.4v9.2L12 21l-8-4.4V7.4z"/><path d="M12 12l8-4.6M12 12L4 7.4M12 12v9"/></svg>
          <h3>{{.T.f3t}}</h3><p>{{.T.f3d}}</p>
        </div>
        <div class="feature">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="6" width="18" height="13" rx="2"/><path d="M3 11h18"/><path d="M6.5 16h4"/><path d="M18 16.8l2.4-2.4M18 16.8V19"/></svg>
          <h3>{{.T.f4t}}</h3><p>{{.T.f4d}}</p>
        </div>
        <div class="feature">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 20.5S4.5 15.5 3 11.2A5.1 5.1 0 0 1 12 7.1a5.1 5.1 0 0 1 9 4.1C19.5 15.5 12 20.5 12 20.5z"/></svg>
          <h3>{{.T.f5t}}</h3><p>{{.T.f5d}}</p>
        </div>
        <div class="feature">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3.5l6.5 2.6v5c0 4.3-2.7 7.6-6.5 9.4-3.8-1.8-6.5-5.1-6.5-9.4v-5z"/><path d="M9.2 9.2h.01M14.8 14.8h.01M10.6 13.4l2.8-2.8"/></svg>
          <h3>{{.T.f6t}}</h3><p>{{.T.f6d}}</p>
        </div>
      </div>
    </section>

    <section id="verticals" class="section">
      <span class="section-label">{{.T.secVerticals}}</span>
      <h2 class="section-title">{{.T.vitTitle}}</h2>
      <p class="section-sub">{{.T.vitLead}}</p>
      <div class="vgrid">
        <span class="v">{{.T.vit1}}</span><span class="v">{{.T.vit2}}</span><span class="v">{{.T.vit3}}</span>
        <span class="v">{{.T.vit4}}</span><span class="v">{{.T.vit5}}</span><span class="v">{{.T.vit6}}</span>
        <span class="v">{{.T.vit7}}</span><span class="v">{{.T.vit8}}</span><span class="v">{{.T.vit9}}</span>
        <span class="v">{{.T.vit10}}</span><span class="v">{{.T.vit11}}</span><span class="v">{{.T.vit12}}</span>
        <span class="v">{{.T.vit13}}</span><span class="v">{{.T.vit14}}</span><span class="v">{{.T.vit15}}</span>
        <span class="v">{{.T.vit16}}</span><span class="v">{{.T.vit17}}</span>
      </div>
    </section>

    <section id="faq" class="section">
      <span class="section-label">{{.T.secFAQ}}</span>
      <h2 class="section-title">{{.T.faqTitle}}</h2>
      <div class="faqs">
        <details><summary>{{.T.q1}}</summary><p>{{.T.a1}}</p></details>
        <details><summary>{{.T.q2}}</summary><p>{{.T.a2}}</p></details>
        <details><summary>{{.T.q3}}</summary><p>{{.T.a3}}</p></details>
        <details><summary>{{.T.q4}}</summary><p>{{.T.a4}}</p></details>
        <details><summary>{{.T.q5}}</summary><p>{{.T.a5}}</p></details>
      </div>
    </section>

    <section id="plans" class="section">
      <span class="section-label">{{.T.secPlans}}</span>
      <h2 class="section-title">{{.T.plansTitle}}</h2>
      <p class="section-sub">{{.T.plansLead}}</p>
      <div class="plans">` + sitePlansGrid + `</div>
      <p class="section-note"><a href="/pricing#compare">{{.T.plansCTA}}</a></p>
    </section>
  </div>
</main>`

const sitePricingBody = `<main>
  <div class="container">
    <div class="page-head">
      <span class="pill">{{.T.secPlans}}</span>
      <h1 class="page-title">{{.T.pricingTitle}}</h1>
      <p class="section-sub">{{.T.pricingLead}}</p>
    </div>
    <section class="section">
      <div class="plans">` + sitePlansGrid + `</div>
      <p class="section-note">{{.T.priceNote}}</p>
    </section>
    <section id="compare" class="section">
      <span class="section-label">{{.T.secCompare}}</span>
      <h2 class="section-title">{{.T.compareTitle}}</h2>
      <p class="section-sub">{{.T.compareLead}}</p>
      <div class="compare-scroll">
        <table class="compare">
          <thead>
            <tr><th scope="col">{{.T.compareFeature}}</th>{{range .ComparePlans}}<th scope="col">{{.}}</th>{{end}}</tr>
          </thead>
          <tbody>
          {{range .Compare}}
            <tr><th scope="row">{{.Label}}</th>{{range .Values}}<td>{{if .}}<svg class="match" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12.5l4.5 4.5L19 7.5"/></svg>{{else}}<span class="dash" aria-hidden="true">—</span>{{end}}</td>{{end}}</tr>
          {{end}}
          </tbody>
        </table>
      </div>
    </section>
    <section id="faq" class="section">
      <span class="section-label">{{.T.secFAQ}}</span>
      <h2 class="section-title">{{.T.faqTitle}}</h2>
      <div class="faqs">
        <details><summary>{{.T.q1}}</summary><p>{{.T.a1}}</p></details>
        <details><summary>{{.T.q2}}</summary><p>{{.T.a2}}</p></details>
        <details><summary>{{.T.q3}}</summary><p>{{.T.a3}}</p></details>
        <details><summary>{{.T.q4}}</summary><p>{{.T.a4}}</p></details>
        <details><summary>{{.T.q5}}</summary><p>{{.T.a5}}</p></details>
      </div>
      <p class="section-note"><a href="/">{{.T.backHome}}</a></p>
    </section>
  </div>
</main>`

const sitePrivacyBody = `<main>
  <div class="container">
    <div class="page-head">
      <span class="section-label">{{.T.navPrivacy}}</span>
      <h1 class="page-title">{{.T.privacyTitle}}</h1>
      <p class="section-sub">{{.T.privacyUpdated}}</p>
    </div>
    <section class="section prose">
      <p>{{.T.privacyIntro}}</p>
      <h2>{{.T.s1t}}</h2><p>{{.T.s1b}}</p>
      <h2>{{.T.s2t}}</h2>
      <ul><li>{{.T.s2a}}</li><li>{{.T.s2b}}</li><li>{{.T.s2c}}</li><li>{{.T.s2d}}</li></ul>
      <h2>{{.T.s3t}}</h2><p>{{.T.s3b}}</p>
      <h2>{{.T.s4t}}</h2><p>{{.T.s4b}}</p>
      <h2>{{.T.s5t}}</h2><p>{{.T.s5b}}</p>
      <h2>{{.T.s6t}}</h2><p>{{.T.s6b}}</p>
      <h2>{{.T.s7t}}</h2><p>{{.T.s7b}}</p>
      <h2>{{.T.s8t}}</h2><p>{{.T.s8b}}</p>
      <p class="section-note"><a href="/">{{.T.privacyBack}}</a></p>
    </section>
  </div>
</main>`

const siteFoot = `<footer class="site-footer">
  <div class="footer container">
    <div class="footer-brand">
      <b>{{.T.brand}} <small>{{.T.brandSub}}</small></b>
      <p>{{.T.footerTag}}</p>
    </div>
    <div class="footer-links" aria-label="{{.T.navLabel}}">
      <a href="/#features">{{.T.navFeatures}}</a>
      <a href="/#verticals">{{.T.navVerticals}}</a>
      <a href="/#how">{{.T.secHow}}</a>
      <a href="/#faq">{{.T.secFAQ}}</a>
    </div>
    <div class="footer-links">
      <a href="/pricing">{{.T.navPricing}}</a>
      <a href="/private">{{.T.navPrivacy}}</a>
      <a href="/admin/">{{.T.navAdmin}}</a>
      <a href="mailto:{{.T.contact}}">{{.T.contact}}</a>
    </div>
    <div class="footer-text">
      <span>© {{.Year}} XAMLtech · {{.T.footerRights}}</span>
    </div>
  </div>
</footer>
</body>
</html>`

// siteTemplates holds the three public pages keyed by their renderSite page
// name. sitePlansGrid is shared by the index + pricing bodies above.
var siteTemplates = parseSiteTemplates()

func parseSiteTemplates() *template.Template {
	t := template.New("site")
	for name, src := range map[string]string{
		"index":   siteHead + siteIndexBody + siteFoot,
		"pricing": siteHead + sitePricingBody + siteFoot,
		"privacy": siteHead + sitePrivacyBody + siteFoot,
	} {
		template.Must(t.New(name).Parse(src))
	}
	return t
}
