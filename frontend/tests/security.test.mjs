// Security audit of 2026-11, points 18 and 20: headers of the frontend, Dependabot and the audit in CI.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const repo = join(root, "..");
const headersOf = () => {
  const cfg = JSON.parse(readFileSync(join(root, "vercel.json"), "utf8"));
  const block = (cfg.headers ?? []).find((h) => h.source === "/(.*)");
  assert.ok(block, 'vercel.json has no "headers" for source "/(.*)"');
  return Object.fromEntries(block.headers.map((h) => [h.key.toLowerCase(), h.value]));
};

test("18: the frontend sends the security headers", () => {
  const h = headersOf();
  assert.equal(h["x-content-type-options"], "nosniff");
  assert.match(h["referrer-policy"], /^(no-referrer|strict-origin-when-cross-origin|same-origin)$/);
  assert.equal(h["x-frame-options"], "DENY");
  assert.match(h["strict-transport-security"], /max-age=\d{7,}/);
  assert.ok(h["permissions-policy"], "no Permissions-Policy");
});

test("18: the CSP is closed: own scripts only, the backend and the photo bucket named, nothing else", () => {
  const csp = headersOf()["content-security-policy"];
  assert.ok(csp, "no Content-Security-Policy");
  const dir = Object.fromEntries(csp.split(";").map((d) => d.trim()).filter(Boolean).map((d) => { const [k, ...v] = d.split(/\s+/); return [k, v]; }));
  assert.deepEqual(dir["default-src"], ["'self'"]);
  assert.deepEqual(dir["script-src"], ["'self'"], "scripts: no inline, no eval, no other host");
  assert.deepEqual(dir["object-src"], ["'none'"]);
  assert.deepEqual(dir["frame-ancestors"], ["'none'"]);
  assert.deepEqual(dir["base-uri"], ["'self'"]);
  assert.deepEqual(dir["form-action"], ["'self'"]);
  assert.ok(dir["connect-src"].includes("https://bakery-manager-ofbi.vercel.app"), "the backend must be allowed in connect-src");
  assert.ok(dir["img-src"].includes("https://hrwbnkikfllpzwuqklrc.supabase.co"), "the photo bucket must be allowed in img-src");
  for (const [name, values] of Object.entries(dir)) {
    assert.ok(!values.includes("*") && !values.some((v) => v.startsWith("*.")), `${name} allows a wildcard`);
    if (name !== "style-src") assert.ok(!values.includes("'unsafe-inline'"), `${name} allows unsafe-inline`);
    assert.ok(!values.includes("'unsafe-eval'"), `${name} allows unsafe-eval`);
  }
});

test("20: Dependabot watches npm, Go modules and the GitHub Actions", () => {
  const path = join(repo, ".github", "dependabot.yml");
  assert.ok(existsSync(path), ".github/dependabot.yml does not exist");
  const s = readFileSync(path, "utf8");
  for (const eco of ["npm", "gomod", "github-actions"]) assert.match(s, new RegExp(`package-ecosystem:\\s*"?${eco}"?`), `no ${eco} entry`);
  assert.match(s, /directory:\s*"?\/frontend"?/);
  assert.match(s, /directory:\s*"?\/backend"?/);
});

test("20: the CI fails on a high vulnerability in production dependencies", () => {
  const ci = readFileSync(join(repo, ".github", "workflows", "ci.yml"), "utf8");
  assert.match(ci, /npm audit --omit=dev --audit-level=high/);
});
