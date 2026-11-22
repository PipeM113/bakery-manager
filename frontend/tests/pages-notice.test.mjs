// The API gives the 200 most recent quotations and sales; when a list is that long the screen says so,
// instead of showing a list that looks complete.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { PAGE_SIZE, mayHaveMore } from "../src/utils/pages.ts";

const src = (p) => readFileSync(new URL(`../src/${p}`, import.meta.url), "utf8");

test("PN1 a list as long as a page may have more; a shorter one does not", () => {
  assert.equal(PAGE_SIZE, 200);
  assert.equal(mayHaveMore(new Array(199)), false);
  assert.equal(mayHaveMore(new Array(200)), true);
  assert.equal(mayHaveMore([]), false);
});

test("PN2 the sales screen tells the person when a list is cut at the 200 most recent", () => {
  const s = src("pages/SalesPage.tsx");
  assert.match(s, /mayHaveMore\(quotations\)/);
  assert.match(s, /mayHaveMore\(sales\)/);
  assert.match(s, /data-testid="more-notice"/);
  assert.match(s, /\{PAGE_SIZE\} más recientes/);
});
