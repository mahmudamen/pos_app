package server

import (
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Public password-recovery pages.
//
// The reset link is a plain GET, because it has to open in whatever browser
// the recipient is using — the person who forgot their password is usually on
// a different device than the one that is signed out. Both pages are static,
// DB-free and auth-free (they carry no data: the token travels in the query
// string and the POST goes to /v1/auth/*), and they are `noindex` because a
// recovery form has no business in a search index.
//
// They are deliberately NOT part of the marketing site template: a password
// form does not want the landing page's nav, gallery or plans grid, and the
// smaller surface is less to keep in step when the brand page changes.

// authPageCSS is the whole stylesheet for both recovery pages — one card, one
// form, one message slot. Dark-mode is handled by color-scheme + the same
// neutral palette the site uses.
const authPageCSS = `
:root {
  --bg: #f7f9fb; --card: #ffffff; --line: #e3e9ef; --text: #0f2233;
  --text-3: #5b6b7c; --accent: #22c55e; --accent-ink: #06340f;
  --danger: #b42318; --ok: #067647;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #0b111a; --card: #121a26; --line: #22303f; --text: #e8eef4;
    --text-3: #93a4b5; --accent: #34d399; --accent-ink: #04150c;
    --danger: #fda4a4; --ok: #86efac;
  }
}
* { box-sizing: border-box; }
body {
  margin: 0; min-height: 100vh; background: var(--bg); color: var(--text);
  font: 16px/1.6 system-ui, -apple-system, "Segoe UI", Roboto, "Noto Sans Arabic", sans-serif;
  display: flex; align-items: center; justify-content: center; padding: 24px;
}
.wrap { width: 100%; max-width: 440px; }
.brand {
  display: flex; align-items: center; gap: 10px; justify-content: center;
  text-decoration: none; color: var(--text); margin-bottom: 18px;
}
.brand svg { width: 34px; height: 34px; }
.brand .brand-mark { width: 34px; height: 34px; display: block; border-radius: 10px; }
.brand b { font-size: 19px; letter-spacing: .01em; }
.brand small { display: block; font-size: 11px; font-weight: 600; color: var(--text-3); letter-spacing: .08em; text-transform: uppercase; }
.card {
  background: var(--card); border: 1px solid var(--line); border-radius: 16px;
  padding: 26px 24px 22px; box-shadow: 0 12px 30px rgba(15, 34, 51, .06);
}
h1 { margin: 0 0 6px; font-size: 22px; line-height: 1.3; }
.lead { margin: 0 0 20px; color: var(--text-3); font-size: 14.5px; }
label { display: block; font-size: 13px; font-weight: 600; margin: 0 0 6px; }
input[type="text"], input[type="email"], input[type="password"] {
  width: 100%; padding: 11px 12px; margin-bottom: 14px; border: 1px solid var(--line);
  border-radius: 10px; background: var(--bg); color: var(--text); font-size: 15.5px;
}
input:focus { outline: 2px solid var(--accent); outline-offset: 1px; border-color: transparent; }
button {
  width: 100%; padding: 12px 14px; border: 0; border-radius: 10px; cursor: pointer;
  background: var(--accent); color: var(--accent-ink); font-size: 15.5px; font-weight: 700;
}
button[disabled] { opacity: .6; cursor: default; }
.msg { display: none; margin: 0 0 16px; padding: 11px 12px; border-radius: 10px; font-size: 14.5px; border: 1px solid var(--line); }
.msg.show { display: block; }
.msg.ok { color: var(--ok); background: color-mix(in srgb, var(--ok) 10%, transparent); border-color: color-mix(in srgb, var(--ok) 35%, transparent); }
.msg.err { color: var(--danger); background: color-mix(in srgb, var(--danger) 10%, transparent); border-color: color-mix(in srgb, var(--danger) 35%, transparent); }
.alt { margin: 16px 0 0; text-align: center; font-size: 14px; color: var(--text-3); }
.alt a { color: var(--text); font-weight: 600; }
.foot { margin: 16px 0 0; text-align: center; font-size: 13px; color: var(--text-3); }
.foot a { color: var(--text-3); }
[dir="rtl"] body, [dir="rtl"] .card { text-align: right; }
`

// authStrings carries the copy for both recovery pages in the two site
// languages. TestAuthPageStringsCoverBothLanguages asserts the key sets match,
// so a template can never render a literal "<no value>".
type authStrings map[string]string

var authPagesText = map[string]authStrings{
	"en": {
		"brandSub": "XAMLtech", "titleForgot": "Reset your password",
		"leadForgot": "Enter your store and the email you sign in with. If they match an account, we send a reset link.",
		"storeLabel": "Store (ID or slug)", "storePh": "demo-restaurant",
		"emailLabel": "Email", "emailPh": "you@store.com", "submit": "Send reset link",
		"sending": "Sending…", "sent": "Check your inbox — the link expires in 30 minutes. Nothing arrived? Check the store and email again, or ask support.",
		"rateLimited":  "Too many attempts from this network. Wait a few minutes and try again.",
		"genericError": "Something went wrong. Please try again.",
		"titleReset":   "Choose a new password",
		"leadReset":    "At least 8 characters. Everyone signed in on this account is signed out once the password changes.",
		"pwLabel":      "New password", "pwPh": "At least 8 characters", "pw2Label": "Confirm new password",
		"submitReset": "Update password", "saving": "Saving…",
		"short": "Use at least 8 characters.", "mismatch": "The two passwords do not match.",
		"updated":    "Password updated. Sign in again on your device.",
		"invalid":    "This reset link is invalid or has expired. Request a new one.",
		"requestNew": "Request a new link", "home": "Back to home",
		"help": "Need help? support@xamltech.com", "rights": "All rights reserved",
	},
	"ar": {
		"brandSub": "زامتِك", "titleForgot": "إعادة تعيين كلمة المرور",
		"leadForgot": "أدخل اسم المتجر والبريد الذي تسجّل الدخول به. إذا كانا مسجّلين فسنرسل رابط إعادة التعيين.",
		"storeLabel": "المتجر (المعرّف أو الاسم)", "storePh": "demo-restaurant",
		"emailLabel": "البريد الإلكتروني", "emailPh": "you@store.com", "submit": "إرسال رابط الاستعادة",
		"sending": "جارٍ الإرسال…", "sent": "تحقق من بريدك — الرابط صالح لمدة 30 دقيقة. لم يصل شيء؟ تأكد من المتجر والبريد مرة أخرى، أو تواصل مع الدعم.",
		"rateLimited":  "محاولات كثيرة من هذه الشبكة. انتظر دقائق ثم أعد المحاولة.",
		"genericError": "حدث خطأ ما. حاول مرة أخرى.",
		"titleReset":   "اختر كلمة مرور جديدة",
		"leadReset":    "٨ أحرف على الأقل. سيتم تسجيل الخروج لجميع الأجهزة بعد تغيير كلمة المرور.",
		"pwLabel":      "كلمة المرور الجديدة", "pwPh": "٨ أحرف على الأقل", "pw2Label": "تأكيد كلمة المرور",
		"submitReset": "تحديث كلمة المرور", "saving": "جارٍ الحفظ…",
		"short": "استخدم ٨ أحرف على الأقل.", "mismatch": "كلمتا المرور غير متطابقتين.",
		"updated":    "تم تحديث كلمة المرور. سجّل الدخول مرة أخرى على جهازك.",
		"invalid":    "رابط الاستعادة غير صالح أو منتهي الصلاحية. اطلب رابطًا جديدًا.",
		"requestNew": "طلب رابط جديد", "home": "العودة إلى الرئيسية",
		"help": "تحتاج مساعدة؟ support@xamltech.com", "rights": "جميع الحقوق محفوظة",
	},
}

// authPageData is what both recovery templates render from.
type authPageData struct {
	Lang  string
	Dir   string
	Title string
	T     authStrings
	Token string
	Year  int
}

const authBrandMark = `<img class="brand-mark" src="/icon.png" width="34" height="34" alt=""/>`

const authHead = `<!doctype html>
<html lang="{{.Lang}}" dir="{{.Dir}}">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<meta name="color-scheme" content="light dark"/>
<meta name="robots" content="noindex, nofollow"/>
<meta name="referrer" content="no-referrer"/>
<title>{{.Title}} — POS.Go</title>
<style>` + authPageCSS + `</style>
</head>
<body>
<main class="wrap">
<a class="brand" href="/">` + authBrandMark + `<span><b>POS.Go<small>{{.T.brandSub}}</small></b></span></a>
<section class="card">`

const authFoot = `</section>
<p class="foot">{{.T.help}}</p>
<p class="foot">© {{.Year}} XAMLtech · {{.T.rights}} · <a href="/private">{{if eq .Lang "ar"}}الخصوصية{{else}}Privacy{{end}}</a></p>
</main>
</body>
</html>`

// The forgot page posts the (store, email) pair the sign-in screen uses; the
// answer is uniform either way, so this form cannot be used to probe which
// stores or addresses exist.
const authForgotBody = `<h1>{{.T.titleForgot}}</h1>
<p class="lead">{{.T.leadForgot}}</p>
<p class="msg" id="msg" role="status"></p>
<form id="form" novalidate>
  <label for="tenant">{{.T.storeLabel}}</label>
  <input id="tenant" name="tenant" type="text" autocomplete="organization" placeholder="{{.T.storePh}}" required/>
  <label for="email">{{.T.emailLabel}}</label>
  <input id="email" name="email" type="email" autocomplete="email" inputmode="email" placeholder="{{.T.emailPh}}" required/>
  <button id="submit" type="submit">{{.T.submit}}</button>
</form>
<p class="alt"><a href="/">{{.T.home}}</a></p>
<script>
(function () {
  var form = document.getElementById('form');
  var msg = document.getElementById('msg');
  var submit = document.getElementById('submit');
  var label = submit.textContent;
  function show(text, ok) {
    msg.textContent = text;
    msg.className = 'msg show ' + (ok ? 'ok' : 'err');
  }
  form.addEventListener('submit', function (e) {
    e.preventDefault();
    var tenant = document.getElementById('tenant').value.trim();
    var email = document.getElementById('email').value.trim();
    if (!tenant || !email) { return; }
    submit.disabled = true;
    submit.textContent = '{{.T.sending}}';
    fetch('/v1/auth/forgot-password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tenant_id: tenant, email: email })
    }).then(function (r) {
      if (r.status === 202) { show('{{.T.sent}}', true); form.reset(); return; }
      if (r.status === 429) { show('{{.T.rateLimited}}', false); return; }
      throw new Error('status ' + r.status);
    }).catch(function () { show('{{.T.genericError}}', false); })
      .finally(function () { submit.disabled = false; submit.textContent = label; });
  });
})();
</script>`

// The reset page carries the token handed to it in the query string; nothing
// else about the account is rendered before the token is consumed.
const authResetBody = `<h1>{{.T.titleReset}}</h1>
<p class="lead">{{.T.leadReset}}</p>
<p class="msg" id="msg" role="status"></p>
<form id="form" novalidate>
  <label for="password">{{.T.pwLabel}}</label>
  <input id="password" name="password" type="password" autocomplete="new-password" placeholder="{{.T.pwPh}}" required/>
  <label for="password2">{{.T.pw2Label}}</label>
  <input id="password2" name="password2" type="password" autocomplete="new-password" required/>
  <button id="submit" type="submit">{{.T.submitReset}}</button>
</form>
<p class="alt"><a href="/forgot-password">{{.T.requestNew}}</a> · <a href="/">{{.T.home}}</a></p>
<script>
(function () {
  var token = {{.Token}};
  var form = document.getElementById('form');
  var msg = document.getElementById('msg');
  var submit = document.getElementById('submit');
  var label = submit.textContent;
  function show(text, ok) {
    msg.textContent = text;
    msg.className = 'msg show ' + (ok ? 'ok' : 'err');
  }
  if (!token) { show('{{.T.invalid}}', false); submit.disabled = true; return; }
  form.addEventListener('submit', function (e) {
    e.preventDefault();
    var password = document.getElementById('password').value;
    var confirm = document.getElementById('password2').value;
    if (password.length < 8) { show('{{.T.short}}', false); return; }
    if (password !== confirm) { show('{{.T.mismatch}}', false); return; }
    submit.disabled = true;
    submit.textContent = '{{.T.saving}}';
    fetch('/v1/auth/reset-password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: token, password: password })
    }).then(function (r) {
      if (r.status === 200) { show('{{.T.updated}}', true); form.reset(); return; }
      if (r.status === 400) { show('{{.T.invalid}}', false); return; }
      throw new Error('status ' + r.status);
    }).catch(function () { show('{{.T.genericError}}', false); })
      .finally(function () { submit.disabled = false; submit.textContent = label; });
  });
})();
</script>`

// authTemplates holds the two recovery pages keyed by their page name, built
// the same way the site pages are (shared head + body + shared foot).
var authTemplates = parseAuthTemplates()

func parseAuthTemplates() *template.Template {
	t := template.New("auth")
	for name, body := range map[string]string{
		"forgot-password": authHead + authForgotBody + authFoot,
		"reset-password":  authHead + authResetBody + authFoot,
	} {
		template.Must(t.New(name).Parse(body))
	}
	return t
}

// registerAuthPages mounts the public recovery pages. They are registered next
// to the site pages so the surface gate in internal/transport/http sees the
// same host rules for both.
func registerAuthPages(engine *gin.Engine) {
	render := func(name, titleKey string) gin.HandlerFunc {
		return func(c *gin.Context) {
			lang := pickSiteLang(c)
			dir := "ltr"
			if lang == "ar" {
				dir = "rtl"
			}
			c.Header("Vary", "Accept-Language")
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.Header("X-Robots-Tag", "noindex, nofollow")
			data := authPageData{
				Lang: lang, Dir: dir, Title: authPagesText[lang][titleKey],
				T: authPagesText[lang], Token: c.Query("token"),
				Year: time.Now().Year(),
			}
			if err := authTemplates.ExecuteTemplate(c.Writer, name, data); err != nil {
				c.String(http.StatusInternalServerError, "render error")
			}
		}
	}
	engine.GET("/forgot-password", render("forgot-password", "titleForgot"))
	engine.GET("/reset-password", render("reset-password", "titleReset"))
}
