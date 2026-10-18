import { cv, type Lang, type Raw } from "../content";
import { localizeHref } from "../i18n";

// FormPanel ports templates/partials/form.html — a native POST form that the
// Go backend handles at /api/{contact,demo,apk}, redirecting back with
// ?sent=1 / ?sent=error.
export function FormPanel({
  lang,
  sent,
  action,
  title,
  sub,
  submit,
  privacyLabel,
  products,
  strings,
}: {
  lang: Lang;
  sent: string;
  action: string;
  title: string;
  sub: string;
  submit: string;
  privacyLabel: string;
  products: string[];
  strings: Raw;
}) {
  return (
    <section className="section">
      <div className="container panel">
        <div className="panel-head">
          <h2 className="section-title">{title}</h2>
          {sub ? <p className="section-sub">{sub}</p> : null}
        </div>
        {sent === "1" ? <div className="notice notice-ok">{cv(strings, "sent")}</div> : null}
        {sent === "error" ? <div className="notice notice-err">{cv(strings, "error")}</div> : null}
        <form className="form" method="post" action={action}>
          <input
            className="hp"
            type="text"
            name="website"
            tabIndex={-1}
            autoComplete="off"
            aria-hidden="true"
          />
          <input type="hidden" name="lang" value={lang} />
          <div className="form-grid">
            <div className="field">
              <label className="field-label" htmlFor="f-name">
                {cv(strings, "name")}
              </label>
              <input className="input" id="f-name" name="name" required maxLength={120} autoComplete="name" />
            </div>
            <div className="field">
              <label className="field-label" htmlFor="f-email">
                {cv(strings, "email")}
              </label>
              <input
                className="input"
                id="f-email"
                name="email"
                type="email"
                required
                maxLength={160}
                autoComplete="email"
              />
            </div>
            <div className="field">
              <label className="field-label" htmlFor="f-phone">
                {cv(strings, "phone")}
              </label>
              <input className="input" id="f-phone" name="phone" dir="ltr" maxLength={40} autoComplete="tel" />
            </div>
            <div className="field">
              <label className="field-label" htmlFor="f-company">
                {cv(strings, "company")}
              </label>
              <input className="input" id="f-company" name="company" maxLength={160} autoComplete="organization" />
            </div>
          </div>
          {products.length ? (
            <div className="field">
              <label className="field-label" htmlFor="f-product">
                {cv(strings, "product")}
              </label>
              <select className="input" id="f-product" name="product" defaultValue="">
                <option value="">{cv(strings, "select")}</option>
                {products.map((p) => (
                  <option value={p} key={p}>
                    {p}
                  </option>
                ))}
              </select>
            </div>
          ) : null}
          <div className="field">
            <label className="field-label" htmlFor="f-msg">
              {cv(strings, "message")}
            </label>
            <textarea className="input" id="f-msg" name="message" rows={5} required maxLength={4000} />
          </div>
          <button className="btn btn-primary btn-lg" type="submit">
            {submit}
          </button>
          <p className="form-note">
            {cv(strings, "privacyNote")}{" "}
            <a href={localizeHref(lang, "/privacy-policy")}>{privacyLabel}</a>
          </p>
        </form>
      </div>
    </section>
  );
}