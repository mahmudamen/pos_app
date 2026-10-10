package mainsite

// siteCSS holds the full stylesheet for the company website. Like the POS.Go
// landing it is served inline with no external requests, and it uses logical
// properties so the same sheet drives both LTR (English) and RTL (Arabic).

const siteCSS = `:root{
  --ink:#0d1630;
  --ink-soft:#33415c;
  --muted:#5b6472;
  --bg:#f6f8fb;
  --surface:#ffffff;
  --panel:#f0f4f8;
  --border:#e3e9f1;
  --border-strong:#c8d3df;
  --teal:#14a6a6;
  --teal-strong:#0f8f95;
  --teal-soft:#e3f8f7;
  --indigo:#4451c7;
  --indigo-soft:#eceefe;
  --wa:#25d366;
  --danger:#d6455d;
  --warn:#c98214;
  --ok:#239b6b;
  --radius:14px;
  --radius-sm:10px;
  --shadow-sm:0 1px 2px rgba(13,22,48,.05),0 2px 8px rgba(13,22,48,.04);
  --shadow-md:0 6px 24px rgba(13,22,48,.08);
  --shadow-lg:0 18px 48px rgba(13,22,48,.14);
  --font:ui-sans-serif,system-ui,"Segoe UI",Tahoma,Arial,"Noto Sans Arabic",sans-serif;
  --mono:ui-monospace,SFMono-Regular,Menlo,Consolas,"Liberation Mono",monospace;
}
*{box-sizing:border-box}
html{scroll-behavior:smooth}
body{margin:0;font-family:var(--font);color:var(--ink);background:var(--bg);
  line-height:1.65;font-size:16px;-webkit-font-smoothing:antialiased;text-rendering:optimizeLegibility}
html[dir=rtl] body{line-height:1.75}
img{max-width:100%}
a{color:var(--teal-strong);text-decoration:none}
a:hover{color:var(--teal)}
::selection{background:var(--teal-soft);color:var(--ink)}
.container{max-width:1140px;margin-inline:auto;padding-inline:24px}
.section{padding-block:64px;scroll-margin-top:84px}
.section.alt{background:var(--surface)}
.section.panel{background:var(--panel)}
.gutter{border-block:1px solid var(--border);background:var(--surface)}

/* ---------- header ---------- */
.site-header{position:sticky;top:0;z-index:50;background:rgba(255,255,255,.86);
  backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);
  border-bottom:1px solid var(--border)}
.nav{display:flex;align-items:center;gap:20px;height:68px;position:relative}
.brand{display:flex;align-items:center;gap:10px;font-weight:800;color:var(--ink);font-size:17px;letter-spacing:.2px}
.brand-mark{width:34px;height:34px;border-radius:10px;display:grid;place-items:center;
  background:linear-gradient(135deg,var(--teal),var(--indigo));color:#fff;font-weight:900;font-size:16px}
.brand-sub{display:block;font-size:10px;font-weight:600;color:var(--muted);letter-spacing:.6px;text-transform:uppercase}
.nav-links{display:flex;align-items:center;gap:2px;margin-inline-start:auto}
.nav-links a{padding:9px 12px;border-radius:9px;color:var(--ink-soft);font-weight:600;font-size:14.5px}
.nav-links a:hover{background:var(--panel);color:var(--ink)}
.nav-links a.active{color:var(--teal-strong);background:var(--teal-soft)}
.nav-cta{margin-inline-start:6px}
.dropdown{position:relative}
.dropdown>button{font:inherit;font-weight:600;font-size:14.5px;color:var(--ink-soft);
  background:none;border:0;padding:9px 12px;border-radius:9px;cursor:pointer;display:flex;align-items:center;gap:5px}
.dropdown>button:hover{background:var(--panel);color:var(--ink)}
.dropdown>button::after{content:"";border:4px solid transparent;border-top-color:currentColor;
  border-bottom:0;transform:translateY(2px)}
.dropdown>button[aria-expanded=true]::after{border-top:0;border-bottom:4px solid currentColor;transform:none}
.dropdown-menu{position:absolute;top:calc(100% + 8px);inset-inline-start:0;min-width:230px;
  background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);
  box-shadow:var(--shadow-md);padding:8px;display:none}
.dropdown:hover .dropdown-menu,.dropdown:focus-within .dropdown-menu{display:block}
.dropdown-menu a{display:block;padding:9px 12px;border-radius:9px;color:var(--ink-soft);font-weight:600;font-size:14px}
.dropdown-menu a:hover{background:var(--panel);color:var(--teal-strong)}
html[dir=rtl] .dropdown-menu{inset-inline-end:0;inset-inline-start:auto}
.nav-burger{display:none;margin-inline-start:auto;background:none;border:1px solid var(--border);
  border-radius:9px;padding:8px 10px;cursor:pointer;line-height:1}
.nav-burger span{display:block;width:20px;height:2px;background:var(--ink);border-radius:2px}
.nav-burger span+span{margin-top:4px}
#nav-toggle{display:none}
@media (max-width:940px){
  .nav-links{display:none;position:absolute;top:100%;inset-inline:0;background:var(--surface);
    border-bottom:1px solid var(--border);box-shadow:var(--shadow-md);padding:12px 16px;flex-direction:column;align-items:stretch}
  #nav-toggle:checked ~ .nav-links{display:flex}
  .nav-burger{display:block}
  .dropdown-menu{position:static;box-shadow:none;border:0;padding:0 0 0 12px}
  .nav-cta{display:none}
}

/* ---------- buttons ---------- */
.btn{display:inline-flex;align-items:center;justify-content:center;gap:8px;font-weight:700;
  font-size:15px;padding:12px 22px;border-radius:11px;border:1px solid transparent;cursor:pointer;
  transition:transform .12s ease,box-shadow .12s ease,background .12s ease;white-space:nowrap}
.btn:active{transform:translateY(1px)}
.btn-primary{background:linear-gradient(135deg,var(--teal),var(--teal-strong));color:#fff;box-shadow:0 8px 20px rgba(20,166,166,.28)}
.btn-primary:hover{box-shadow:0 10px 26px rgba(20,166,166,.38);color:#fff}
.btn-ghost{background:var(--surface);color:var(--ink);border-color:var(--border-strong)}
.btn-ghost:hover{border-color:var(--teal);color:var(--teal-strong)}
.btn-wa{background:var(--wa);color:#fff;box-shadow:0 8px 20px rgba(37,211,102,.28)}
.btn-wa:hover{box-shadow:0 10px 26px rgba(37,211,102,.4);color:#fff}

/* ---------- hero ---------- */
.hero{padding-block:84px 72px;background:
  radial-gradient(900px 420px at 85% -10%,rgba(20,166,166,.14),transparent 60%),
  radial-gradient(760px 380px at 0% 110%,rgba(68,81,199,.12),transparent 60%),
  var(--surface);border-bottom:1px solid var(--border)}
html[dir=rtl] .hero{background:
  radial-gradient(900px 420px at 15% -10%,rgba(20,166,166,.14),transparent 60%),
  radial-gradient(760px 380px at 100% 110%,rgba(68,81,199,.12),transparent 60%),
  var(--surface)}
.hero .container{max-width:920px;text-align:center}
.eyebrow{display:inline-flex;align-items:center;gap:8px;font-size:13px;font-weight:700;
  color:var(--teal-strong);background:var(--teal-soft);border:1px solid rgba(20,166,166,.25);
  padding:6px 14px;border-radius:999px;margin-bottom:20px;letter-spacing:.3px}
h1{font-size:clamp(30px,5.4vw,52px);line-height:1.12;margin:0 0 18px;letter-spacing:-.5px;font-weight:800}
.lead{font-size:clamp(16px,2vw,19px);color:var(--muted);max-width:680px;margin-inline:auto;margin-bottom:30px}
.hero-ctas{display:flex;gap:12px;justify-content:center;flex-wrap:wrap;margin-bottom:34px}
.hero-tiles{display:flex;flex-wrap:wrap;gap:10px;justify-content:center}
.hero-tiles span{display:inline-flex;align-items:center;gap:7px;font-size:13.5px;font-weight:600;
  color:var(--ink-soft);background:var(--panel);border:1px solid var(--border);
  padding:8px 14px;border-radius:999px}
.hero-note{margin-top:22px;font-size:13.5px;color:var(--muted)}
.hero-note a{font-weight:700}
@media (max-width:640px){.hero{padding-block:56px 44px}}

/* ---------- section header ---------- */
.sec-label{font-size:12.5px;font-weight:800;letter-spacing:1.4px;text-transform:uppercase;color:var(--teal-strong);margin-bottom:10px;display:block}
.sec-head{margin-bottom:40px;max-width:720px}
.center .sec-head{margin-inline:auto;text-align:center}
h2{font-size:clamp(24px,3.6vw,34px);line-height:1.2;margin:0 0 10px;letter-spacing:-.3px;font-weight:800}
.sec-head p{color:var(--muted);font-size:16.5px;margin:0}
.sec-head .sec-label{margin-bottom:10px}

/* ---------- feature grid ---------- */
.grid{display:grid;gap:18px}
.grid-2{grid-template-columns:repeat(2,1fr)}
.grid-3{grid-template-columns:repeat(3,1fr)}
.grid-4{grid-template-columns:repeat(4,1fr)}
@media (max-width:940px){.grid-4{grid-template-columns:repeat(2,1fr)}}
@media (max-width:720px){.grid-2,.grid-3,.grid-4{grid-template-columns:1fr}}
.feature{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);
  padding:24px;box-shadow:var(--shadow-sm);transition:transform .15s ease,box-shadow .15s ease,border-color .15s ease}
.feature:hover{transform:translateY(-3px);box-shadow:var(--shadow-md);border-color:var(--teal)}
.feature .ico{width:44px;height:44px;border-radius:12px;display:grid;place-items:center;font-size:22px;
  background:var(--teal-soft);margin-bottom:14px}
.feature h3{margin:0 0 6px;font-size:16.5px;font-weight:800}
.feature p{margin:0;color:var(--muted);font-size:14.5px}
.feature.plain{box-shadow:none;background:var(--surface)}

/* ---------- card grid ---------- */
.card{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);
  padding:26px;display:flex;flex-direction:column;gap:12px;box-shadow:var(--shadow-sm);
  transition:transform .15s ease,box-shadow .15s ease,border-color .15s ease;position:relative}
.card:hover{transform:translateY(-3px);box-shadow:var(--shadow-md);border-color:var(--teal)}
.card .ico{width:48px;height:48px;border-radius:12px;display:grid;place-items:center;font-size:24px;
  background:linear-gradient(135deg,var(--teal-soft),var(--indigo-soft))}
.card .eyebrow{font-size:11.5px;font-weight:800;letter-spacing:1px;text-transform:uppercase;color:var(--muted);margin:0}
.card .badge{position:absolute;top:16px;inset-inline-end:16px;font-size:11px;font-weight:800;
  color:var(--teal-strong);background:var(--teal-soft);border:1px solid rgba(20,166,166,.3);
  padding:4px 10px;border-radius:999px}
.card .badge.hot{color:var(--danger);background:#fdeef1;border-color:rgba(214,69,93,.3)}
.card .badge.free{color:var(--ok);background:#e9f7f0;border-color:rgba(35,155,107,.3)}
.card h3{margin:0;font-size:17px;font-weight:800}
.card p{margin:0;color:var(--muted);font-size:14.5px;flex:1}
.card .meta{display:flex;flex-wrap:wrap;gap:6px;margin-top:4px}
.card .meta span{font-size:12px;font-weight:700;color:var(--ink-soft);background:var(--panel);
  border:1px solid var(--border);padding:3px 9px;border-radius:999px}
.card .price-row{display:flex;align-items:baseline;gap:8px;margin-top:2px}
.card .price{font-size:20px;font-weight:800;color:var(--ink)}
.card .rating{font-size:12.5px;color:var(--warn);font-weight:800}
.card .row{display:flex;align-items:center;gap:8px}
.card-read{margin-top:10px;font-weight:700;font-size:14px}
.card-read::after{content:"\2192";margin-inline-start:6px;transition:transform .15s ease;display:inline-block}
html[dir=rtl] .card-read::after{content:"\2190"}
.card:hover .card-read::after{transform:translateX(-3px)}
html[dir=rtl] .card:hover .card-read::after{transform:translateX(3px)}

/* ---------- steps ---------- */
.steps{counter-reset:step;display:grid;gap:18px}
.step{display:flex;gap:18px;background:var(--surface);border:1px solid var(--border);
  border-radius:var(--radius);padding:22px;box-shadow:var(--shadow-sm)}
.step .num{counter-increment:step;flex:none;width:44px;height:44px;border-radius:12px;
  display:grid;place-items:center;font-weight:800;font-size:16px;color:#fff;
  background:linear-gradient(135deg,var(--teal),var(--indigo))}
.step .num::before{content:counter(step,padded-leading-zero)}
@counter-style padded-leading-zero{system:extends decimal;pad:2 "0"}
.step h3{margin:0 0 4px;font-size:16px;font-weight:800}
.step p{margin:0;color:var(--muted);font-size:14.5px}

/* ---------- stats ---------- */
.stats{display:grid;grid-template-columns:repeat(4,1fr);gap:18px}
@media (max-width:940px){.stats{grid-template-columns:repeat(2,1fr)}}
@media (max-width:560px){.stats{grid-template-columns:1fr 1fr}}
.stat{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);
  padding:26px;text-align:center;box-shadow:var(--shadow-sm)}
.stat .v{font-size:clamp(26px,3vw,38px);font-weight:800;letter-spacing:-.5px;
  background:linear-gradient(135deg,var(--teal),var(--indigo));-webkit-background-clip:text;background-clip:text;color:transparent}
.stat .l{color:var(--muted);font-size:13.5px;font-weight:600;margin-top:4px}

/* ---------- plans ---------- */
.plans{display:grid;grid-template-columns:repeat(4,1fr);gap:18px;align-items:stretch}
@media (max-width:1100px){.plans{grid-template-columns:repeat(2,1fr)}}
@media (max-width:640px){.plans{grid-template-columns:1fr}}
.plan{display:flex;flex-direction:column;gap:12px;background:var(--surface);border:1px solid var(--border);
  border-radius:var(--radius);padding:26px;box-shadow:var(--shadow-sm);position:relative}
.plan.featured{border:2px solid var(--teal);box-shadow:var(--shadow-md)}
.plan .badge{position:absolute;top:-13px;inset-inline-start:20px;font-size:11px;font-weight:800;color:#fff;
  background:linear-gradient(135deg,var(--teal),var(--indigo));padding:5px 12px;border-radius:999px}
.plan h3{margin:0;font-size:16px;font-weight:800}
.plan .desc{margin:0;color:var(--muted);font-size:13.5px;flex:1}
.plan .price-row{display:flex;align-items:baseline;gap:6px;margin-top:6px}
.plan .price{font-size:28px;font-weight:800;letter-spacing:-.5px}
.plan .per{color:var(--muted);font-size:12.5px}
.plan ul{list-style:none;margin:6px 0 0;padding:0;display:grid;gap:8px}
.plan li{font-size:14px;color:var(--ink-soft);display:flex;gap:9px;align-items:flex-start}
.plan li::before{content:"\2713";flex:none;width:18px;height:18px;border-radius:50%;font-size:11px;
  font-weight:800;display:grid;place-items:center;color:#fff;background:var(--teal);transform:translateY(1px)}
.plan .btn{margin-top:14px}

/* ---------- prose / split / faq ---------- */
.prose{max-width:760px}
.prose h3{font-size:19px;font-weight:800;margin:26px 0 8px}
.prose p{color:var(--ink-soft);margin:0 0 12px}
.prose ul{padding-inline-start:20px;color:var(--ink-soft)}
.prose li{margin-bottom:8px}
.prose .code{font-family:var(--mono);font-size:13.5px;background:var(--panel);border:1px solid var(--border);
  padding:1px 6px;border-radius:6px}
.split{display:grid;grid-template-columns:1fr 1fr;gap:24px;align-items:start}
@media (max-width:860px){.split{grid-template-columns:1fr}}
.split-box{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:26px;box-shadow:var(--shadow-sm)}
.split-box h3{margin:0 0 8px;font-size:16px;font-weight:800}
.split-box p{margin:0;color:var(--muted);font-size:14.5px}
.faq{display:grid;gap:12px;max-width:820px}
.faq details{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius-sm);
  padding:0;box-shadow:var(--shadow-sm);overflow:hidden}
.faq summary{list-style:none;cursor:pointer;display:flex;align-items:center;justify-content:space-between;
  gap:16px;padding:16px 20px;font-weight:700;font-size:15.5px}
.faq summary::after{content:"\002B";font-size:20px;color:var(--teal-strong);font-weight:800;transition:transform .15s ease;flex:none}
.faq details[open] summary::after{content:"\2212"}
.faq details .a{padding:0 20px 18px;color:var(--muted);font-size:14.5px;border-top:1px solid var(--border);padding-top:14px}
.faq details .a p{margin:0}
.faq details[open]{border-color:var(--teal)}

/* ---------- CTA band ---------- */
.cta{background:linear-gradient(135deg,#12395a 0%,#0f6b78 100%);border-radius:20px;color:#fff;
  padding:56px 40px;text-align:center;box-shadow:var(--shadow-lg);position:relative;overflow:hidden}
.cta::before{content:"";position:absolute;inset-inline-start:-80px;inset-block-start:-80px;width:260px;height:260px;
  border-radius:50%;background:rgba(255,255,255,.06)}
.cta::after{content:"";position:absolute;inset-inline-end:-120px;inset-block-end:-120px;width:320px;height:320px;
  border-radius:50%;background:rgba(20,166,166,.35)}
.cta h2{color:#fff;margin-bottom:12px}
.cta p{color:rgba(255,255,255,.85);font-size:16.5px;margin:0 auto 26px;max-width:560px}
.cta .hero-ctas{margin-bottom:0}

/* ---------- footer ---------- */
.site-footer{background:#0b1226;color:#9aa6bd;margin-top:0}
.footer-top{padding:56px 0 40px}
.footer-grid{display:grid;grid-template-columns:2fr 1fr 1fr 1fr;gap:32px}
@media (max-width:860px){.footer-grid{grid-template-columns:1fr 1fr}}
@media (max-width:520px){.footer-grid{grid-template-columns:1fr}}
.footer-brand .brand{color:#fff;margin-bottom:12px}
.footer-brand p{font-size:14px;margin:0 0 16px;max-width:300px}
.socials{display:flex;gap:10px}
.socials a{width:38px;height:38px;border-radius:10px;display:grid;place-items:center;font-size:16px;
  background:rgba(255,255,255,.06);border:1px solid rgba(255,255,255,.1);color:#cfd8ea}
.socials a:hover{background:var(--teal);color:#fff;border-color:var(--teal)}
.footer-col h4{color:#fff;font-size:13px;font-weight:800;letter-spacing:1.2px;text-transform:uppercase;margin:0 0 14px}
.footer-col a{display:block;padding:4px 0;font-size:14px;color:#9aa6bd}
.footer-col a:hover{color:#fff}
.footer-bottom{border-top:1px solid rgba(255,255,255,.08);padding:20px 0;font-size:13px}
.footer-bottom .container{display:flex;align-items:center;justify-content:space-between;gap:16px;flex-wrap:wrap}
.footer-bottom a{color:#7f8ba4;display:inline-flex;gap:18px}
.footer-bottom a:hover{color:#fff}
.footer-bottom .legals{display:flex;gap:18px}

/* ---------- WhatsApp float ---------- */
.wa-float{position:fixed;inset-block-end:22px;inset-inline-end:22px;z-index:60;width:56px;height:56px;
  border-radius:50%;background:var(--wa);color:#fff;display:grid;place-items:center;font-size:26px;
  box-shadow:0 10px 28px rgba(37,211,102,.4);transition:transform .15s ease}
.wa-float:hover{transform:translateY(-3px);color:#fff}
@media print{.wa-float,.site-header{display:none}}

/* ---------- misc ---------- */
.center{text-align:center}
.mt-8{margin-top:8px}.mt-16{margin-top:16px}.mt-24{margin-top:24px}.mt-40{margin-top:40px}
.section-ctas{display:flex;gap:12px;flex-wrap:wrap;justify-content:center;margin-top:36px}
.code-block{font-family:var(--mono);background:#0b1226;color:#7fe0dc;border-radius:var(--radius-sm);
  padding:18px 20px;font-size:13.5px;overflow-x:auto;line-height:1.7;border:1px solid rgba(255,255,255,.08)}
.key-val{display:grid;grid-template-columns:120px 1fr;gap:6px 18px;font-size:14.5px}
.key-val dt{font-weight:700;color:var(--ink)}
.key-val dd{margin:0;font-family:var(--mono);color:var(--teal-strong)}
@media (max-width:640px){
  .section{padding-block:48px}
  h1{letter-spacing:-.3px}
}
`
