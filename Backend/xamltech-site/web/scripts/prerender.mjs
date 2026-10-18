// Writes the prerendered HTML for every route x language into
// web/dist/pages/<lang>/<path>/index.html, embedding the asset tags Vite
// emitted for the client build. The Go server then embeds web/dist and maps
// route+lang directly onto those files.
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { renderDocument, ALL_PAGES } from "../dist-server/ssr.js";

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = join(__dirname, "..");
const clientDir = join(root, "dist", "client");
const pagesOut = join(root, "dist", "pages");

// --- client asset tags -------------------------------------------------------
const clientHtml = readFileSync(join(clientDir, "index.html"), "utf8");
const assetTags = [...clientHtml.matchAll(/<(?:link|script)\b[^>]*>(?:<\/script>)?/gi)]
  .map((m) => m[0])
  .filter((t) => !t.includes('/src/main')) // drop the dev entry reference if any
  .filter((t) => /\/(?:static\/assets\/[^"']+|assets\/[^"']+)/.test(t));

if (!assetTags.length) {
  console.error("No client asset tags found in dist/client/index.html — did vite build run?");
  process.exit(1);
}

// --- emit pages --------------------------------------------------------------
let count = 0;
for (const { path, lang } of ALL_PAGES) {
  const { html } = renderDocument(path, lang, "", assetTags);
  const rel = lang + "/" + path.replace(/^\//, "") + "/index.html"; // "/" → "ar/index.html"
  const file = join(pagesOut, rel);
  mkdirSync(dirname(file), { recursive: true });
  writeFileSync(file, html);
  count++;
}
console.log(`prerendered ${count} pages (${ALL_PAGES.length} route x lang) -> ${pagesOut}`);
console.log(`asset tags embedded:\n  ${assetTags.join("\n  ")}`);