package mainsite

// siteCSS holds the full stylesheet for the company website. Like the POS.Go
// landing it is served inline with no external requests, and it uses logical
// properties so the same sheet drives both LTR (English) and RTL (Arabic).
//
// Visual direction: an enterprise "Modernize" system with a first-class
// dark/light theme pair. The dark theme is the brand default (near-black
// #03010E canvas, violet #937AFF / #4D36D0 accent); the light theme keeps the
// same violet identity on a soft #f5f6fb canvas. Every colour is a token, so
// the two themes are one stylesheet switched by html[data-theme]. Inter Tight
// body + Urbanist headings, IBM Plex Sans Arabic for RTL.

const siteCSS = `:root{
  color-scheme:dark;
  --bg:#03010E;
  --bg-2:#0B0C17;
  --bg-3:#111222;
  --surface:rgba(255,255,255,.04);
  --surface-2:rgba(255,255,255,.06);
  --border:#202128;
  --border-strong:#2d2e39;
  --heading:#ffffff;
  --text:#DAD4DE;
  --text-strong:#E5E5E5;
  --muted:#817E84;
  --primary:#937AFF;
  --primary-light:#4D36D0;
  --primary-soft:rgba(147,122,255,.12);
  --primary-border:rgba(147,122,255,.3);
  --primary-ink:#c3b6ff;
  --secondary:#FF5353;
  --ok:#28C114;
  --wa:#25d366;
  --header:rgba(3,1,14,.8);
  --card-shadow:0 0 0 rgba(0,0,0,0);
  --grid-line:rgba(255,255,255,.045);
  --dropdown-shadow:0 30px 60px -20px rgba(0,0,0,.7);
  --hero-bg:
    radial-gradient(760px 380px at 50% -12%,rgba(77,54,208,.38),transparent 62%),
    radial-gradient(620px 340px at 86% 12%,rgba(147,122,255,.14),transparent 60%),
    var(--bg);
  --radius:14px;
  --radius-sm:8px;
  --radius-lg:24px;
  --radius-xl:36px;
  --pill:999px;
  --grad:linear-gradient(0deg,var(--primary-light),var(--primary));
  --font:"Inter Tight","IBM Plex Sans Arabic",ui-sans-serif,system-ui,"Segoe UI",Tahoma,Arial,sans-serif;
  --font-head:"Urbanist","IBM Plex Sans Arabic",ui-sans-serif,system-ui,"Segoe UI",Tahoma,Arial,sans-serif;
  --font-ar:"IBM Plex Sans Arabic","Inter Tight",system-ui,"Segoe UI",Tahoma,Arial,sans-serif;
  --mono:"IBM Plex Mono",ui-monospace,SFMono-Regular,Menlo,Consolas,"Liberation Mono",monospace;
}
html[data-theme=light]{
  color-scheme:light;
  --bg:#f5f6fb;
  --bg-2:#ffffff;
  --bg-3:#ffffff;
  --surface:#ffffff;
  --surface-2:#f1f2f9;
  --border:#e6e8f0;
  --border-strong:#d5d8e6;
  --heading:#0c0e18;
  --text:#3d4252;
  --text-strong:#1f2330;
  --muted:#6c7284;
  --primary:#5b3ef0;
  --primary-light:#4D36D0;
  --primary-soft:rgba(91,62,240,.08);
  --primary-border:rgba(91,62,240,.22);
  --primary-ink:#4D36D0;
  --secondary:#e23b3b;
  --ok:#1a9c0e;
  --header:rgba(255,255,255,.85);
  --card-shadow:0 14px 34px -22px rgba(20,24,60,.3);
  --grid-line:rgba(12,14,24,.05);
  --dropdown-shadow:0 24px 50px -18px rgba(20,24,60,.25);
  --hero-bg:
    radial-gradient(760px 380px at 50% -12%,rgba(91,62,240,.16),transparent 62%),
    radial-gradient(620px 340px at 86% 12%,rgba(147,122,255,.14),transparent 60%),
    var(--bg);
}
*{box-sizing:border-box}
html{scroll-behavior:smooth}
body{margin:0;font-family:var(--font);color:var(--text);background:var(--bg);
  line-height:1.65;font-size:16.5px;-webkit-font-smoothing:antialiased;text-rendering:optimizeLegibility}
html[dir=rtl] body{font-family:var(--font-ar);line-height:1.85}
h1,h2,h3,h4,h5,h6,.brand{font-family:var(--font-head);color:var(--heading)}
img,svg{max-width:100%}
a{color:var(--primary);text-decoration:none}
a:hover{color:var(--primary-light)}
::selection{background:var(--primary);color:#fff}
.container{max-width:1160px;margin-inline:auto;padding-inline:28px}
.section{padding-block:88px;scroll-margin-top:84px;position:relative}
main>section+section{border-top:1px solid var(--border)}
.section.alt,.section.panel{background:var(--bg-2)}
.gutter{border-block:1px solid var(--border);background:var(--bg-2)}

/* ---------- header ---------- */
.site-header{position:sticky;top:0;z-index:50;background:var(--header);
  backdrop-filter:blur(14px);-webkit-backdrop-filter:blur(14px);
  border-bottom:1px solid var(--border);transition:box-shadow .25s ease}
.site-header.scrolled{box-shadow:0 12px 34px -22px rgba(0,0,0,.55)}
.nav{display:flex;align-items:center;gap:18px;height:74px;position:relative}
.brand{display:flex;align-items:center;gap:11px;font-weight:700;color:var(--heading);font-size:19px;letter-spacing:-.01em}
.brand .mark{flex:none;width:38px;height:38px;display:block}
.brand-sub{display:block;font-size:10px;font-weight:500;color:var(--muted);
  letter-spacing:.12em;text-transform:uppercase;margin-top:1px;font-family:var(--font)}
.nav-links{display:flex;align-items:center;gap:2px;margin-inline-start:auto}
.nav-links a{padding:9px 14px;border-radius:var(--pill);color:var(--text-strong);
  font-weight:500;font-size:15px;transition:color .2s ease,background .2s ease}
.nav-links a:hover{color:var(--primary);background:var(--surface-2)}
.nav-links a.active{color:var(--primary);background:var(--primary-soft)}
.nav-cta{margin-inline-start:6px}
.nav-tools{display:flex;align-items:center;gap:10px}
.nav-links-lang{font-weight:600;font-size:14px;color:var(--text-strong);padding:8px 16px;
  border:1px solid var(--border-strong);border-radius:var(--pill);transition:border-color .2s ease,color .2s ease}
.nav-links-lang:hover{border-color:var(--primary);color:var(--primary)}
.theme-toggle{width:40px;height:40px;flex:none;border:1px solid var(--border-strong);border-radius:var(--pill);
  background:transparent;color:var(--text-strong);cursor:pointer;display:grid;place-items:center;
  transition:border-color .2s ease,color .2s ease,background .2s ease}
.theme-toggle:hover{border-color:var(--primary);color:var(--primary)}
.theme-toggle svg{width:18px;height:18px}
.theme-toggle .i-sun{display:grid}
.theme-toggle .i-moon{display:none}
html[data-theme=light] .theme-toggle .i-sun{display:none}
html[data-theme=light] .theme-toggle .i-moon{display:grid}
.dropdown{position:relative}
.dropdown>button{font:inherit;font-weight:500;font-size:15px;color:var(--text-strong);
  background:none;border:0;padding:9px 14px;border-radius:var(--pill);cursor:pointer;
  display:flex;align-items:center;gap:6px;transition:color .2s ease,background .2s ease}
.dropdown>button:hover,.dropdown:focus-within>button{color:var(--primary);background:var(--surface-2)}
.dropdown>button::after{content:"";width:7px;height:7px;border-right:1.5px solid currentColor;
  border-bottom:1.5px solid currentColor;transform:rotate(45deg) translateY(-2px);opacity:.7;transition:transform .2s ease}
.dropdown:hover>button::after,.dropdown:focus-within>button::after{transform:rotate(-135deg) translateY(-2px)}
.dropdown-menu{position:absolute;top:calc(100% + 10px);inset-inline-start:0;min-width:260px;
  background:var(--bg-2);border:1px solid var(--border);border-radius:var(--radius);
  padding:8px;display:none;box-shadow:var(--dropdown-shadow);z-index:60}
.dropdown:hover .dropdown-menu,.dropdown:focus-within .dropdown-menu{display:block}
.dropdown-menu a{display:block;padding:10px 14px;border-radius:var(--radius-sm);
  color:var(--text);font-weight:500;font-size:14.5px;transition:color .2s ease,background .2s ease}
.dropdown-menu a:hover{color:var(--primary);background:var(--surface-2)}
html[dir=rtl] .dropdown-menu{inset-inline-end:0;inset-inline-start:auto}
.nav-burger{display:none;background:none;border:1px solid var(--border-strong);
  border-radius:var(--radius-sm);padding:9px 11px;cursor:pointer;line-height:1}
.nav-burger span{display:block;width:20px;height:1.5px;background:var(--text-strong);border-radius:2px}
.nav-burger span+span{margin-top:5px}
#nav-toggle{display:none}
@media (max-width:960px){
  .nav-links{display:none;position:absolute;top:100%;inset-inline:0;background:var(--bg-2);
    border-bottom:1px solid var(--border);padding:14px 20px 20px;flex-direction:column;align-items:stretch;
    box-shadow:var(--dropdown-shadow)}
  #nav-toggle:checked ~ .nav-links{display:flex}
  .nav-tools{margin-inline-start:auto}
  .nav-burger{display:block}
  .dropdown-menu{position:static;display:block;box-shadow:none;border:0;padding:0 0 0 14px;background:none}
  .nav-links a{padding:11px 12px}
  .nav-cta{display:none}
}

/* ---------- buttons ---------- */
.btn{display:inline-flex;align-items:center;justify-content:center;gap:9px;font-weight:500;
  font-size:15px;font-family:inherit;padding:14px 30px;border-radius:var(--pill);
  border:1.5px solid transparent;cursor:pointer;transition:filter .25s ease,box-shadow .25s ease,border-color .25s ease,background .25s ease,color .25s ease;white-space:nowrap}
.btn-primary{background:var(--grad);color:#fff}
.btn-primary:hover{filter:brightness(1.14);color:#fff;box-shadow:0 10px 30px -12px var(--primary-light)}
.btn-ghost{background:transparent;color:var(--heading);border-color:var(--border-strong)}
.btn-ghost:hover{background:var(--primary);border-color:var(--primary);color:#fff;box-shadow:0 10px 26px -14px var(--primary)}
.btn-wa{background:transparent;color:var(--wa);border-color:rgba(37,211,102,.5)}
.btn-wa:hover{background:var(--wa);color:#fff;border-color:var(--wa)}

/* ---------- hero ---------- */
.hero{padding-block:112px 96px;position:relative;overflow:hidden;border-bottom:1px solid var(--border);
  background:var(--hero-bg)}
.hero::before{content:"";position:absolute;inset:0;pointer-events:none;opacity:.6;
  background-image:linear-gradient(var(--grid-line) 1px,transparent 1px),
    linear-gradient(90deg,var(--grid-line) 1px,transparent 1px);background-size:56px 56px;
  mask-image:radial-gradient(120% 80% at 50% 0,#000,transparent 75%);
  -webkit-mask-image:radial-gradient(120% 80% at 50% 0,#000,transparent 75%)}
.hero .container{max-width:900px;text-align:center;position:relative;z-index:1}
.eyebrow{display:inline-flex;align-items:center;gap:8px;font-size:13px;font-weight:500;
  color:var(--primary-ink);background:var(--primary-soft);border:1px solid var(--primary-border);
  padding:8px 16px;border-radius:var(--pill);margin-bottom:26px}
h1{font-size:clamp(34px,5.8vw,60px);line-height:1.06;margin:0 0 22px;letter-spacing:-.02em;font-weight:700}
.lead{font-size:clamp(17px,2vw,20px);color:var(--text);max-width:700px;margin-inline:auto;margin-bottom:36px;line-height:1.6}
.hero-ctas{display:flex;gap:14px;justify-content:center;flex-wrap:wrap;margin-bottom:40px}
.hero-tiles{display:flex;flex-wrap:wrap;gap:10px;justify-content:center}
.hero-tiles span{display:inline-flex;align-items:center;gap:7px;font-size:13.5px;font-weight:500;
  color:var(--text-strong);background:var(--surface);border:1px solid var(--border);
  padding:9px 16px;border-radius:var(--pill)}
.hero-note{margin-top:28px;font-size:14px;color:var(--muted)}
.hero-note a{font-weight:600}
@media (max-width:640px){.hero{padding-block:68px 56px}}

/* ---------- section header ---------- */
.sec-label{font-size:12px;font-weight:600;letter-spacing:.16em;text-transform:uppercase;
  color:var(--primary);margin-bottom:14px;display:inline-block}
.sec-head{margin-bottom:52px;max-width:660px}
.center .sec-head{margin-inline:auto;text-align:center}
h2{font-size:clamp(28px,4vw,44px);line-height:1.12;margin:0 0 14px;letter-spacing:-.02em;font-weight:700}
.sec-head p{color:var(--muted);font-size:17px;margin:0;line-height:1.6}

/* ---------- feature grid ---------- */
.grid{display:grid;gap:22px}
.grid-2{grid-template-columns:repeat(2,1fr)}
.grid-3{grid-template-columns:repeat(3,1fr)}
.grid-4{grid-template-columns:repeat(4,1fr)}
@media (max-width:960px){.grid-4{grid-template-columns:repeat(2,1fr)}}
@media (max-width:720px){.grid-2,.grid-3,.grid-4{grid-template-columns:1fr}}
.feature{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius-lg);
  padding:30px;box-shadow:var(--card-shadow);transition:border-color .2s ease,background .2s ease,transform .2s ease}
.feature:hover{border-color:var(--primary);transform:translateY(-3px)}
.feature h3{margin:0 0 9px;font-size:19px;font-weight:600;letter-spacing:-.01em}
.feature p{margin:0;color:var(--muted);font-size:15px;line-height:1.65}
.feature.plain{border:0;background:none;padding:0;box-shadow:none}

/* ---------- card grid ---------- */
.card{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius-lg);
  padding:30px;display:flex;flex-direction:column;gap:13px;box-shadow:var(--card-shadow);
  transition:border-color .2s ease,background .2s ease,transform .2s ease,box-shadow .2s ease;position:relative}
.card:hover{border-color:var(--primary);transform:translateY(-3px)}
.card .eyebrow{font-size:11px;font-weight:600;letter-spacing:.14em;text-transform:uppercase;color:var(--primary);margin:0}
.card .badge{position:absolute;top:22px;inset-inline-end:22px;font-size:11px;font-weight:600;
  color:var(--primary);border:1px solid var(--primary-border);padding:4px 12px;border-radius:var(--pill)}
.card .badge.hot{color:var(--secondary);border-color:rgba(255,83,83,.4)}
.card .badge.free{color:var(--ok);border-color:rgba(40,193,20,.4)}
.card h3{margin:0;font-size:20px;font-weight:600;letter-spacing:-.01em}
.card p{margin:0;color:var(--muted);font-size:15px;flex:1;line-height:1.65}
.card .meta{display:flex;flex-wrap:wrap;gap:6px;margin-top:2px}
.card .meta span{font-size:12px;font-weight:500;color:var(--text);border:1px solid var(--border);
  padding:4px 11px;border-radius:var(--pill)}
.card .price-row{display:flex;align-items:baseline;gap:9px;margin-top:4px}
.card .price{font-size:24px;font-weight:600;color:var(--heading)}
.card .rating{font-size:12.5px;color:var(--primary);font-weight:600}
.card .row{display:flex;align-items:center;gap:8px}
.card-read{margin-top:12px;font-weight:600;font-size:14.5px;color:var(--primary)}
.card-read::after{content:"\2192";margin-inline-start:7px;transition:transform .2s ease;display:inline-block}
html[dir=rtl] .card-read::after{content:"\2190"}
.card:hover .card-read::after{transform:translateX(4px)}
html[dir=rtl] .card:hover .card-read::after{transform:translateX(-4px)}

/* ---------- steps ---------- */
.steps{counter-reset:step;display:grid;gap:0}
.step{display:flex;gap:24px;border-top:1px solid var(--border);padding:28px 0}
.step .num{counter-increment:step;flex:none;font-family:var(--mono);font-size:13px;font-weight:500;
  color:var(--primary);padding-top:3px;min-width:34px}
.step .num::before{content:counter(step,decimal-leading-zero)}
.step h3{margin:0 0 5px;font-size:18px;font-weight:600;letter-spacing:-.01em}
.step p{margin:0;color:var(--muted);font-size:15px;line-height:1.65}

/* ---------- stats ---------- */
.stats{display:grid;grid-template-columns:repeat(4,1fr);gap:22px}
@media (max-width:960px){.stats{grid-template-columns:repeat(2,1fr)}}
@media (max-width:560px){.stats{grid-template-columns:1fr 1fr}}
.stat{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius-lg);
  padding:30px 26px;text-align:center;box-shadow:var(--card-shadow)}
.stat .v{font-size:clamp(30px,3.6vw,48px);font-weight:700;letter-spacing:-.03em;line-height:1.05;font-family:var(--font-head);
  background:var(--grad);-webkit-background-clip:text;background-clip:text;color:transparent}
html[dir=rtl] .stat .v{letter-spacing:0}
.stat .l{color:var(--muted);font-size:14px;font-weight:500;margin-top:6px}

/* ---------- plans ---------- */
.plans{display:grid;grid-template-columns:repeat(4,1fr);gap:22px;align-items:stretch}
@media (max-width:1100px){.plans{grid-template-columns:repeat(2,1fr)}}
@media (max-width:640px){.plans{grid-template-columns:1fr}}
.plan{display:flex;flex-direction:column;gap:13px;background:var(--surface);
  border:1px solid var(--border);border-radius:var(--radius-lg);padding:30px;position:relative;box-shadow:var(--card-shadow)}
.plan.featured{border-color:var(--primary);background:var(--primary-soft);
  box-shadow:0 0 44px -18px var(--primary-light)}
.plan .badge{position:absolute;top:-12px;inset-inline-start:28px;font-size:11px;font-weight:600;color:#fff;
  background:var(--grad);padding:5px 14px;border-radius:var(--pill)}
.plan h3{margin:0;font-size:18px;font-weight:600}
.plan .desc{margin:0;color:var(--muted);font-size:14px;flex:1;line-height:1.6}
.plan .price-row{display:flex;align-items:baseline;gap:7px;margin-top:6px}
.plan .price{font-size:34px;font-weight:700;letter-spacing:-.03em;color:var(--heading)}
html[dir=rtl] .plan .price{letter-spacing:0}
.plan .per{color:var(--muted);font-size:12.5px}
.plan ul{list-style:none;margin:8px 0 0;padding:0;display:grid;gap:10px}
.plan li{font-size:14.5px;color:var(--text);display:flex;gap:10px;align-items:flex-start;line-height:1.5}
.plan li::before{content:"";flex:none;width:16px;height:16px;margin-top:3px;border-radius:50%;
  background:var(--primary-soft);border:1px solid var(--primary);position:relative}
.plan .btn{margin-top:16px}

/* ---------- prose / split / faq ---------- */
.prose{max-width:760px}
.prose h3{font-size:22px;font-weight:600;margin:34px 0 10px;letter-spacing:-.01em}
.prose p{color:var(--text);margin:0 0 14px;line-height:1.7}
.prose ul{padding-inline-start:20px;color:var(--text);line-height:1.7}
.prose li{margin-bottom:9px}
.prose .code{font-family:var(--mono);font-size:13.5px;background:var(--surface-2);border:1px solid var(--border);
  padding:1px 7px;border-radius:var(--radius-sm);color:var(--primary)}
.split{display:grid;grid-template-columns:1fr 1fr;gap:22px;align-items:start}
@media (max-width:860px){.split{grid-template-columns:1fr}}
.split-box{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius-lg);padding:30px;box-shadow:var(--card-shadow)}
.split-box h3{margin:0 0 10px;font-size:18px;font-weight:600}
.split-box p{margin:0;color:var(--muted);font-size:15px;line-height:1.7}
.faq{display:grid;gap:14px;max-width:820px}
.faq details{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);
  padding:0;overflow:hidden;transition:border-color .2s ease;box-shadow:var(--card-shadow)}
.faq details[open]{border-color:var(--primary)}
.faq summary{list-style:none;cursor:pointer;display:flex;align-items:center;justify-content:space-between;
  gap:16px;padding:20px 24px;font-weight:600;font-size:16.5px;color:var(--heading)}
.faq summary::-webkit-details-marker{display:none}
.faq summary::after{content:"+";font-size:22px;color:var(--primary);font-weight:400;flex:none;transition:transform .2s ease}
.faq details[open] summary::after{content:"\2212"}
.faq details .a{padding:0 24px 22px;color:var(--muted);font-size:15px;line-height:1.7}
.faq details .a p{margin:0}

/* ---------- CTA band ---------- */
.cta{background:var(--bg-2);border:1px solid var(--border);border-radius:var(--radius-xl);color:var(--heading);
  padding:64px 44px;text-align:center;position:relative;overflow:hidden;box-shadow:var(--card-shadow)}
.cta::before{content:"";position:absolute;inset:0;pointer-events:none;
  background:radial-gradient(600px 300px at 50% -30%,var(--primary-soft),transparent 65%)}
.cta::after{content:"";position:absolute;inset-inline-end:-140px;inset-block-end:-160px;width:380px;height:380px;
  border-radius:50%;background:radial-gradient(circle,var(--primary-soft),transparent 70%);pointer-events:none}
.cta>*{position:relative;z-index:1}
.cta h2{margin-bottom:14px}
.cta p{color:var(--text);font-size:17px;margin:0 auto 30px;max-width:560px;line-height:1.6}
.cta .hero-ctas{margin-bottom:0}

/* ---------- footer (always dark, matches the brand) ---------- */
.site-footer{background:#060610;color:#9aa0b4;border-top:1px solid #1b1c2a}
.footer-top{padding:72px 0 48px}
.footer-grid{display:grid;grid-template-columns:2fr 1fr 1fr 1fr;gap:36px}
@media (max-width:860px){.footer-grid{grid-template-columns:1fr 1fr}}
@media (max-width:520px){.footer-grid{grid-template-columns:1fr}}
.footer-brand .brand{color:#fff;margin-bottom:16px}
.footer-brand .brand-sub{color:#767c93}
.footer-brand p{font-size:14.5px;margin:0 0 20px;max-width:300px;line-height:1.7}
.socials{display:flex;gap:10px}
.socials a{width:42px;height:42px;border-radius:50%;display:grid;place-items:center;
  background:rgba(255,255,255,.06);color:#fff;transition:background .2s ease,color .2s ease}
.socials a:hover{background:var(--primary);color:#fff}
.socials svg{width:18px;height:18px}
.footer-col h4{color:#fff;font-size:13px;font-weight:600;letter-spacing:.1em;text-transform:uppercase;
  margin:0 0 18px;font-family:var(--font-head)}
.footer-col a{display:block;padding:5px 0;font-size:14.5px;color:#9aa0b4;transition:color .2s ease}
.footer-col a:hover{color:#fff}
.footer-bottom{border-top:1px solid #1b1c2a;padding:24px 0;font-size:13.5px}
.footer-bottom .container{display:flex;align-items:center;justify-content:space-between;gap:16px;flex-wrap:wrap}
.footer-bottom .legals{display:flex;gap:20px}
.footer-bottom a{color:#9aa0b4}
.footer-bottom a:hover{color:#fff}

/* ---------- WhatsApp float ---------- */
.wa-float{position:fixed;inset-block-end:24px;inset-inline-end:24px;z-index:60;width:56px;height:56px;
  border-radius:50%;background:var(--wa);color:#fff;display:grid;place-items:center;
  transition:transform .2s ease;box-shadow:0 12px 32px -10px rgba(37,211,102,.7)}
.wa-float:hover{transform:translateY(-3px);color:#fff}
.wa-float svg{width:26px;height:26px}
@media print{.wa-float,.site-header{display:none}}

/* ---------- misc ---------- */
.center{text-align:center}
.mt-8{margin-top:8px}.mt-16{margin-top:16px}.mt-24{margin-top:24px}.mt-40{margin-top:40px}
.section-ctas{display:flex;gap:14px;flex-wrap:wrap;justify-content:center;margin-top:44px}
.code-block{font-family:var(--mono);background:#0b0c17;color:var(--primary);border:1px solid var(--border);
  border-radius:var(--radius);padding:20px 22px;font-size:13.5px;overflow-x:auto;line-height:1.7}
.key-val{display:grid;grid-template-columns:120px 1fr;gap:8px 20px;font-size:15px}
.key-val dt{font-weight:600;color:var(--heading)}
.key-val dd{margin:0;font-family:var(--mono);color:var(--primary)}
@media (max-width:640px){.section{padding-block:60px}}

/* Arabic must never carry Latin letter-spacing (it breaks glyph joining). */
html[dir=rtl] h1,html[dir=rtl] h2,html[dir=rtl] h3,html[dir=rtl] h4,
html[dir=rtl] .brand,html[dir=rtl] .brand-sub,html[dir=rtl] .sec-label,
html[dir=rtl] .eyebrow,html[dir=rtl] .footer-col h4,html[dir=rtl] .btn,
html[dir=rtl] .card .eyebrow,html[dir=rtl] .lead,html[dir=rtl] .card h3,
html[dir=rtl] .feature h3,html[dir=rtl] .step h3,html[dir=rtl] .plan h3,
html[dir=rtl] .nav-links a{letter-spacing:normal}
`
