import { getStrings, arrOf, cv, type Lang, type Raw } from "./content";
import { localizeHref } from "./i18n";

export interface NavItem {
  label: string;
  href: string;
  external: boolean;
  children: NavItem[];
}

export interface FooterLink {
  label: string;
  href: string;
  external: boolean;
}

export interface FooterCol {
  title: string;
  links: FooterLink[];
}

// navItems resolves the strings.json "nav" block with localised links.
export function navItems(lang: Lang): NavItem[] {
  const items = arrOf(getStrings(lang).nav);
  return items.map((m) => {
    const href = cv(m, "href");
    const ni: NavItem = {
      label: cv(m, "label"),
      href: localizeHref(lang, href),
      external: href.startsWith("http"),
      children: [],
    };
    ni.children = arrOf(m.children).map((k) => {
      const khref = cv(k, "href");
      return {
        label: cv(k, "label"),
        href: localizeHref(lang, khref),
        external: khref.startsWith("http"),
        children: [],
      };
    });
    return ni;
  });
}

export function footerCols(lang: Lang): FooterCol[] {
  const cols = arrOf(getStrings(lang).footer);
  return cols.map((c) => ({
    title: cv(c, "title"),
    links: arrOf(c.links).map((l) => {
      const href = cv(l, "href");
      return { label: cv(l, "label"), href: localizeHref(lang, href), external: href.startsWith("http") };
    }),
  }));
}

export function contactOf(lang: Lang): Raw {
  const c = getStrings(lang).contact;
  return (c && typeof c === "object" ? c : {}) as Raw;
}