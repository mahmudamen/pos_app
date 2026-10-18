// Copies the Go content store JSON files into web/src/content/data so the
// React bundle (client + SSR) can import the exact same bilingual source the
// Go site used. Keep this directory single-sourced from
// ../internal/content/data. Run before every build.
import { mkdirSync, readFileSync, writeFileSync, readdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const root = join(here, "..");
const src = join(root, "..", "internal", "content", "data");
const out = join(root, "src", "content", "data");

mkdirSync(join(out, "pages"), { recursive: true });
const pages = readdirSync(join(src, "pages")).filter((f) => f.endsWith(".json"));
for (const f of pages) {
  writeFileSync(join(out, "pages", f), readFileSync(join(src, "pages", f)));
}
writeFileSync(join(out, "strings.json"), readFileSync(join(src, "strings.json")));
console.log(`gen-content: copied ${pages.length} pages + strings.json  ->  src/content/data/`);