package server

// siteCSS is the shared design system for every public POS.Go page. Direction-
// aware (logical properties, so the same file drives LTR English and RTL Arabic),
// dark "Nocturne" brand canvas with the POS.Go emerald reserved for the single
// interactive accent: CTAs, links, focus, active states and the invoice stamp.
// No external fonts or network assets — the stack renders Arabic natively.
const siteCSS = `
:root {
  color-scheme: dark;
  --bg: #0a1320;
  --bg-2: #0d1826;
  --well-teal: rgba(20, 130, 170, 0.16);
  --well-emerald: rgba(34, 197, 94, 0.13);
  --surface: rgba(255, 255, 255, 0.04);
  --surface-2: rgba(255, 255, 255, 0.028);
  --border: rgba(148, 187, 214, 0.16);
  --border-strong: rgba(148, 187, 214, 0.32);
  --text: #eaf2f8;
  --text-2: rgba(234, 242, 248, 0.74);
  --text-3: rgba(234, 242, 248, 0.52);
  --accent: #22c55e;
  --accent-hi: #34d399;
  --accent-hover: #16a34a;
  --accent-dim: rgba(34, 197, 94, 0.13);
  --accent-text: #06290f;
  --radius: 16px;
  --shadow: 0 26px 64px rgba(0, 0, 0, 0.46);
  --font: 'Segoe UI', system-ui, -apple-system, Roboto, 'Helvetica Neue', 'Noto Sans Arabic', Tahoma, sans-serif;
  --mono: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, 'Liberation Mono', monospace;
}
* { box-sizing: border-box; }
html { scroll-behavior: smooth; }
@media (prefers-reduced-motion: reduce) {
  html { scroll-behavior: auto; }
  *, *::before, *::after { animation-duration: 0.001s !important; transition-duration: 0.001s !important; }
}
body {
  margin: 0;
  font-family: var(--font);
  font-size: 16px;
  line-height: 1.6;
  color: var(--text);
  -webkit-font-smoothing: antialiased;
  background:
    radial-gradient(900px 480px at 88% -8%, var(--well-teal), transparent 62%),
    radial-gradient(760px 420px at -12% 108%, var(--well-emerald), transparent 60%),
    linear-gradient(180deg, var(--bg-2) 0%, var(--bg) 48%, var(--bg) 100%);
  background-attachment: fixed;
  min-height: 100vh;
}
a { color: var(--accent); text-decoration: none; }
a:hover { color: var(--accent-hi); }
:focus-visible { outline: 2px solid var(--accent); outline-offset: 3px; }
::selection { background: var(--accent); color: var(--accent-text); }
.container { max-width: 1120px; margin-inline: auto; padding-inline: 24px; }

/* ---------- header / nav ---------- */
.site-header { padding: 18px 0 6px; }
.nav { display: flex; align-items: center; gap: 20px; flex-wrap: wrap; }
.brand {
  display: inline-flex; align-items: center; gap: 11px;
  color: var(--text); font-weight: 700; font-size: 17px; letter-spacing: 0.01em;
}
.brand svg { width: 32px; height: 32px; flex: none; }
.brand b { display: block; line-height: 1.05; }
.brand small {
  display: block; font-size: 10.5px; font-weight: 500;
  color: var(--text-3); letter-spacing: 0.06em; text-transform: uppercase;
}
.nav-links { display: flex; align-items: center; gap: 2px; margin-inline-start: auto; flex-wrap: wrap; }
.nav-links a:not(.btn) {
  color: var(--text-2); font-size: 14.5px; font-weight: 500;
  padding: 8px 12px; border-radius: 10px;
}
.nav-links a:not(.btn):hover { color: var(--text); background: var(--surface); }
.nav-links a.lang { color: var(--text-3); font-size: 13.5px; letter-spacing: 0.01em; }

/* ---------- buttons ---------- */
.btn {
  display: inline-flex; align-items: center; justify-content: center; gap: 8px;
  font-weight: 600; font-size: 15px; line-height: 1;
  padding: 13px 22px; border-radius: 12px;
  border: 1px solid var(--border); background: var(--surface); color: var(--text);
  transition: background 0.15s ease, border-color 0.15s ease, color 0.15s ease;
}
.btn-primary { background: var(--accent); color: var(--accent-text); border-color: var(--accent); }
.btn-primary:hover { background: var(--accent-hover); color: #fff; border-color: var(--accent-hover); }
.btn-ghost { background: transparent; }
.btn-ghost:hover { border-color: var(--border-strong); color: var(--text); }
.btn-sm { padding: 10px 16px; font-size: 14px; border-radius: 10px; }
.btn-block { width: 100%; }

/* ---------- hero (centered, the gallery carries the visuals) ---------- */
.hero {
  display: grid; grid-template-columns: minmax(0, 1fr);
  place-items: center; padding: 76px 0 26px;
}
@media (max-width: 880px) {
  .hero { padding-top: 40px; }
}
.hero-copy { text-align: center; max-width: 820px; }
.hero-copy .hero-actions, .hero-copy .applinks, .hero-copy .stats { justify-content: center; }
.pill {
  display: inline-flex; align-items: center; gap: 9px;
  font-size: 12.5px; font-weight: 600; letter-spacing: 0.07em; text-transform: uppercase;
  color: var(--accent); background: var(--accent-dim);
  border: 1px solid rgba(34, 197, 94, 0.3); padding: 7px 13px; border-radius: 999px;
}
.pill::before {
  content: ""; width: 7px; height: 7px; flex: none; border-radius: 50%;
  background: var(--accent); box-shadow: 0 0 0 4px rgba(34, 197, 94, 0.18);
}
.hero-title {
  font-size: clamp(2.15rem, 5vw + 0.6rem, 3.55rem);
  line-height: 1.06; letter-spacing: -0.02em; margin: 22px 0 18px;
  font-weight: 700; text-wrap: balance;
}
.hero-lead {
  color: var(--text-2); font-size: clamp(1rem, 1.2vw + 0.4rem, 1.125rem);
  max-width: 56ch; margin: 0 auto 30px; text-wrap: pretty;
}
.hero-actions { display: flex; gap: 12px; flex-wrap: wrap; align-items: center; }

/* app download links under the hero CTAs */
.applinks { display: flex; gap: 12px; flex-wrap: wrap; margin-top: 18px; }
.applink {
  display: inline-flex; align-items: center; gap: 11px;
  padding: 9px 14px; border: 1px solid var(--border); border-radius: 12px;
  background: var(--surface); color: var(--text); text-decoration: none;
  transition: background 0.15s ease, border-color 0.15s ease;
}
.applink svg { width: 22px; height: 22px; flex: none; color: var(--accent); }
.applink span { display: grid; gap: 1px; line-height: 1.15; }
.applink b { font-size: 13.5px; font-weight: 600; }
.applink small { font-size: 11.5px; color: var(--text-3); }
.applink:hover { border-color: var(--border-strong); background: var(--surface-2); }

/* ---------- hero stats (metrics as the single accent) ---------- */
.stats { display: flex; gap: 30px; margin-top: 34px; flex-wrap: wrap; }
.stats .s { display: grid; gap: 3px; padding-inline-end: 30px; border-inline-end: 1px solid var(--border); }
.stats .s:last-child { border-inline-end: 0; }
.stats .s:only-child { border-inline-end: 0; }
.stats b {
  font-size: 1.85rem; line-height: 1; letter-spacing: -0.02em;
  color: var(--accent); font-variant-numeric: tabular-nums;
}
.stats small { color: var(--text-3); font-size: 12.5px; letter-spacing: 0.02em; }
@media (max-width: 720px) {
  .stats { gap: 22px 18px; }
  .stats .s { padding-inline-end: 18px; }
}

/* ---------- screenshot gallery: real app screens in a scroll-snap carousel ---------- */
.gallery-section { padding-top: 4px; }
.gallery { position: relative; }
.gal-track {
  display: grid; grid-auto-flow: column; grid-auto-columns: minmax(240px, 300px);
  gap: 20px; overflow-x: auto; overscroll-behavior-x: contain;
  scroll-snap-type: x mandatory; padding: 8px 2px 20px;
  scrollbar-width: none;
}
.gal-track::-webkit-scrollbar { display: none; }
.gitem {
  scroll-snap-align: center; margin: 0;
  border: 1px solid var(--border); border-radius: 18px; overflow: hidden;
  background: var(--surface-2); box-shadow: var(--shadow);
}
.gitem img {
  display: block; width: 100%; height: auto;
  aspect-ratio: 600 / 1334; object-fit: cover;
}
.gitem figcaption {
  padding: 12px 16px 14px; font-size: 13.5px; font-weight: 600; color: var(--text-2);
  border-top: 1px solid var(--border); text-align: center;
}
.gal-btn {
  position: absolute; z-index: 2; top: 45%; translate: 0 -50%;
  width: 46px; height: 46px; border-radius: 50%;
  appearance: none; cursor: pointer; font-size: 22px; line-height: 1;
  color: var(--text); background: rgba(13, 24, 38, 0.92);
  border: 1px solid var(--border-strong);
  display: grid; place-items: center; padding: 0;
  transition: border-color 0.15s ease, color 0.15s ease;
}
.gal-btn:hover { border-color: var(--accent); color: var(--accent); }
.gal-btn.prev { inset-inline-start: 4px; }
.gal-btn.next { inset-inline-end: 4px; }
.gal-dots { display: flex; gap: 8px; justify-content: center; margin-top: 10px; }
.gal-dots .dot {
  width: 9px; height: 9px; padding: 0; border-radius: 50%;
  appearance: none; cursor: pointer; border: 1px solid transparent;
  background: var(--border-strong); transition: background 0.15s ease, transform 0.15s ease;
}
.gal-dots .dot.is-on { background: var(--accent); transform: scale(1.3); }
@media (max-width: 640px) {
  .gal-track { grid-auto-columns: 70%; gap: 14px; }
  .gal-btn { display: none; }
}

/* ---------- sections ---------- */
.section { padding: 46px 0; }
section[id] { scroll-margin-top: 18px; }
.section-label {
  display: block; font-size: 12.5px; font-weight: 600; letter-spacing: 0.09em;
  text-transform: uppercase; color: var(--accent);
}
.section-title { font-size: clamp(1.6rem, 3vw + 0.4rem, 2.1rem); line-height: 1.15; letter-spacing: -0.01em; margin: 12px 0 8px; text-wrap: balance; }
.section-sub { color: var(--text-2); max-width: 62ch; margin: 0 0 34px; text-wrap: pretty; }
.section-note { color: var(--text-3); font-size: 14px; margin: 26px 0 0; }

/* ---------- features (hairline-divided grid, no cards) ---------- */
.feature-grid {
  display: grid; grid-template-columns: repeat(3, 1fr);
  gap: 36px; border-top: 1px solid var(--border); padding-top: 4px;
}
@media (max-width: 880px) { .feature-grid { grid-template-columns: repeat(2, 1fr); gap: 30px; } }
@media (max-width: 560px) { .feature-grid { grid-template-columns: 1fr; } }
.feature { padding-top: 26px; border-top: 1px solid var(--border); margin-top: -1px; }
.feature svg { width: 22px; height: 22px; color: var(--accent); }
.feature h3 { font-size: 1.05rem; margin: 14px 0 6px; font-weight: 600; letter-spacing: 0.01em; }
.feature p { color: var(--text-2); font-size: 14.5px; margin: 0; line-height: 1.55; text-wrap: pretty; }

/* ---------- how it works (register-day steps) ---------- */
.steps {
  list-style: none; margin: 0; padding: 0;
  display: grid; grid-template-columns: repeat(3, 1fr); gap: 22px;
}
@media (max-width: 880px) { .steps { grid-template-columns: 1fr; max-width: 520px; } }
.steps li {
  display: flex; gap: 16px; align-items: flex-start;
  border: 1px solid var(--border); border-radius: var(--radius);
  background: var(--surface-2); padding: 22px 20px;
}
.step-num {
  display: grid; place-items: center; flex: none;
  width: 38px; height: 38px; border-radius: 12px;
  background: var(--accent-dim); color: var(--accent);
  border: 1px solid rgba(34, 197, 94, 0.3);
  font-weight: 700; font-size: 15px; font-variant-numeric: tabular-nums;
}
.steps h3 { font-size: 1.02rem; margin: 2px 0 6px; font-weight: 600; letter-spacing: 0.01em; }
.steps p { color: var(--text-2); font-size: 14px; margin: 0; line-height: 1.55; text-wrap: pretty; }

/* ---------- FAQ (native details, no JS) ---------- */
.faqs { border-top: 1px solid var(--border); }
.faqs details { border-bottom: 1px solid var(--border); }
.faqs summary {
  cursor: pointer; list-style: none; display: flex; justify-content: space-between;
  align-items: center; gap: 16px; padding: 20px 2px;
  font-weight: 600; font-size: 16px; color: var(--text);
}
.faqs summary::-webkit-details-marker { display: none; }
.faqs summary::after { content: "+"; color: var(--accent); font-size: 21px; line-height: 1; flex: none; transition: transform 0.18s ease; }
.faqs details[open] summary::after { content: "–"; }
.faqs details p {
  margin: 0 0 20px; color: var(--text-2); font-size: 15px; line-height: 1.65;
  max-width: 64ch; text-wrap: pretty;
}

/* ---------- verticals ---------- */
.vgrid {
  display: grid; grid-template-columns: repeat(3, 1fr); gap: 11px 28px;
  border-top: 1px solid var(--border); padding-top: 26px;
}
@media (max-width: 880px) { .vgrid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 560px) { .vgrid { grid-template-columns: 1fr; } }
.v { display: flex; align-items: center; gap: 11px; color: var(--text-2); font-size: 14.5px; }
.v::before {
  content: ""; width: 6px; height: 6px; flex: none; border-radius: 50%;
  background: var(--accent); opacity: 0.85;
}

/* ---------- plans (interactive cards — justified) ---------- */
.plans { display: grid; grid-template-columns: repeat(3, 1fr); gap: 18px; align-items: stretch; }
@media (max-width: 880px) { .plans { grid-template-columns: 1fr; max-width: 520px; margin-inline: auto; } }
.plan {
  position: relative; display: flex; flex-direction: column;
  background: var(--surface-2); border: 1px solid var(--border);
  border-radius: var(--radius); padding: 28px 25px;
}
.plan.featured {
  background: linear-gradient(180deg, rgba(34, 197, 94, 0.1), rgba(34, 197, 94, 0.02) 55%), var(--surface-2);
  border-color: rgba(34, 197, 94, 0.45); box-shadow: 0 20px 44px rgba(0, 0, 0, 0.28);
}
.plan-tag {
  position: absolute; top: -11px; inset-inline-start: 26px;
  background: var(--accent); color: var(--accent-text);
  font-size: 11px; font-weight: 700; letter-spacing: 0.07em; text-transform: uppercase;
  padding: 5px 11px; border-radius: 999px;
}
.plan h3 { font-size: 1.05rem; margin: 6px 0 4px; font-weight: 600; }
.plan .desc { color: var(--text-3); font-size: 13.5px; margin: 0 0 18px; line-height: 1.5; }
.plan .price { display: flex; align-items: baseline; gap: 8px; margin-bottom: 20px; }
.plan .price .amount { font-size: 2.05rem; font-weight: 700; letter-spacing: -0.02em; font-variant-numeric: tabular-nums; }
.plan .price .per { color: var(--text-3); font-size: 13px; }
.plan ul { list-style: none; margin: 0 0 24px; padding: 0; display: grid; gap: 9px; font-size: 13.5px; color: var(--text-2); }
.plan ul li { display: flex; gap: 10px; align-items: flex-start; }
.plan ul li::before {
  content: ""; margin-top: 8px; width: 6px; height: 6px; flex: none;
  border-radius: 50%; background: var(--accent); opacity: 0.9;
}
.plan .fill { margin-top: auto; }

/* ---------- feature comparison table (pricing) ---------- */
.compare-scroll { overflow-x: auto; border-top: 1px solid var(--border); }
table.compare { width: 100%; border-collapse: collapse; font-size: 14px; min-width: 620px; }
.compare th, .compare td { padding: 13px 18px; border-bottom: 1px solid var(--border); }
.compare thead th {
  color: var(--text-3); font-size: 12.5px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.06em; text-align: center;
}
.compare thead th:first-child, .compare tbody th { text-align: start; font-weight: 500; color: var(--text-2); }
.compare tbody th { white-space: nowrap; }
.compare td { text-align: center; }
.compare tr:last-child th, .compare tr:last-child td { border-bottom: 0; }
.compare .match { width: 16px; height: 16px; color: var(--accent); }
.compare .dash { color: var(--text-3); }

/* ---------- page heads + privacy prose ---------- */
.page-head { padding: 60px 0 12px; max-width: 720px; }
.page-title { font-size: clamp(1.9rem, 3.5vw + 0.6rem, 2.7rem); line-height: 1.1; letter-spacing: -0.02em; margin: 16px 0 12px; text-wrap: balance; }
.prose { max-width: 70ch; color: var(--text-2); font-size: 15.5px; line-height: 1.7; text-wrap: pretty; }
.prose h2 { font-size: 1.25rem; color: var(--text); margin: 36px 0 10px; font-weight: 600; letter-spacing: 0.01em; }
.prose p { margin: 0 0 14px; }
.prose ul { margin: 0 0 16px; padding-inline-start: 22px; }
.prose li { margin: 0 0 8px; }
.prose strong { color: var(--text); }
.prose a { text-decoration: underline; text-underline-offset: 3px; }

/* ---------- footer (structured, enterprise-style) ---------- */
.site-footer { border-top: 1px solid var(--border); margin-top: 44px; }
.footer {
  display: grid; grid-template-columns: minmax(0, 1.4fr) auto auto minmax(0, 1fr);
  gap: 28px 44px; align-items: start; padding: 28px 24px 44px;
  color: var(--text-3); font-size: 13.5px;
}
.footer-brand b { color: var(--text); font-size: 15px; }
.footer-brand b small { color: var(--text-3); font-weight: 500; font-size: 12px; }
.footer-brand p { margin: 6px 0 0; color: var(--text-3); font-size: 13px; max-width: 34ch; text-wrap: pretty; }
.footer-links { display: grid; gap: 9px; font-size: 13.5px; }
.footer-links a { color: var(--text-2); width: fit-content; }
.footer-links a:hover { color: var(--text); }
.footer-text { display: grid; justify-items: end; gap: 6px; font-size: 13px; }
@media (max-width: 900px) {
  .footer { grid-template-columns: 1fr; gap: 22px; }
  .footer-text { justify-items: start; }
}

@media (max-width: 560px) {
  .hero-title { letter-spacing: -0.01em; }
  .nav-links { margin-inline-start: 0; width: 100%; }
}
`
