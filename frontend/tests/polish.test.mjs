// Review of 2026-11-22 (design, accessibility and best practices): small rules that keep the interface consistent.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";

const root = new URL("../src/", import.meta.url).pathname;
const walk = (dir) => readdirSync(dir).flatMap((f) => {
  const p = join(dir, f);
  return statSync(p).isDirectory() ? walk(p) : /\.(tsx?|css)$/.test(f) ? [p] : [];
});
const files = walk(root);
const read = (p) => readFileSync(join(root, p), "utf8");

test("P1 no control animates every property: transitions name what changes", () => {
  for (const f of files) assert.doesNotMatch(readFileSync(f, "utf8"), /transition-all/, f);
});

test("P2 the browser console does not get the money of the business", () => {
  for (const f of files) assert.doesNotMatch(readFileSync(f, "utf8"), /console\.(log|debug)\(/, f);
});

test("P3 headings balance their lines, paragraphs avoid orphans, and figures line up", () => {
  const css = read("index.css");
  assert.match(css, /text-wrap:\s*balance/);
  assert.match(css, /text-wrap:\s*pretty/);
  assert.match(css, /font-variant-numeric:\s*tabular-nums/);
});

test("P4 photos have a faint outline so they do not melt into the cream", () => {
  assert.match(read("index.css"), /img[^{]*\{[^}]*outline:\s*1px solid/);
});

test("P5 there is a link to skip the navigation, and the landmarks are named", () => {
  const layout = read("components/layout/AppLayout.tsx");
  assert.match(layout, /href="#contenido"/);
  assert.match(layout, /Saltar al contenido/);
  assert.match(layout, /id="contenido"/);
  assert.match(read("components/layout/Sidebar.tsx"), /<nav[^>]*aria-label="Navegación principal"/s);
  assert.match(read("components/layout/BottomNav.tsx"), /<nav[^>]*aria-label="Navegación principal"/s);
});
