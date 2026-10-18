import { getPage, getStrings, routeFor, aliases, type Lang } from "./content";
import type { Boot } from "./i18n";
import { Header } from "./components/header";
import { Footer } from "./components/footer";
import { Page } from "./components/page";
import { contactOf } from "./site";

// App renders the full body content for a (path, lang, sent) boot state. The
// SSR pass and the client hydration pass share this tree, so it must stay
// deterministic (no random/time-dependent markup).
export function App({ path, lang, sent }: Boot) {
  const resolved = aliases[path] ?? path;
  const route = routeFor(resolved);
  if (!route) {
    return notFound(lang);
  }
  const page = getPage(resolved, lang);
  const strings = getStrings(lang);
  const whatsapp = String(contactOf(lang).whatsapp ?? "");
  return (
    <>
      <a className="skip" href="#main">
        {strings.skip}
      </a>
      <Header lang={lang} path={resolved} />
      <main id="main">
        <Page route={route} page={page} lang={lang} sent={sent ?? ""} />
      </main>
      <Footer lang={lang} />
      {whatsapp ? (
        <a
          className="wa-float"
          href={`https://wa.me/${whatsapp}`}
          target="_blank"
          rel="noopener"
          aria-label={strings.ctaWhatsapp}
        >
          <svg viewBox="0 0 32 32" aria-hidden="true">
            <path d="M16 3a13 13 0 0 0-11 20l-2 6 6-2A13 13 0 1 0 16 3zm7 18c-.3.9-1.8 1.7-2.5 1.7-.7 0-1.6.3-5.1-1.3-4.3-2-7-6.5-7.2-6.8-.2-.3-1.7-2.3-1.7-4.4s1.1-3.1 1.5-3.5c.4-.4.9-.5 1.2-.5h.9c.3 0 .7-.1 1 .7l1.3 3c.2.4.2.7 0 1l-.6.9c-.2.3-.5.5-.2 1 .3.5 1.3 2.1 2.8 3.4 1.9 1.7 2.7 1.9 3.1 2 .3.1.6.1.8-.1.3-.3 1-1.2 1.3-1.6.3-.4.5-.3.9-.2l3 1.4c.4.2.7.3.8.5.1.2.1 1-.2 1.9z" />
          </svg>
        </a>
      ) : null}
    </>
  );
}

function notFound(lang: Lang) {
  const strings = getStrings(lang);
  return (
    <>
      <a className="skip" href="#main">
        {strings.skip}
      </a>
      <Header lang={lang} path="/" />
      <main id="main">
        <section className="section">
          <div className="container page-head">
            <h1 className="page-title">404</h1>
            <p className="prose">{lang === "ar" ? "الصفحة غير موجودة." : "Page not found."}</p>
          </div>
        </section>
      </main>
      <Footer lang={lang} />
    </>
  );
}

export type { Boot };