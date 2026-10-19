// The tab icon is the brand monogram, not the Vite template's bolt.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const read = (p) => readFileSync(join(root, p), "utf8");

test("index.html points to the favicon", () => {
  assert.match(read("index.html"), /<link rel="icon" type="image\/svg\+xml" href="\/favicon\.svg"/);
});

test("the favicon is the brand's, not the template's", () => {
  const svg = read("public/favicon.svg");
  assert.doesNotMatch(svg, /863bff/i, "it is still the purple bolt of the Vite template");
  assert.match(svg, /#1C1C1C/i, "it should use the brand black");
  assert.match(svg, /#FAF6F1/i, "it should sit on the brand cream, so it shows in dark tabs too");
  assert.match(svg, /viewBox="[^"]+"/);
});
