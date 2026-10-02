// Fails the build when the JavaScript needed for first paint exceeds the
// budget (IMPLEMENTATION_PLAN §7.9). "Initial" = the index.html entry chunk
// plus every chunk it imports statically; lazy route chunks are excluded.
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { gzipSync } from "node:zlib";

const BUDGET_KB = 180;
const dist = new URL("../dist/", import.meta.url).pathname;
const manifest = JSON.parse(readFileSync(join(dist, ".vite/manifest.json"), "utf8"));

const entryKey = Object.keys(manifest).find((k) => manifest[k].isEntry);
if (!entryKey) throw new Error("no entry chunk in Vite manifest");

const seen = new Set();
function collect(key) {
  if (seen.has(key)) return;
  seen.add(key);
  for (const dep of manifest[key].imports ?? []) collect(dep);
}
collect(entryKey);

let total = 0;
const rows = [];
for (const key of seen) {
  const file = manifest[key].file;
  if (!file.endsWith(".js")) continue;
  const gz = gzipSync(readFileSync(join(dist, file)), { level: 9 }).length;
  total += gz;
  rows.push([file, (gz / 1024).toFixed(1)]);
}
const totalKB = total / 1024;
for (const [file, kb] of rows) console.log(`  ${kb.padStart(7)} KB  ${file}`);
console.log(`Initial JS: ${totalKB.toFixed(1)} KB gzip (budget ${BUDGET_KB} KB)`);
if (totalKB > BUDGET_KB) {
  console.error(`Bundle budget exceeded by ${(totalKB - BUDGET_KB).toFixed(1)} KB`);
  process.exit(1);
}
