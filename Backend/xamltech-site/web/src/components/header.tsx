import { useState } from "react";
import { getStrings, type Lang } from "../content";
import { langSwitchHref, otherLang, selfLabel } from "../i18n";
import { navItems, contactOf, type NavItem } from "../site";

function Logo() {
  return (
    <svg viewBox="0 0 32 32" aria-hidden="true" className="brand-logo">
      <rect x="1" y="1" width="30" height="30" rx="8" fill="#16a34a" />
      <path
        d="M9 22.5 16 9l3.4 6.4H23L16 9"
        fill="none"
        stroke="#fff"
        strokeWidth="2.4"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path d="M16 9v13.5" stroke="#fff" strokeWidth="2.4" strokeLinecap="round" />
    </svg>
  );
}

export function Header({ lang, path }: { lang: Lang; path: string }) {
  const strings = getStrings(lang);
  const contact = contactOf(lang);
  const items = navItems(lang);
  const [menuOpen, setMenuOpen] = useState(false);
  const [openSub, setOpenSub] = useState<string | null>(null);
  const homeHref = lang === "en" ? "/?lang=en" : "/";
  const demoHref = lang === "en" ? "/request-demo?lang=en" : "/request-demo";

  return (
    <>
      <header className="site-header" id="top">
        <div className="container nav">
          <a className="brand" href={homeHref} aria-label="XamlTech">
            <Logo />
            <span>
              <b>
                Xaml<span className="brand-accent">Tech</span>
              </b>
              <small>{strings.tagline}</small>
            </span>
          </a>

          <nav
            className={`nav-links${menuOpen ? " open" : ""}`}
            id="site-nav"
            aria-label="Primary"
          >
            <ul>
              {items.map((it) => (
                <NavEntry
                  key={it.label}
                  item={it}
                  open={openSub === it.label}
                  onToggle={() => setOpenSub((s) => (s === it.label ? null : it.label))}
                />
              ))}
            </ul>
            <div className="nav-foot">
              <a className="btn btn-primary btn-block" href={demoHref} onClick={() => setMenuOpen(false)}>
                {strings.ctaDemo}
              </a>
              <a
                className="btn btn-wa btn-block"
                href={`https://wa.me/${String(contact.whatsapp)}`}
                target="_blank"
                rel="noopener"
                onClick={() => setMenuOpen(false)}
              >
                {strings.ctaWhatsapp}
              </a>
            </div>
          </nav>

          <div className="hdr-actions">
            <a className="hdr-phone" href="tel:+201201410140">
              <span dir="ltr">+20 120 141 0140</span>
            </a>
            <a className="btn btn-primary btn-sm hdr-cta" href={demoHref}>
              {strings.ctaDemo}
            </a>
            <a
              className="lang-btn"
              href={langSwitchHref(lang, path)}
              aria-label={`Language: ${otherLang(lang) === "en" ? "English" : "العربية"}`}
            >
              {selfLabel(otherLang(lang))}
            </a>
            <button
              className="icon-btn burger"
              aria-label={strings.menu}
              aria-controls="site-nav"
              aria-expanded={menuOpen}
              onClick={() => setMenuOpen((v) => !v)}
            >
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M4 7h16M4 12h16M4 17h16" />
              </svg>
            </button>
          </div>
        </div>
      </header>
      <div className="nav-scrim" hidden={!menuOpen} onClick={() => setMenuOpen(false)} />
    </>
  );
}

function NavEntry({
  item,
  open,
  onToggle,
}: {
  item: NavItem;
  open: boolean;
  onToggle: () => void;
}) {
  if (item.children.length) {
    return (
      <li className={`nav-item has-menu${open ? " open" : ""}`}>
        <button className="nav-link" aria-expanded={open} aria-haspopup="true" onClick={onToggle}>
          {item.label}
          <svg className="i-caret" viewBox="0 0 24 24" aria-hidden="true">
            <path d="M6 9l6 6 6-6" />
          </svg>
        </button>
        <ul className="nav-menu">
          {item.children.map((c) => (
            <li key={c.label}>
              <a href={c.href} {...(c.external ? { target: "_blank", rel: "noopener" } : {})}>
                {c.label}
              </a>
            </li>
          ))}
        </ul>
      </li>
    );
  }
  return (
    <li className="nav-item">
      <a className="nav-link" href={item.href} {...(item.external ? { target: "_blank", rel: "noopener" } : {})}>
        {item.label}
      </a>
    </li>
  );
}