import type { Lang } from "./content";

export interface Boot {
  path: string;
  lang: Lang;
  sent?: string;
}

export function bootFromWindow(): Boot {
  const w = (typeof window !== "undefined" ? window : undefined) as
    | (Window & { __XT__?: Boot })
    | undefined;
  return w?.__XT__ ?? { path: "/", lang: "ar", sent: "" };
}

// dirFor maps a language to the document text direction.
export function dirFor(lang: Lang): "rtl" | "ltr" {
  return lang === "ar" ? "rtl" : "ltr";
}

// locale maps a language to the BCP-47 / og:locale value.
export function localeFor(lang: Lang): string {
  return lang === "ar" ? "ar" : "en";
}

export function ogLocaleFor(lang: Lang): string {
  return lang === "ar" ? "ar_EG" : "en_US";
}

export function selfLabel(lang: Lang): string {
  return lang === "ar" ? "العربية" : "English";
}

export function otherLang(lang: Lang): Lang {
  return lang === "ar" ? "en" : "ar";
}

// localizeHref appends ?lang=en to internal links when serving English —
// identical semantics to the Go `localizeHref`.
export function localizeHref(lang: Lang, href: string): string {
  if (
    href === "" ||
    href.startsWith("http") ||
    href.startsWith("mailto:") ||
    href.startsWith("tel:") ||
    href.startsWith("#")
  ) {
    return href;
  }
  if (lang === "en") {
    const sep = href.includes("?") ? "&" : "?";
    return href + sep + "lang=en";
  }
  return href;
}

// hrefFor builds the language-switch target for the current path.
export function langSwitchHref(lang: Lang, path: string): string {
  if (lang === "ar") return `${path}?lang=en`;
  return `${path}`;
}