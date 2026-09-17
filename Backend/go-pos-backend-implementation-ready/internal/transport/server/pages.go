package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Public brand + privacy pages served at the API root and /private. nginx on
// api.xamltech.com proxies everything except /admin/ (the React admin SPA) and
// /.well-known/ to this app, so GET / is the branded landing visitors see when
// they open the domain, and GET /private is the privacy policy pointed to from
// the Android app and store listing.

const posGoStyling = `
:root {
  --bg: #08131f;
  --card: #0f2233;
  --border: #1d3a52;
  --text: #e6eef5;
  --muted: #93a8ba;
  --accent: #22c55e;
  --accent-dark: #15803d;
}
* { box-sizing: border-box; }
html, body { margin: 0; padding: 0; }
body {
  font-family: 'Segoe UI', system-ui, -apple-system, Roboto, sans-serif;
  background:
    radial-gradient(900px 420px at 85% -10%, rgba(34,197,94,.16), transparent 60%),
    radial-gradient(700px 380px at -10% 110%, rgba(14,116,144,.18), transparent 60%),
    var(--bg);
  color: var(--text);
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
main { flex: 1; width: 100%; max-width: 760px; margin: 0 auto; padding: 48px 20px; }
.card {
  background: linear-gradient(180deg, var(--card), #0c1d2c);
  border: 1px solid var(--border);
  border-radius: 18px;
  padding: 44px 36px;
  text-align: center;
  box-shadow: 0 18px 50px rgba(0,0,0,.35);
}
.logo { width: 84px; height: 84px; margin: 0 auto 6px; display: block; }
h1 { font-size: 30px; margin: 10px 0 8px; font-weight: 700; letter-spacing: .3px; }
h2 { font-size: 22px; margin: 0 0 14px; }
p.tagline { color: var(--muted); margin: 0 auto 28px; font-size: 16px; max-width: 520px; }
.actions { display: flex; gap: 12px; justify-content: center; flex-wrap: wrap; margin-bottom: 26px; }
a.btn {
  display: inline-block;
  padding: 13px 26px;
  border-radius: 12px;
  text-decoration: none;
  font-weight: 600;
  font-size: 15px;
  border: 1px solid var(--border);
}
a.primary { background: var(--accent); color: #052e12; border-color: var(--accent); }
a.primary:hover { background: var(--accent-dark); color: #fff; }
a.ghost { background: transparent; color: var(--text); }
a.ghost:hover { border-color: var(--accent); color: var(--accent); }
footer { color: var(--muted); font-size: 13px; text-align: center; padding: 14px 0 26px; }
.muted { color: var(--muted); }
.prose { text-align: left; color: #c7d6e2; line-height: 1.65; font-size: 15px; }
.prose h2 { color: var(--text); }
.prose p { margin: 0 0 16px; }
.prose ul { margin: 0 0 18px; padding-left: 22px; }
.prose li { margin-bottom: 8px; }
a { color: var(--accent); }
`

const posGoLogo = `
<svg class="logo" viewBox="0 0 96 96" xmlns="http://www.w3.org/2000/svg">
  <rect x="4" y="4" width="88" height="88" rx="22" fill="#0f2233" stroke="#22c55e" stroke-width="4"/>
  <text x="48" y="68" text-anchor="middle" font-family="Verdana, sans-serif" font-size="52" font-weight="700" fill="#22c55e">P</text>
  <circle cx="72" cy="24" r="6" fill="#e2fde9"/>
</svg>`

const posGoIndexHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
    <meta name="description" content="POS.Go — point of sale for restaurants, shops and every business. Sign in to the admin console."/>
    <link rel="icon" type="image/svg+xml" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 96 96'%3E%3Crect x='4' y='4' width='88' height='88' rx='22' fill='%230f2233' stroke='%2322c55e' stroke-width='4'/%3E%3Ctext x='48' y='68' text-anchor='middle' font-family='Verdana' font-size='52' font-weight='700' fill='%2322c55e'%3EP%3C/text%3E%3C/svg%3E"/>
    <title>POS.Go — Powers every business</title>
    <style>` + posGoStyling + `</style>
  </head>
  <body>
    <main>
      <div class="card">
        ` + posGoLogo + `
        <h1>POS.Go</h1>
        <p class="tagline">Point of sale that works offline-first: restaurants, retail, pharmacy and more — backed by XAMLtech.</p>
        <div class="actions">
          <a class="btn primary" href="/admin/">Admin sign in</a>
          <a class="btn ghost" href="/private">Privacy policy</a>
        </div>
        <p class="muted" style="font-size:13px;margin:0">The console manages subscriptions, plans &amp; billing across all tenants.</p>
      </div>
    </main>
    <footer>© <span id="year"></span> XAMLtech &middot; <a href="/private">Privacy</a> &middot; <a href="/admin/">Admin sign in</a></footer>
    <script>document.getElementById('year').textContent = new Date().getFullYear();</script>
  </body>
</html>`

const posGoPrivacyHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
    <meta name="robots" content="noindex"/>
    <title>Privacy Policy — POS.Go</title>
    <style>` + posGoStyling + `</style>
  </head>
  <body>
    <main>
      <div class="card">
        ` + posGoLogo + `
        <h1>Privacy Policy</h1>
        <p class="tagline">POS.Go by XAMLtech — last updated January 1, 2026.</p>
        <div class="prose">
          <h2>1. Who we are</h2>
          <p>POS.Go is a point-of-sale service operated by XAMLtech. This policy explains what personal data we process, why,
          and the choices you have. By using POS.Go you agree to the practices described here.</p>

          <h2>2. Data we collect</h2>
          <ul>
            <li><strong>Account data:</strong> name, email, password (stored hashed), tenant and role, access level and permission settings.</li>
            <li><strong>Terminal &amp; device data:</strong> a generated device identifier and the device name you choose for each register.</li>
            <li><strong>Business data:</strong> products, categories, customers, sales, payments, invoices, inventory and receipts you enter in POS.Go.</li>
            <li><strong>Usage &amp; technical data:</strong> crash reports, screen names, app version, and basic request logs (IP address) for security.</li>
          </ul>

          <h2>3. How we use data</h2>
          <p>We use your data only to run the service: authenticate users, sync data across your devices, produce receipts and reports,
          apply your discount and access rules, prevent fraud and abuse, and provide support. We never sell your data.</p>

          <h2>4. Sharing</h2>
          <p>We share data only with sub-processors needed to host and operate the service (for example the cloud provider hosting our
          servers). Those parties are bound by appropriate confidentiality and security commitments. Business data stays isolated per
          tenant and is never shared between tenants.</p>

          <h2>5. Retention</h2>
          <p>We keep your account and transaction data for as long as your tenant is active, and for the periods required by tax law for
          receipt and invoice records. Telemetry and crash data are retained for shorter operational periods.</p>

          <h2>6. Security</h2>
          <p>Traffic is encrypted in transit (TLS), passwords and PINs are stored hashed, access to tenant data is enforced by
          row-level security, and credentials are kept in secure storage. You can remove users, close registers and export your data
          from the app at any time.</p>

          <h2>7. Your rights</h2>
          <p>You may access, correct, export or delete your personal data, and object to processing, by contacting us. Account deletion
          removes your personal data subject to legal retention periods.</p>

          <h2>8. Contact</h2>
          <p>Questions about this policy or your data: <a href="mailto:support@xamltech.com">support@xamltech.com</a>.</p>
        </div>
        <p class="muted" style="font-size:13px;margin:16px 0 0"><a href="/">← Back to POS.Go</a></p>
      </div>
    </main>
    <footer>© <span id="year"></span> XAMLtech &middot; <a href="/admin/">Admin sign in</a></footer>
    <script>document.getElementById('year').textContent = new Date().getFullYear();</script>
  </body>
</html>`

// registerSitePages mounts the public brand landing and privacy pages. They
// need no auth or database, matching how the OpenAPI generator walks the route
// table offline.
func registerSitePages(engine *gin.Engine) {
	engine.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(posGoIndexHTML))
	})
	engine.GET("/private", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(posGoPrivacyHTML))
	})
}
