// B7: a production build must know where the API is. Without VITE_API_URL it used to fall
// back to http://localhost:8080 silently and ship a site that cannot log in.
//
// Run with: npm test   (node's built-in test runner, no extra dependency)
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, readdirSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const frontendDir = join(dirname(fileURLToPath(import.meta.url)), "..");

function build(extraEnv) {
  const outDir = mkdtempSync(join(tmpdir(), "bakery-build-"));
  const env = { ...process.env, ...extraEnv };
  if (!("VITE_API_URL" in extraEnv)) delete env.VITE_API_URL;
  const result = spawnSync(
    "npx",
    ["vite", "build", "--mode", "production", "--outDir", outDir, "--emptyOutDir"],
    { cwd: frontendDir, env, encoding: "utf8", shell: process.platform === "win32" },
  );
  return { result, outDir, output: `${result.stdout}\n${result.stderr}` };
}

function bundleText(outDir) {
  const assets = join(outDir, "assets");
  return readdirSync(assets)
    .filter((f) => f.endsWith(".js"))
    .map((f) => readFileSync(join(assets, f), "utf8"))
    .join("\n");
}

test("B7: the production build fails, naming the variable, when VITE_API_URL is missing", () => {
  const { result, outDir, output } = build({});
  try {
    assert.notEqual(result.status, 0, "the build should fail without VITE_API_URL");
    assert.match(output, /VITE_API_URL/);
  } finally {
    rmSync(outDir, { recursive: true, force: true });
  }
});

test("B7: with VITE_API_URL the build works and the bundle points to that API, not to localhost", () => {
  const { result, outDir, output } = build({ VITE_API_URL: "https://api.example.test" });
  try {
    assert.equal(result.status, 0, `the build should pass:\n${output}`);
    const js = bundleText(outDir);
    assert.ok(js.includes("https://api.example.test"), "the bundle should contain the API url");
    assert.ok(!js.includes("localhost:8080"), "the bundle must not fall back to localhost");
  } finally {
    rmSync(outDir, { recursive: true, force: true });
  }
});

test("B7: a malformed VITE_API_URL is rejected", () => {
  const { result, outDir, output } = build({ VITE_API_URL: "not a url" });
  try {
    assert.notEqual(result.status, 0, "the build should fail for an invalid url");
    assert.match(output, /VITE_API_URL/);
  } finally {
    rmSync(outDir, { recursive: true, force: true });
  }
});
