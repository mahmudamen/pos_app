// Bilingual content store, mirroring the Go `internal/content` package.
// Pages are loaded eagerly from the generated JSON copies so both the client
// bundle and the SSR build see identical data.
import stringsData from "./content/data/strings.json";

const pageModules = import.meta.glob("./content/data/pages/*.json", {
  eager: true,
}) as Record<string, { default: PageData }>;

export type Raw = Record<string, unknown>;

export interface LangPage {
  meta: Raw;
  [key: string]: unknown;
}

export interface PageData {
  en: LangPage;
  ar: LangPage;
}

export interface Route {
  path: string;
  slug: string;
  index: boolean;
  sitemap?: boolean;
  kind: "sections" | "home" | "form";
}

export type Lang = "ar" | "en";
export const LANGS: Lang[] = ["ar", "en"];

// Route table mirrors internal/web/server.go. Home, the three form pages and
// everything else render from `sections[]` (see the Page renderer).
export const routes: Route[] = [
  { path: "/", slug: "home", index: true, kind: "home" },
  { path: "/solutions/erp", slug: "solutions-erp", index: true, kind: "sections" },
  { path: "/solutions/it-infrastructure", slug: "solutions-it-infrastructure", index: true, kind: "sections" },
  { path: "/solutions/web", slug: "solutions-web", index: true, kind: "sections" },
  { path: "/odoo-pos-egypt", slug: "odoo-pos-egypt", index: true, kind: "sections" },
  { path: "/erp-agriculture-egypt", slug: "erp-agriculture-egypt", index: true, kind: "sections" },
  { path: "/addons-store", slug: "addons-store", index: true, kind: "sections" },
  { path: "/ai-solutions", slug: "ai-solutions", index: true, kind: "sections" },
  { path: "/pricing", slug: "pricing", index: true, kind: "sections" },
  { path: "/portfolio", slug: "portfolio", index: true, kind: "sections" },
  { path: "/testimonials", slug: "testimonials", index: true, kind: "sections" },
  { path: "/knowledge", slug: "knowledge", index: true, kind: "sections" },
  { path: "/insights", slug: "insights", index: true, kind: "sections" },
  { path: "/insights/ai-marketing-copilot", slug: "insights-ai-marketing-copilot", index: true, kind: "sections" },
  { path: "/insights/ai-pos-copilot", slug: "insights-ai-pos-copilot", index: true, kind: "sections" },
  { path: "/networktool", slug: "networktool", index: true, kind: "sections" },
  { path: "/apk-download", slug: "apk-download", index: true, kind: "form" },
  { path: "/request-demo", slug: "request-demo", index: true, kind: "form" },
  { path: "/contact-us", slug: "contact-us", index: true, kind: "form" },
  { path: "/theme-info", slug: "theme-info", index: true, kind: "sections" },
  { path: "/privacy-policy", slug: "privacy-policy", index: true, kind: "sections" },
  { path: "/terms", slug: "terms", index: true, kind: "sections" },
];

export const aliases: Record<string, string> = {
  "/contactus": "/contact-us",
  "/home": "/",
  "/blogs": "/insights",
};

const bySlug = new Map<string, PageData>();
const pages: Record<string, PageData> = {};

for (const key of Object.keys(pageModules)) {
  const base = key.split("/").pop()?.replace(/\.json$/, "") ?? "";
  const data = pageModules[key].default;
  if (!bySlug.has(base)) bySlug.set(base, data);
  const rt = routes.find((r) => r.slug === base);
  if (rt) pages[rt.path] = data;
}

export function pageFor(path: string): PageData | undefined {
  return pages[path];
}

export function routeFor(path: string): Route | undefined {
  return routes.find((r) => r.path === path);
}

export function getStrings(lang: Lang) {
  return stringsData[lang];
}

export function getPage(path: string, lang: Lang): LangPage {
  const rt = routeFor(path);
  const data = rt ? bySlug.get(rt.slug) : undefined;
  if (!data) throw new Error(`no content for ${path}`);
  return data[lang];
}

// --- typed accessors over the loosely-typed JSON ----------------------------

export function str(v: unknown): string {
  if (v == null) return "";
  if (typeof v === "string") return v;
  if (typeof v === "number" || typeof v === "boolean") return String(v);
  return JSON.stringify(v);
}

export function arrOf(v: unknown): Raw[] {
  return Array.isArray(v) ? (v as Raw[]) : [];
}

// value helper: nested string field
export function cv(obj: Raw | undefined, key: string): string {
  return str(obj?.[key]);
}

export type Strings = typeof stringsData["ar"];