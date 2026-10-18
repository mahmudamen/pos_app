import { getStrings, type Lang } from "../content";
import { contactOf, footerCols } from "../site";

export function Footer({ lang, year }: { lang: Lang; year?: number }) {
  const strings = getStrings(lang);
  const contact = contactOf(lang);
  const cols = footerCols(lang);
  const homeHref = lang === "en" ? "/?lang=en" : "/";
  const y = year ?? new Date().getUTCFullYear();
  return (
    <footer className="site-footer">
      <div className="container footer">
        <div className="footer-brand">
          <b>
            Xaml
            <small>
              <span className="brand-accent">Tech</span>
            </small>
          </b>
          <p>{strings.footerAbout}</p>
          <div className="footer-social">
            <a href="https://github.com/XamlTech" target="_blank" rel="noopener" aria-label="GitHub">
              {/* GitHub octocat */}
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M12 2a10 10 0 0 0-3.2 19.5c.5.1.7-.2.7-.5v-1.9c-2.8.6-3.4-1.2-3.4-1.2-.5-1.1-1.1-1.4-1.1-1.4-.9-.6.1-.6.1-.6 1 .1 1.5 1 1.5 1 .9 1.5 2.3 1.1 2.9.8.1-.6.3-1.1.6-1.4-2.2-.2-4.6-1.1-4.6-5 0-1.1.4-2 1-2.7-.1-.2-.5-1.3.1-2.6 0 0 .8-.3 2.8 1a9.5 9.5 0 0 1 5 0c1.9-1.3 2.7-1 2.7-1 .6 1.3.2 2.4.1 2.6.6.7 1 1.6 1 2.7 0 3.9-2.4 4.8-4.6 5 .4.3.7.9.7 1.9v2.8c0 .3.2.6.7.5A10 10 0 0 0 12 2z" />
              </svg>
            </a>
            <a
              href="https://apps.odoo.com/apps/modules/browse?search=xamltech"
              target="_blank"
              rel="noopener"
              aria-label="Odoo Apps"
            >
              {String(contact.appsStore)}
            </a>
          </div>
        </div>

        {cols.map((col) => (
          <div className="footer-links" key={col.title}>
            <h4>{col.title}</h4>
            {col.links.map((l) => (
              <a
                key={l.label}
                href={l.href}
                {...(l.external ? { target: "_blank", rel: "noopener" } : {})}
              >
                {l.label}
              </a>
            ))}
          </div>
        ))}

        <div className="footer-links">
          <h4>{strings.getInTouch}</h4>
          <a href="tel:+201201410140">
            <span dir="ltr">{String(contact.phone || "")}</span>
          </a>
          <a href="tel:+201007982468">
            <span dir="ltr">{String(contact.phone2 || "")}</span>
          </a>
          <a href={`mailto:${String(contact.email)}`}>{String(contact.email)}</a>
          <a href={`https://wa.me/${String(contact.whatsapp)}`} target="_blank" rel="noopener">
            {strings.ctaWhatsapp}
          </a>
          <span className="footer-tiny">{String(contact.address)}</span>
        </div>

        <div className="footer-text">
          <p>
            © {y} XamlTech. {strings.rights}
          </p>
          <p className="footer-tiny">{strings.madeIn}</p>
          <a className="footer-tiny" href={homeHref}>
            {homeHref === "/" ? "xamltech.com" : "xamltech.com/en"}
          </a>
        </div>
      </div>
    </footer>
  );
}