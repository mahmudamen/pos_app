// SEO head builder — mirrors internal/web `pageData` metadata emission. Used
// by the SSR renderer to produce the fully-prerendered <head> for each route
// × language. Canonical URLs always point at the production origin so
// preview deployments render identical SEO.
import { ogLocaleFor, dirFor, localeFor, type Boot } from "./i18n";
import { routeFor, cv, type Lang } from "./content";

export const BASE_URL = "https://xamltech.com";
export const CLARITY_ID = "y1l8d34cyq";

export interface HeadMeta {
  title: string;
  description: string;
  ogImage: string;
  type: string;
  index: boolean;
}

export interface HeadInput {
  boot: Boot;
  meta: HeadMeta;
  assetTags: string[];
  clarityId?: string;
}

function canonicalOf(path: string, lang: Lang): string {
  return lang === "en" ? `${BASE_URL}${path}?lang=en` : `${BASE_URL}${path}`;
}

function escapeAttr(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;");
}

function jsonLd(path: string, _lang: Lang, title: string): string {
  const base = BASE_URL;
  const graph: unknown[] = [
    {
      "@type": "Organization",
      "@id": `${base}/#org`,
      name: "XamlTech",
      url: base,
      logo: `${base}/static/img/logo.png`,
      email: "info@xamltech.com",
      telephone: "+201201410140",
      address: { "@type": "PostalAddress", addressLocality: "Aswan", addressCountry: "EG" },
      sameAs: [
        "https://github.com/XamlTech",
        "https://apps.odoo.com/apps/modules/browse?search=xamltech",
      ],
    },
    {
      "@type": "WebSite",
      "@id": `${base}/#website`,
      url: base,
      name: "XamlTech",
      publisher: { "@id": `${base}/#org` },
      inLanguage: ["ar", "en"],
    },
  ];
  if (path !== "/") {
    graph.push({
      "@type": "BreadcrumbList",
      itemListElement: [
        { "@type": "ListItem", position: 1, name: "XamlTech", item: `${base}/` },
        { "@type": "ListItem", position: 2, name: title, item: `${base}${path}` },
      ],
    });
  }
  return JSON.stringify({ "@context": "https://schema.org", "@graph": graph });
}

export function buildHead({ boot, meta, assetTags, clarityId }: HeadInput): string {
  const { path, lang, sent } = boot;
  void sent;
  const canonical = canonicalOf(path, lang);
  const arURL = `${BASE_URL}${path}`;
  const enURL = `${BASE_URL}${path}?lang=en`;
  const alternates = [
    `  <link rel="alternate" hreflang="ar" href="${arURL}">`,
    `  <link rel="alternate" hreflang="en" href="${enURL}">`,
    `  <link rel="alternate" hreflang="x-default" href="${arURL}">`,
  ].join("\n");
  const robots = meta.index
    ? "index,follow,max-image-preview:large"
    : "noindex,nofollow";
  const og = meta.ogImage || `${BASE_URL}/static/img/pos-light.png`;
  const lines = [
    '<meta charset="utf-8">',
    '<meta name="viewport" content="width=device-width, initial-scale=1">',
    `<title>${escapeAttr(meta.title)}</title>`,
    `<meta name="description" content="${escapeAttr(meta.description)}">`,
    `<link rel="canonical" href="${canonical}">`,
    alternates,
    `<meta name="robots" content="${robots}">`,
    '<meta name="author" content="XamlTech">',
    '<meta name="generator" content="xamltech-site-react">',
    '<meta name="theme-color" content="#ffffff">',
    '<meta name="geo.region" content="EG-ASN">',
    '<meta name="geo.placename" content="Aswan">',
    `<meta property="og:type" content="${escapeAttr(meta.type)}">`,
    '<meta property="og:site_name" content="XamlTech">',
    `<meta property="og:locale" content="${ogLocaleFor(lang)}">`,
    `<meta property="og:title" content="${escapeAttr(meta.title)}">`,
    `<meta property="og:description" content="${escapeAttr(meta.description)}">`,
    `<meta property="og:url" content="${canonical}">`,
    `<meta property="og:image" content="${og}">`,
    '<meta name="twitter:card" content="summary_large_image">',
    `<meta name="twitter:title" content="${escapeAttr(meta.title)}">`,
    `<meta name="twitter:description" content="${escapeAttr(meta.description)}">`,
    '<link rel="icon" href="/static/img/favicon.ico" sizes="any">',
    '<link rel="icon" type="image/png" sizes="32x32" href="/static/img/favicon-32x32.png">',
    '<link rel="icon" type="image/png" sizes="16x16" href="/static/img/favicon-16x16.png">',
    '<link rel="apple-touch-icon" href="/static/img/logo.png">',
    '<link rel="manifest" href="/manifest.webmanifest">',
    `  <script type="application/ld+json">${jsonLd(path, lang, meta.title)}</script>`,
  ];
  // inject static asset tags emitted by Vite (css + module scripts)
  for (const tag of assetTags) lines.push(`  ${tag}`);
  const clarity = clarityId
    ? `\n<script type="text/javascript">(function(c,l,a,r,i,t,y){c[a]=c[a]||function(){(c[a].q=c[a].q||[]).push(arguments)};t=l.createElement(r);t.async=1;t.src="https://www.clarity.ms/tag/"+i;y=l.getElementsByTagName(r)[0];y.parentNode.insertBefore(t,y);})(window,document,"clarity","script","${clarityId}");</script>`
    : "";
  return `<head>\n${lines.join("\n")}${clarity}\n</head>`;
}

// metaFrom mirrors the Go metaFrom(content) extractor.
export function metaFrom(page: Record<string, unknown>): HeadMeta {
  const m = (page.meta ?? {}) as Record<string, unknown>;
  const index = typeof m["index"] === "boolean" ? m["index"] : true;
  const type = typeof m["type"] === "string" && m["type"] !== "" ? m["type"] : "website";
  return {
    title: cv(m, "title"),
    description: cv(m, "description"),
    ogImage: cv(m, "og_image"),
    type,
    index,
  };
}

export function pageMeta(path: string, _lang: Lang, page: Record<string, unknown>): HeadMeta {
  const meta = metaFrom(page);
  const rt = routeFor(path);
  if (rt && !rt.index) meta.index = false;
  return meta;
}

export const _dirFor = dirFor;
export const _localeFor = localeFor;