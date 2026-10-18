import { arrOf, cv, getStrings, str, type Lang, type LangPage, type Raw, type Route } from "../content";
import { Section } from "./sections";
import { FormPanel } from "./form";

// HomePage renders home.json's named keys in the same order as the Go
// templates/pages/home.html.
const HOME_ORDER: Array<[string, string]> = [
  ["hero", "hero"],
  ["chips", "chips"],
  ["why", "features"],
  ["services", "features"],
  ["odoo", "features"],
  ["method", "steps"],
  ["demo", "features"],
  ["addons", "features"],
  ["cta", "cta"],
];

const FORM_ACTIONS: Record<string, string> = {
  "/contact-us": "/api/contact",
  "/request-demo": "/api/demo",
  "/apk-download": "/api/apk",
};

export function Page({
  route,
  page,
  lang,
  sent,
}: {
  route: Route;
  page: LangPage;
  lang: Lang;
  sent: string;
}) {
  if (route.kind === "home") {
    return (
      <>
        {HOME_ORDER.map(([key, partial]) => {
          const data = page[key] as Raw | undefined;
          if (!data) return null;
          return <Section key={key} partial={partial} data={data} lang={lang} />;
        })}
      </>
    );
  }
  if (route.kind === "form") {
    const strings = getStrings(lang);
    return (
      <>
        <Section partial="hero" data={page.hero as Raw} lang={lang} />
        <FormPanel
          lang={lang}
          sent={sent}
          action={FORM_ACTIONS[route.path] ?? ""}
          title={cv(page as Raw, "formTitle")}
          sub={cv(page as Raw, "formSub")}
          submit={cv(page as Raw, "submit")}
          privacyLabel={cv(page as Raw, "privacyLabel")}
          products={arrOf(page.products).map(str)}
          strings={strings.form as Raw}
        />
        <Section partial="cta" data={page.cta as Raw} lang={lang} />
      </>
    );
  }
  return (
    <>
      {arrOf(page.sections).map((s, i) => (
        <Section key={i} partial={cv(s, "partial")} data={(s.data ?? {}) as Raw} lang={lang} />
      ))}
    </>
  );
}