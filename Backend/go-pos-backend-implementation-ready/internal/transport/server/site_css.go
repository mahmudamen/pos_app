package server

// siteCSS is the shared design system for every public POS.Go page. Direction-
// aware (logical properties, so the same file drives LTR English and RTL Arabic),
// light clean-SaaS canvas with the POS.Go emerald reserved for the single
// interactive accent: CTAs, links, focus, active states and soft tints.
// No external fonts or network assets — the stack renders Arabic natively.
const siteCSS = `
:root {
  color-scheme: light;
  --bg: #f7f9fb;
  --bg-wash: radial-gradient(960px 520px at 85% -10%, rgba(34, 197, 94, 0.08), transparent 60%);
  --surface: #ffffff;
  --surface-2: #fbfdfc;
  --border: #e5eaf0;
  --border-strong: #d3dce6;
  --text: #0b1b2a;
  --text-2: #46586b;
  --text-3: #8aa0b4;
  --accent: #16a34a;
  --accent-hi: #22c55e;
  --accent-hover: #15803d;
  --accent-dim: #f0fdf4;
  --accent-text: #062212;
  --radius: 16px;
  --shadow-sm: 0 1px 2px rgba(15, 23, 42, 0.05);
  --shadow: 0 1px 2px rgba(15, 23, 42, 0.05), 0 18px 44px -20px rgba(15, 23, 42, 0.16);
  --shadow-lift: 0 2px 6px rgba(15, 23, 42, 0.06), 0 26px 60px -24px rgba(15, 23, 42, 0.22);
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
    var(--bg-wash),
    linear-gradient(180deg, #fbfdfc 0%, var(--bg) 30%, var(--bg) 100%);
  background-attachment: fixed;
  min-height: 100vh;
}
a { color: var(--accent); text-decoration: none; }
a:hover { color: var(--accent-hover); }
:focus-visible { outline: 2px solid var(--accent); outline-offset: 3px; }
::selection { background: var(--accent); color: #fff; }
.container { max-width: 1140px; margin-inline: auto; padding-inline: 24px; }

/* ---------- header / nav ---------- */
.site-header {
  position: sticky; top: 0; z-index: 30;
  background: rgba(247, 249, 251, 0.82);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--border);
  padding: 14px 0;
}
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
.nav-links a:not(.btn):hover { color: var(--text); background: #eef3f7; }
.nav-links a.lang { color: var(--text-3); font-size: 13.5px; letter-spacing: 0.01em; }

/* ---------- buttons ---------- */
.btn {
  display: inline-flex; align-items: center; justify-content: center; gap: 8px;
  font-weight: 600; font-size: 15px; line-height: 1;
  padding: 13px 22px; border-radius: 999px;
  border: 1px solid var(--border); background: var(--surface); color: var(--text);
  box-shadow: var(--shadow-sm);
  transition: background 0.15s ease, border-color 0.15s ease, color 0.15s ease, box-shadow 0.15s ease;
}
.btn-primary { background: var(--accent); color: #fff; border-color: var(--accent); box-shadow: 0 8px 20px -10px rgba(22, 163, 74, 0.55); }
.btn-primary:hover { background: var(--accent-hover); color: #fff; border-color: var(--accent-hover); box-shadow: 0 10px 24px -10px rgba(22, 163, 74, 0.6); }
.btn-ghost { background: transparent; box-shadow: none; }
.btn-ghost:hover { border-color: var(--border-strong); color: var(--text); }
.btn-sm { padding: 10px 16px; font-size: 14px; border-radius: 999px; }
.btn-block { width: 100%; }

/* ---------- hero ---------- */
.hero {
  display: grid; grid-template-columns: minmax(0, 1fr);
  place-items: center; padding: 84px 0 34px;
}
@media (max-width: 880px) {
  .hero { padding-top: 44px; }
}
.hero-copy { text-align: center; max-width: 860px; }
.hero-copy .hero-actions, .hero-copy .applinks, .hero-copy .stats { justify-content: center; }
.pill {
  display: inline-flex; align-items: center; gap: 9px;
  font-size: 12.5px; font-weight: 600; letter-spacing: 0.05em;
  color: var(--accent); background: var(--accent-dim);
  border: 1px solid rgba(22, 163, 74, 0.22); padding: 7px 14px; border-radius: 999px;
}
.pill::before {
  content: ""; width: 7px; height: 7px; flex: none; border-radius: 50%;
  background: var(--accent); box-shadow: 0 0 0 4px rgba(22, 163, 74, 0.14);
}
.hero-title {
  font-size: clamp(2.3rem, 5vw + 0.6rem, 3.8rem);
  line-height: 1.05; letter-spacing: -0.025em; margin: 26px 0 18px;
  font-weight: 800; text-wrap: balance;
}
.hero-lead {
  color: var(--text-2); font-size: clamp(1.05rem, 0.9vw + 0.6rem, 1.2rem);
  max-width: 62ch; margin: 0 auto 32px; text-wrap: pretty;
}
.hero-actions { display: flex; gap: 12px; flex-wrap: wrap; align-items: center; }

/* app download links under the hero CTAs */
.applinks { display: flex; gap: 12px; flex-wrap: wrap; margin-top: 20px; }
.applink {
  display: inline-flex; align-items: center; gap: 11px;
  padding: 9px 16px; border: 1px solid var(--border); border-radius: 12px;
  background: var(--surface); color: var(--text); text-decoration: none;
  box-shadow: var(--shadow-sm);
  transition: background 0.15s ease, border-color 0.15s ease;
}
.applink svg { width: 22px; height: 22px; flex: none; color: var(--accent); }
.applink span { display: grid; gap: 1px; line-height: 1.15; }
.applink b { font-size: 13.5px; font-weight: 600; }
.applink small { font-size: 11.5px; color: var(--text-3); }
.applink:hover { border-color: var(--border-strong); background: var(--surface-2); }

/* ---------- hero stats ---------- */
.stats { display: flex; gap: 32px; margin-top: 40px; flex-wrap: wrap; }
.stats .s { display: grid; gap: 3px; padding-inline-end: 32px; border-inline-end: 1px solid var(--border); }
.stats .s:last-child { border-inline-end: 0; }
.stats .s:only-child { border-inline-end: 0; }
.stats b {
  font-size: 1.9rem; line-height: 1; letter-spacing: -0.02em;
  color: var(--accent); font-variant-numeric: tabular-nums;
}
.stats small { color: var(--text-3); font-size: 12.5px; letter-spacing: 0.02em; }
@media (max-width: 720px) {
  .stats { gap: 22px 18px; }
  .stats .s { padding-inline-end: 18px; }
}

/* ---------- screenshot gallery: real app screens, framed in phone mockups ---------- */
.gallery-section { padding-top: 10px; }
.gallery { position: relative; }
.gal-track {
  display: grid; grid-auto-flow: column; grid-auto-columns: 310px;
  gap: 28px; overflow-x: auto; overscroll-behavior-x: contain;
  scroll-snap-type: x mandatory; padding: 18px 2px 28px;
  scrollbar-width: none;
}
.gal-track::-webkit-scrollbar { display: none; }
.gitem { scroll-snap-align: center; margin: 0; }
.phone {
  position: relative; width: 100%;
  padding: 11px;
  background: linear-gradient(180deg, #101c2b, #0d1622);
  border-radius: 38px;
  box-shadow: 0 30px 60px -24px rgba(15, 23, 42, 0.5), inset 0 0 0 1px #243447;
}
.phone::before {
  content: ""; position: absolute; top: 18px; left: 50%; translate: -50% 0;
  width: 96px; height: 24px; border-radius: 999px;
  background: #0d1622; box-shadow: inset 0 0 0 1px #243447; z-index: 2;
}
.phone-screen {
  border-radius: 28px; overflow: hidden; background: #fff;
  aspect-ratio: 600 / 1334;
}
.phone-screen img {
  display: block; width: 100%; height: 100%; object-fit: cover;
}
.gitem figcaption {
  margin-top: 14px; text-align: center;
  font-size: 13px; font-weight: 600; color: var(--text-2);
}
.gal-btn {
  position: absolute; z-index: 2; top: 44%; translate: 0 -50%;
  width: 46px; height: 46px; border-radius: 50%;
  appearance: none; cursor: pointer; font-size: 22px; line-height: 1;
  color: var(--text); background: var(--surface);
  border: 1px solid var(--border-strong);
  box-shadow: var(--shadow-sm);
  display: grid; place-items: center; padding: 0;
  transition: border-color 0.15s ease, color 0.15s ease;
}
.gal-btn:hover { border-color: var(--accent); color: var(--accent); }
.gal-btn.prev { inset-inline-start: 4px; }
.gal-btn.next { inset-inline-end: 4px; }
.gal-dots { display: flex; gap: 8px; justify-content: center; margin-top: 6px; }
.gal-dots .dot {
  width: 9px; height: 9px; padding: 0; border-radius: 50%;
  appearance: none; cursor: pointer; border: 1px solid transparent;
  background: #cfd9e3; transition: background 0.15s ease, transform 0.15s ease;
}
.gal-dots .dot.is-on { background: var(--accent); transform: scale(1.3); }
@media (max-width: 640px) {
  .gal-track { grid-auto-columns: 72%; gap: 20px; }
  .gal-btn { display: none; }
}

/* ---------- sections ---------- */
.section { padding: 54px 0; }
section[id] { scroll-margin-top: 66px; }
.section-label {
  display: block; font-size: 12.5px; font-weight: 700; letter-spacing: 0.09em;
  text-transform: uppercase; color: var(--accent);
}
.section-title { font-size: clamp(1.7rem, 3vw + 0.4rem, 2.3rem); line-height: 1.12; letter-spacing: -0.02em; margin: 12px 0 10px; text-wrap: balance; }
.section-sub { color: var(--text-2); max-width: 64ch; margin: 0 0 36px; text-wrap: pretty; }
.section-note { color: var(--text-3); font-size: 14px; margin: 26px 0 0; }

/* ---------- features (hairline-divided grid, no cards) ---------- */
.feature-grid {
  display: grid; grid-template-columns: repeat(3, 1fr);
  gap: 38px; border-top: 1px solid var(--border); padding-top: 4px;
}
@media (max-width: 880px) { .feature-grid { grid-template-columns: repeat(2, 1fr); gap: 30px; } }
@media (max-width: 560px) { .feature-grid { grid-template-columns: 1fr; } }
.feature { padding-top: 26px; border-top: 1px solid var(--border); margin-top: -1px; }
.feature svg { width: 22px; height: 22px; color: var(--accent); }
.feature h3 { font-size: 1.05rem; margin: 14px 0 6px; font-weight: 700; letter-spacing: 0.01em; }
.feature p { color: var(--text-2); font-size: 14.5px; margin: 0; line-height: 1.58; text-wrap: pretty; }

/* ---------- all-features modules (bento cards) ---------- */
.all-grid {
  display: grid; grid-template-columns: repeat(3, 1fr);
  gap: 16px; align-items: stretch;
}
@media (max-width: 940px) { .all-grid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 600px) { .all-grid { grid-template-columns: 1fr; } }
.all-card {
  display: flex; flex-direction: column;
  background: var(--surface); border: 1px solid var(--border);
  border-radius: 18px; padding: 24px 22px;
  box-shadow: var(--shadow-sm);
  transition: box-shadow 0.2s ease, border-color 0.2s ease, translate 0.2s ease;
}
.all-card:hover { box-shadow: var(--shadow); border-color: var(--border-strong); translate: 0 -2px; }
.all-head { display: flex; align-items: center; gap: 14px; }
.all-ico {
  display: grid; place-items: center; flex: none;
  width: 44px; height: 44px; border-radius: 12px;
  background: var(--accent-dim); color: var(--accent);
  border: 1px solid rgba(22, 163, 74, 0.18);
}
.all-ico svg { width: 22px; height: 22px; }
.all-card h3 { font-size: 1.08rem; font-weight: 700; letter-spacing: -0.01em; }
.all-card .lead { color: var(--text-2); font-size: 14.5px; margin: 12px 0 0; line-height: 1.58; text-wrap: pretty; }
.all-card ul { list-style: none; margin: 16px 0 0; padding: 0; display: grid; gap: 9px; }
.all-card ul li {
  display: flex; gap: 10px; align-items: flex-start;
  font-size: 13.5px; color: var(--text-2); line-height: 1.5;
}
.all-card ul li::before {
  content: ""; flex: none; margin-top: 7px;
  width: 14px; height: 14px; border-radius: 50%;
  background: var(--accent-dim) url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 14 14' fill='none' stroke='%2316a34a' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M3 7.5l2.6 2.6L11 4.5'/%3E%3C/svg%3E") center / 11px no-repeat;
}

/* ---------- Copilot / AI support ---------- */
.copilot-sec { position: relative; display: grid; gap: 16px; }
.copilot-sec::before {
  content: ""; position: absolute; inset: -2px 0; pointer-events: none;
  background:
    radial-gradient(620px 340px at 6% 0%, rgba(0, 194, 255, 0.1), transparent 70%),
    radial-gradient(520px 320px at 98% 100%, rgba(0, 102, 255, 0.1), transparent 70%);
}
.copt-card {
  position: relative; background: var(--surface);
  border: 1px solid var(--border); border-radius: 20px;
  padding: 40px 38px; overflow: hidden;
  box-shadow: var(--shadow-sm);
}
.copt-live {
  display: inline-flex; align-items: center; gap: 9px;
  font-size: 12px; font-weight: 600; color: #15803d;
  background: var(--accent-dim); border: 1px solid rgba(22, 163, 74, 0.24);
  padding: 7px 15px; border-radius: 999px; letter-spacing: 0.02em;
}
.copt-live i { width: 6px; height: 6px; border-radius: 50%; background: #16a34a; animation: copt-pulse 1.8s ease-in-out infinite; }
@keyframes copt-pulse { 0%, to { opacity: 1; } 50% { opacity: 0.35; } }
.copt-title {
  font-size: clamp(1.7rem, 3vw + 0.4rem, 2.3rem); font-weight: 800;
  line-height: 1.12; letter-spacing: -0.02em; margin: 20px 0 14px; text-wrap: balance;
  color: var(--text);
}
.co-grad {
  background: linear-gradient(135deg, #00a3ef, #2563eb);
  -webkit-background-clip: text; background-clip: text;
  -webkit-text-fill-color: transparent; color: transparent;
}
.copt-lead { color: var(--text-2); max-width: 62ch; margin: 0 0 26px; font-size: 16px; line-height: 1.7; text-wrap: pretty; }
.copt-stats {
  display: grid; grid-template-columns: repeat(3, 1fr);
  border: 1px solid var(--border); border-radius: 14px; overflow: hidden;
  background: var(--surface-2);
}
.copt-stat { display: grid; gap: 7px; padding: 16px 18px; border-inline-end: 1px solid var(--border); }
.copt-stat:last-child { border-inline-end: 0; }
.copt-stat small { font-size: 9px; letter-spacing: 0.6px; text-transform: uppercase; color: var(--text-3); }
.copt-stat b { font-family: var(--mono); font-size: 22px; line-height: 1; font-weight: 700; font-variant-numeric: tabular-nums; }
.copt-stat em { font-style: normal; font-size: 10px; color: var(--text-3); }
.c-s1 { color: #16a34a; } .c-s2 { color: #0284c7; } .c-s3 { color: #d97706; }
.copt-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; }
.copt-feat {
  background: var(--surface); border: 1px solid var(--border);
  border-radius: 14px; padding: 22px;
  box-shadow: var(--shadow-sm);
}
.copt-ico {
  display: grid; place-items: center; width: 40px; height: 40px;
  border-radius: 10px; background: #e6f4fe; margin-bottom: 14px;
}
.copt-ico svg { width: 20px; height: 20px; color: #0284c7; }
.copt-feat h3 { font-size: 14.5px; font-weight: 700; margin: 0 0 6px; }
.copt-feat p { margin: 0; font-size: 13.5px; line-height: 1.6; color: var(--text-2); text-wrap: pretty; }
.copt-cta-row { display: flex; justify-content: center; }
.copt-cta-wrap { position: relative; width: fit-content; }
.copt-halo {
  position: absolute; inset: -6px; border-radius: 18px;
  background: linear-gradient(135deg, #00a3ef, #2563eb, #00a3ef);
  filter: blur(12px); opacity: 0.28; animation: copt-glow 2.5s ease-in-out infinite;
}
@keyframes copt-glow { 0%, to { opacity: 0.2; transform: scale(1); } 50% { opacity: 0.42; transform: scale(1.04); } }
.copt-cta {
  position: relative; display: inline-flex; align-items: center; gap: 9px;
  min-height: 52px; padding: 0 26px; border-radius: 14px; overflow: hidden;
  background: linear-gradient(135deg, #00a3ef, #2563eb); color: #fff;
  font-weight: 700; font-size: 15.5px; text-decoration: none;
  transition: transform 0.15s ease, opacity 0.15s ease;
}
.copt-cta:hover { color: #fff; transform: scale(1.02); opacity: 0.94; }
.copt-label, .copt-arrow { position: relative; }
[dir="rtl"] .copt-arrow { transform: scaleX(-1); }
.copt-shimmer {
  position: absolute; inset: 0; pointer-events: none;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.22), transparent);
  animation: copt-shimmer 3s ease-in-out infinite;
}
@keyframes copt-shimmer { 0% { transform: translateX(-100%); } to { transform: translateX(100%); } }
[dir="rtl"] .copt-shimmer { animation-direction: reverse; }
@media (max-width: 880px) {
  .copt-grid { grid-template-columns: 1fr; }
  .copt-stats { grid-template-columns: 1fr; }
  .copt-stat { border-inline-end: 0; border-bottom: 1px solid var(--border); }
  .copt-stat:last-child { border-bottom: 0; }
  .copt-card { padding: 30px 22px; }
}

/* ---------- how it works ---------- */
.steps {
  list-style: none; margin: 0; padding: 0;
  display: grid; grid-template-columns: repeat(3, 1fr); gap: 18px;
}
@media (max-width: 880px) { .steps { grid-template-columns: 1fr; max-width: 520px; } }
.steps li {
  display: flex; gap: 16px; align-items: flex-start;
  border: 1px solid var(--border); border-radius: var(--radius);
  background: var(--surface); padding: 22px 20px; box-shadow: var(--shadow-sm);
}
.step-num {
  display: grid; place-items: center; flex: none;
  width: 38px; height: 38px; border-radius: 12px;
  background: var(--accent-dim); color: var(--accent);
  border: 1px solid rgba(22, 163, 74, 0.22);
  font-weight: 800; font-size: 15px; font-variant-numeric: tabular-nums;
}
.steps h3 { font-size: 1.02rem; margin: 2px 0 6px; font-weight: 700; letter-spacing: 0.01em; }
.steps p { color: var(--text-2); font-size: 14px; margin: 0; line-height: 1.58; text-wrap: pretty; }

/* ---------- FAQ ---------- */
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
  display: grid; grid-template-columns: repeat(3, 1fr); gap: 13px 28px;
  border-top: 1px solid var(--border); padding-top: 28px;
}
@media (max-width: 880px) { .vgrid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 560px) { .vgrid { grid-template-columns: 1fr; } }
.v {
  display: flex; align-items: center; gap: 11px; color: var(--text-2); font-size: 14.5px;
  background: var(--surface); border: 1px solid var(--border); border-radius: 12px;
  padding: 11px 14px; box-shadow: var(--shadow-sm);
}
.v::before {
  content: ""; width: 7px; height: 7px; flex: none; border-radius: 50%;
  background: var(--accent); opacity: 0.85;
}

/* ---------- plans ---------- */
.plans { display: grid; grid-template-columns: repeat(3, 1fr); gap: 18px; align-items: stretch; }
@media (max-width: 880px) { .plans { grid-template-columns: 1fr; max-width: 520px; margin-inline: auto; } }
.plan {
  position: relative; display: flex; flex-direction: column;
  background: var(--surface); border: 1px solid var(--border);
  border-radius: var(--radius); padding: 28px 25px; box-shadow: var(--shadow-sm);
}
.plan.featured {
  background: linear-gradient(180deg, rgba(22, 163, 74, 0.06), rgba(22, 163, 74, 0.01) 55%), var(--surface);
  border-color: rgba(22, 163, 74, 0.4); box-shadow: 0 22px 48px -22px rgba(22, 163, 74, 0.32);
}
.plan-tag {
  position: absolute; top: -11px; inset-inline-start: 26px;
  background: var(--accent); color: #fff;
  font-size: 11px; font-weight: 700; letter-spacing: 0.07em; text-transform: uppercase;
  padding: 5px 12px; border-radius: 999px;
}
.plan h3 { font-size: 1.05rem; margin: 6px 0 4px; font-weight: 700; }
.plan .desc { color: var(--text-3); font-size: 13.5px; margin: 0 0 18px; line-height: 1.5; }
.plan .price { display: flex; align-items: baseline; gap: 8px; margin-bottom: 20px; }
.plan .price .amount { font-size: 2.05rem; font-weight: 800; letter-spacing: -0.02em; font-variant-numeric: tabular-nums; }
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
.page-head { padding: 66px 0 12px; max-width: 720px; }
.page-title { font-size: clamp(1.9rem, 3.5vw + 0.6rem, 2.7rem); line-height: 1.1; letter-spacing: -0.02em; margin: 16px 0 12px; text-wrap: balance; }
.prose { max-width: 70ch; color: var(--text-2); font-size: 15.5px; line-height: 1.7; text-wrap: pretty; }
.prose h2 { font-size: 1.25rem; color: var(--text); margin: 36px 0 10px; font-weight: 700; letter-spacing: 0.01em; }
.prose p { margin: 0 0 14px; }
.prose ul { margin: 0 0 16px; padding-inline-start: 22px; }
.prose li { margin: 0 0 8px; }
.prose strong { color: var(--text); }
.prose a { text-decoration: underline; text-underline-offset: 3px; }

/* ---------- footer ---------- */
.site-footer { border-top: 1px solid var(--border); margin-top: 52px; background: #fbfdfc; }
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
.footer-tiny { color: var(--text-3); font-size: 12px; }
.footer-tiny:hover { color: var(--text-2); }
@media (max-width: 900px) {
  .footer { grid-template-columns: 1fr; gap: 22px; }
  .footer-text { justify-items: start; }
}

@media (max-width: 560px) {
  .hero-title { letter-spacing: -0.01em; }
}
`
