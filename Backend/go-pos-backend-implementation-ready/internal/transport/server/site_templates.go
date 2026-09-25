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
<meta property="og:site_name" content="{{.T.brand}} — {{.T.tagline}}"/>
<meta property="og:type" content="website"/>
<meta property="og:title" content="{{if eq .Page "pricing"}}{{.T.pricingTitle}}{{else if eq .Page "privacy"}}{{.T.privacyTitle}}{{else}}{{.T.brand}} — {{.T.tagline}}{{end}}"/>
<meta property="og:description" content="{{if eq .Page "pricing"}}{{.T.pricingMeta}}{{else if eq .Page "privacy"}}{{.T.privacyMeta}}{{else}}{{.T.heroLead}}{{end}}"/>
<meta property="og:locale" content="{{if eq .Lang "ar"}}ar_EG{{else}}en_GB{{end}}"/>
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
      <a href="/#copilot">{{.T.navCopilot}}</a>
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
      <div class="hero-copy">
        <span class="pill">{{.T.heroPill}}</span>
        <h1 class="hero-title">{{.T.heroTitle}}</h1>
        <p class="hero-lead">{{.T.heroLead}}</p>
        <div class="hero-actions">
          <a class="btn btn-primary" href="/admin/">{{.T.ctaPrimary}}</a>
          <a class="btn btn-ghost" href="/pricing">{{.T.ctaSecondary}}</a>
        </div>
        <div class="applinks">
          <a class="applink" href="https://play.google.com/store/apps/details?id=com.xamltech.pos_go" target="_blank" rel="noopener">
            <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M3.6 1.8l9.6 10.2L3.6 22.2c-.4-.2-.6-.6-.6-1.1V2.9c0-.5.2-.9.6-1.1zM14.3 12.6L16.8 10l5.6 3.2c.8.5.8 1.7 0 2.2l-5.6 3.2-2.5-2.6a1.7 1.7 0 0 1 0-3.4zM14.3 11.4a1.7 1.7 0 0 0 0-3.4l2.5-2.6L22.4 8a1.6 1.6 0 0 1 0 2.1l.1.1-2.6 2.6-5.6-3.4z"/><path d="M3.6 1.8l9.6 10.2L14.8 10 6.2 1.5z"/></svg>
            <span><b>{{.T.storePlay}}</b><small>{{.T.storePlaySub}}</small></span>
          </a>
          <a class="applink" href="/apk/pos_go.apk">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="7.5" y="8.5" width="9" height="9" rx="2.2"/><path d="M9.5 8.5V6.8a2.5 2.5 0 0 1 5 0v1.7"/><circle cx="10.6" cy="12.4" r="0.8" fill="currentColor"/><circle cx="13.4" cy="12.4" r="0.8" fill="currentColor"/></svg>
            <span><b>{{.T.downloadApk}}</b><small>{{.T.apkHint}}</small></span>
          </a>
        </div>
        <div class="stats">
          <span class="s"><b>{{.T.stat1}}</b><small>{{.T.stat1L}}</small></span>
          <span class="s"><b>{{.T.stat2}}</b><small>{{.T.stat2L}}</small></span>
          <span class="s"><b>{{.T.stat3}}</b><small>{{.T.stat3L}}</small></span>
          <span class="s"><b>{{.T.stat4}}</b><small>{{.T.stat4L}}</small></span>
        </div>
      </div>
    </section>

    <section id="gallery" class="section gallery-section" aria-label="{{.T.secGallery}}">
      <span class="section-label">{{.T.secGallery}}</span>
      <h2 class="section-title">{{.T.galTitle}}</h2>
      <p class="section-sub">{{.T.galLead}}</p>
      <div class="gallery">
        <div class="gal-track" tabindex="0">
          {{range .Gallery}}<figure class="gitem">
            <img src="{{.Src}}" alt="{{.Caption}}" width="{{.Width}}" height="{{.Height}}" loading="lazy" decoding="async"/>
            <figcaption>{{.Caption}}</figcaption>
          </figure>
          {{end}}
        </div>
        <button class="gal-btn prev" type="button" aria-label="{{.T.galPrev}}">‹</button>
        <button class="gal-btn next" type="button" aria-label="{{.T.galNext}}">›</button>
      </div>
      <div class="gal-dots" role="tablist" aria-label="{{.T.secGallery}}">
        {{range .Gallery}}<button class="dot" type="button" role="tab" aria-label="{{.Caption}}"></button>{{end}}
      </div>
    </section>
    <script>
      (function () {
        var track = document.querySelector('.gal-track');
        if (!track) return;
        var items = Array.prototype.slice.call(track.querySelectorAll('.gitem'));
        var dots = Array.prototype.slice.call(document.querySelectorAll('.gal-dots .dot'));
        function stepPx() {
          var gap = parseFloat(getComputedStyle(track).columnGap) || 18;
          return items.length ? items[0].getBoundingClientRect().width + gap : 320;
        }
        function dir() { return getComputedStyle(track).direction === 'rtl' ? -1 : 1; }
        document.querySelectorAll('.gal-btn').forEach(function (b) {
          b.addEventListener('click', function () {
            var k = b.classList.contains('next') ? 1 : -1;
            track.scrollBy({ left: dir() * k * stepPx(), behavior: 'smooth' });
          });
        });
        function mark() {
          var r = track.getBoundingClientRect();
          var mid = r.left + r.width / 2, best = 0, bd = 1e9;
          items.forEach(function (it, i) {
            var ir = it.getBoundingClientRect();
            var d = Math.abs(ir.left + ir.width / 2 - mid);
            if (d < bd) { bd = d; best = i; }
          });
          dots.forEach(function (d, i) {
            d.classList.toggle('is-on', i === best);
            d.setAttribute('aria-selected', i === best ? 'true' : 'false');
          });
        }
        track.addEventListener('scroll', function () {
          clearTimeout(track._t);
          track._t = setTimeout(mark, 60);
        }, { passive: true });
        dots.forEach(function (d, i) {
          d.addEventListener('click', function () {
            if (!items[i]) return;
            var ir = items[i].getBoundingClientRect();
            var r = track.getBoundingClientRect();
            track.scrollBy({ left: ir.left + ir.width / 2 - (r.left + r.width / 2), behavior: 'smooth' });
          });
        });
        mark();
      })();
    </script>

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
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3.5l1.6 4.4 4.4 1.6-4.4 1.6L12 15.5l-1.6-4.4L6 10.4l4.4-1.6z"/><path d="M18.5 15.5l.7 1.9 1.9.7-1.9.7-.7 1.9-.7-1.9-1.9-.7 1.9-.7z"/></svg>
          <h3>{{.T.f7t}}</h3><p>{{.T.f7d}}</p>
        </div>
      </div>
    </section>

    <section id="copilot" class="section copilot-sec">
      <div class="copt-card">
        <span class="copt-live"><i aria-hidden="true"></i>{{.T.coptPill}}</span>
        <h2 class="copt-title">{{.T.coptTitleLead}} <span class="co-grad">{{.T.coptTitleAccent}}</span></h2>
        <p class="copt-lead">{{.T.coptLead}}</p>
        <div class="copt-stats">
          <div class="copt-stat"><small>{{.T.coptS1L}}</small><b class="c-s1">{{.T.coptS1V}}</b><em>{{.T.coptS1S}}</em></div>
          <div class="copt-stat"><small>{{.T.coptS2L}}</small><b class="c-s2">{{.T.coptS2V}}</b><em>{{.T.coptS2S}}</em></div>
          <div class="copt-stat"><small>{{.T.coptS3L}}</small><b class="c-s3">{{.T.coptS3V}}</b><em>{{.T.coptS3S}}</em></div>
        </div>
      </div>
      <div class="copt-grid">
        <div class="copt-feat">
          <span class="copt-ico"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3.5l1.6 4.4 4.4 1.6-4.4 1.6L12 15.5l-1.6-4.4L6 10.4l4.4-1.6z"/><path d="M18.5 15.5l.7 1.9 1.9.7-1.9.7-.7 1.9-.7-1.9-1.9-.7 1.9-.7z"/></svg></span>
          <h3>{{.T.coptC1t}}</h3><p>{{.T.coptC1d}}</p>
        </div>
        <div class="copt-feat">
          <span class="copt-ico"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 11.2a8.5 8.5 0 0 1-8.5 8.5c-1.4 0-2.7-.3-3.9-.9L3 20l1.2-5.6A8.5 8.5 0 0 1 3 11.2a8.5 8.5 0 0 1 17 0z"/><path d="M8.5 11h.01M12 11h.01M15.5 11h.01"/></svg></span>
          <h3>{{.T.coptC2t}}</h3><p>{{.T.coptC2d}}</p>
        </div>
        <div class="copt-feat">
          <span class="copt-ico"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M13 3L4 14h6l-1 7 9-11h-6z"/></svg></span>
          <h3>{{.T.coptC3t}}</h3><p>{{.T.coptC3d}}</p>
        </div>
      </div>
      <div class="copt-cta-row">
        <span class="copt-cta-wrap">
          <span class="copt-halo" aria-hidden="true"></span>
          <a class="copt-cta" href="#plans"><span class="copt-label">{{.T.coptCta}}</span><span class="copt-arrow" aria-hidden="true">→</span><span class="copt-shimmer" aria-hidden="true"></span></a>
        </span>
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
      <a href="/#copilot">{{.T.navCopilot}}</a>
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
    <div class="footer-links">
      <a href="https://play.google.com/store/apps/details?id=com.xamltech.pos_go" target="_blank" rel="noopener">{{.T.storePlay}}</a>
      <a href="/apk/pos_go.apk">{{.T.downloadApk}}</a>
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
