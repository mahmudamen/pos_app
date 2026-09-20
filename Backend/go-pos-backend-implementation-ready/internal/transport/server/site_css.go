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

/* ---------- hero ---------- */
.hero {
  display: grid; grid-template-columns: minmax(0, 1.05fr) minmax(0, 0.95fr);
  gap: 56px; align-items: center; padding: 64px 0 36px;
}
@media (max-width: 880px) {
  .hero { grid-template-columns: 1fr; gap: 44px; padding-top: 34px; }
}
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
  max-width: 56ch; margin: 0 0 30px; text-wrap: pretty;
}
.hero-actions { display: flex; gap: 12px; flex-wrap: wrap; align-items: center; }

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

/* ---------- ESC/POS style receipt (hero artifact) ---------- */
.receipt {
  position: relative; max-width: 360px; justify-self: end; width: 100%;
  background: linear-gradient(180deg, #0f1d2c, #0c1723);
  border: 1px solid var(--border); border-radius: var(--radius);
  box-shadow: var(--shadow); padding: 26px 24px 20px;
  font-family: var(--mono); font-size: 12.5px; color: var(--text-2);
  animation: rpaper 0.5s ease both;
}
@keyframes rpaper { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: none; } }
@media (max-width: 880px) { .receipt { justify-self: center; } }
.receipt-head { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; color: var(--text); font-weight: 600; letter-spacing: 0.01em; }
.receipt-date { margin: 3px 0 14px; color: var(--text-3); font-size: 11.5px; }
.receipt .stamp {
  position: absolute; inset-inline-end: 22px; top: 66px; rotate: -14deg;
  color: var(--accent); border: 2px solid var(--accent); border-radius: 6px;
  padding: 3px 10px; font-weight: 700; letter-spacing: 0.12em; font-size: 11px; opacity: 0.92;
}
.receipt-row { display: flex; justify-content: space-between; gap: 12px; align-items: baseline; }
.receipt-row + .receipt-row { margin-top: 7px; }
.receipt-item-name { overflow-wrap: anywhere; }
.receipt .muted { color: var(--text-3); }
.receipt-rule { border: 0; border-top: 1px dashed rgba(148, 187, 214, 0.35); margin: 12px 0; }
.receipt-total { display: flex; justify-content: space-between; align-items: baseline; font-size: 16px; color: var(--text); font-weight: 700; }
.receipt-total .money { color: var(--accent); font-variant-numeric: tabular-nums; }
.receipt .ok-line { display: flex; align-items: center; gap: 7px; }
.receipt .ok-line svg { width: 14px; height: 14px; color: var(--accent); flex: none; }
.receipt-thanks { margin: 14px 0 0; text-align: center; color: var(--text-3); font-size: 11.5px; }
.barcode { display: block; width: 100%; height: 26px; margin-top: 13px; opacity: 0.85; }
.receipt-total { animation: rtotal 1s ease 0.85s both; }
@keyframes rtotal {
  0% { transform: scale(1); }
  40% { transform: scale(1.03); color: var(--accent-hi); }
  100% { transform: scale(1); }
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
