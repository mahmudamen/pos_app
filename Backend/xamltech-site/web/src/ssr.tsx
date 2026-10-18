import { renderToString } from "react-dom/server";
import { App } from "./app";
import { getPage, routeFor, routes, LANGS, type Lang } from "./content";
import type { Boot } from "./i18n";
import { buildHead, pageMeta, CLARITY_ID } from "./seo";

export interface RenderResult {
  html: string; // `<html ...><head>…</head><body>…</body></html>`
  title: string;
}

// ALL_PAGES lists every (path, lang) the prerender step must emit.
export const ALL_PAGES: Array<{ path: string; lang: Lang }> = routes.flatMap((r) =>
  LANGS.map((lang) => ({ path: r.path, lang })),
);

// renderDocument produces the full prerendered document for a route+language.
// It is called once per page at build time; the client hydrates the identical
// tree from the window.__XT__ boot payload.
export function renderDocument(path: string, lang: Lang, sent: string, assetTags: string[]): RenderResult {
  const boot: Boot = { path, lang, sent: sent || "" };
  const route = routeFor(path);
  const meta = route ? pageMeta(path, lang, getPage(path, lang)) : { title: "404 — XamlTech", description: "", ogImage: "", type: "website", index: false };
  const head = buildHead({ boot, meta, assetTags, clarityId: CLARITY_ID });
  const body = renderToString(<App {...boot} />);
  // The boot script keeps "__SENT__" as a placeholder the Go server replaces
  // with the form-submission outcome ("", "1" or "error") on ?sent= requests.
  const html = `<!doctype html>\n<html lang="${lang === "ar" ? "ar" : "en"}" dir="${lang === "ar" ? "rtl" : "ltr"}">\n${head}\n<body>\n<div id="root">${body}</div>\n<script>window.__XT__={"path":"${path}","lang":"${lang}","sent":"__SENT__"};</script>\n</body>\n</html>`;
  return { html, title: meta.title };
}

export { App };