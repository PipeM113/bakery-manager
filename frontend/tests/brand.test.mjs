// Sprint 4, rules DS1 to DS7: the manager follows the brand manual of the web
// (angeles_bakery_web/docs/diseno.md). Static checks over the sources; DS8 (screenshots) is manual.
//
// Run with: npm test
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { join, dirname, relative } from "node:path";
import { fileURLToPath } from "node:url";

const frontendDir = join(dirname(fileURLToPath(import.meta.url)), "..");
const repoDir = join(frontendDir, "..");
const read = (rel) => readFileSync(join(frontendDir, rel), "utf8");

function sources(dir) {
  return readdirSync(dir).flatMap((name) => {
    const full = join(dir, name);
    if (statSync(full).isDirectory()) return name === "assets" ? [] : sources(full);
    return /\.(ts|tsx|css)$/.test(name) ? [full] : [];
  });
}
const files = [
  ...sources(join(frontendDir, "src")),
  join(frontendDir, "tailwind.config.js"),
  join(frontendDir, "index.html"),
].map((f) => ({ path: relative(frontendDir, f), text: readFileSync(f, "utf8") }));

const { default: tailwind } = await import(join(frontendDir, "tailwind.config.js"));
const colors = tailwind.theme?.extend?.colors ?? tailwind.theme?.colors ?? {};
const oro = typeof colors.oro === "string" ? colors.oro : colors.oro?.DEFAULT;
const oroClaro = colors["oro-claro"] ?? colors.oro?.claro;
const brand = {
  crema: colors.crema, lino: colors.lino, salvia: colors.salvia, rosa: colors.rosa,
  negro: colors.negro, oro, "oro-claro": oroClaro, peligro: colors.peligro,
};

// ---- WCAG contrast ----
const rgb = (hex) => [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16));
const lum = ([r, g, b]) => {
  const f = (c) => { c /= 255; return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4; };
  return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
};
const contrast = (a, b) => {
  const [hi, lo] = [lum(a), lum(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
};
const over = (fg, bg, alpha) => fg.map((c, i) => c * alpha + bg[i] * (1 - alpha));

test("DS1: the theme defines the seven colors of the brand manual, exactly", () => {
  assert.deepEqual(
    Object.fromEntries(Object.entries(brand).filter(([k]) => k !== "peligro").map(([k, v]) => [k, v?.toUpperCase()])),
    { crema: "#FAF6F1", lino: "#E8DDD3", salvia: "#C4C9B8", rosa: "#D9B8AE", negro: "#1C1C1C", oro: "#735523", "oro-claro": "#EECC6A" },
  );
  assert.equal(brand.peligro?.toLowerCase(), "#8a1c1c", "the error red of the web admin panel");
});

test("DS1: the old palette is gone from the theme and from the sources", () => {
  const config = read("tailwind.config.js");
  assert.doesNotMatch(config, /\b(vanilla|terracota|olive|gold|stone)\b\s*:/, "tailwind.config.js still defines the old colors");
  const legacyHex = /#(B8860B|D4A017|8B6508|FDFAF4|F7F0E4|EDE0C8|C4714A|B05A35|8E4528|5C6B3A|47542C|5C5248|4A3D35|1E1410)\b/i;
  const raw = /\b(?:bg|text|border|ring|from|to|via|divide|placeholder|accent|fill|stroke|outline)-(?:stone|gold|vanilla|terracota|olive|red|green|emerald|amber|blue|yellow|orange|gray|slate|zinc|neutral)\b/;
  for (const f of files) {
    assert.doesNotMatch(f.text, legacyHex, `${f.path} uses a color of the old palette`);
    assert.doesNotMatch(f.text, raw, `${f.path} uses a utility color that is not in the brand palette`);
    assert.doesNotMatch(f.text, /\btext-white\b/, `${f.path}: text on dark surfaces is crema, not white`);
  }
});

test("DS2: Playfair Display for titles and DM Sans for the text, shipped with the app", () => {
  const css = read("src/index.css");
  assert.match(css, /@fontsource\/playfair-display/);
  assert.match(css, /@fontsource\/dm-sans/);
  for (const f of files) {
    assert.doesNotMatch(f.text, /fonts\.googleapis\.com|Cormorant|\bJost\b/, `${f.path} still refers to the old fonts`);
  }
  const fonts = tailwind.theme?.extend?.fontFamily ?? {};
  assert.match(String(fonts.display), /Playfair Display/);
  assert.match(String(fonts.sans), /DM Sans/);
  const pkg = JSON.parse(read("package.json"));
  assert.ok(pkg.dependencies["@fontsource/playfair-display"], "package.json lacks @fontsource/playfair-display");
  assert.ok(pkg.dependencies["@fontsource/dm-sans"], "package.json lacks @fontsource/dm-sans");
});

test("DS3: every allowed text and background pair has a contrast of at least 4.5", () => {
  const pairs = [
    ["negro", "crema"], ["negro", "lino"], ["negro", "salvia"], ["negro", "rosa"], ["negro", "#FFFFFF"],
    ["oro", "crema"], ["oro", "#FFFFFF"], ["oro-claro", "negro"], ["crema", "negro"],
    ["peligro", "#FFFFFF"], ["peligro", "crema"],
  ];
  for (const [fg, bg] of pairs) {
    const a = fg.startsWith("#") ? fg : brand[fg];
    const b = bg.startsWith("#") ? bg : brand[bg];
    assert.ok(a && b, `the theme has no color ${a ? bg : fg}`);
    const ratio = contrast(rgb(a), rgb(b));
    assert.ok(ratio >= 4.5, `${fg} on ${bg} has a contrast of ${ratio.toFixed(2)}, below 4.5`);
  }
  // Secondary text is the black at 70%: it must stay readable on the three light grounds.
  for (const bg of ["crema", "lino", "#FFFFFF"]) {
    const hex = bg.startsWith("#") ? bg : brand[bg];
    assert.ok(hex, `the theme has no color ${bg}`);
    const ground = rgb(hex);
    assert.ok(brand.negro, "the theme has no color negro");
    const ratio = contrast(over(rgb(brand.negro), ground, 0.7), ground);
    assert.ok(ratio >= 4.5, `negro at 70% on ${bg} has a contrast of ${ratio.toFixed(2)}, below 4.5`);
  }
});

test("DS3: salvia, rosa and the light gold are never a text color on a light ground", () => {
  for (const f of files) {
    assert.doesNotMatch(f.text, /\btext-(salvia|rosa)\b/, `${f.path} uses salvia or rosa as text`);
    for (const line of f.text.split("\n")) {
      if (/\btext-oro-claro\b/.test(line)) {
        assert.match(line, /\bbg-negro\b/, `${f.path}: light gold text needs a black ground on the same element: ${line.trim()}`);
      }
    }
  }
});

test("DS4: buttons are black with cream letters; no solid gold buttons", () => {
  for (const f of files) {
    assert.doesNotMatch(f.text, /\bbg-oro\b(?!\/)/, `${f.path} has a solid gold fill`);
    assert.doesNotMatch(f.text, /\bbg-oro-claro\b/, `${f.path} has a light gold fill`);
  }
  const buttons = files.filter((f) => f.path.endsWith(".tsx")).flatMap((f) => f.text.match(/bg-negro[^"'`]*text-crema|text-crema[^"'`]*bg-negro/g) ?? []);
  assert.ok(buttons.length >= 10, `expected the primary buttons to be bg-negro text-crema; found ${buttons.length}`);
});

test("DS5: states use the palette: salvia for success, rosa for warnings, peligro for errors", () => {
  const all = files.map((f) => f.text).join("\n");
  assert.match(all, /\bbg-salvia\b/, "no success state on salvia");
  assert.match(all, /\btext-peligro\b/, "no error state in peligro");
});

test("DS6: the brand is visible: logo, tab title, language and focus", () => {
  const html = read("index.html");
  assert.match(html, /<html lang="es"/);
  assert.match(html, /<title>Angeles Bakery/);
  assert.doesNotMatch(html, /<title>frontend/);
  assert.ok(existsSync(join(frontendDir, "src/assets/brand/logo-color.svg")), "src/assets/brand/logo-color.svg is missing");
  for (const page of ["src/components/layout/Sidebar.tsx", "src/pages/LoginPage.tsx"]) {
    assert.match(read(page), /assets\/brand\//, `${page} does not show the brand logo`);
  }
  const css = read("src/index.css");
  assert.match(css, /:focus-visible\s*\{[^}]*outline[^}]*(oro|#735523)/s, "the focus outline should be the dark gold");
});

test("DS7: docs/diseno.md records the rules and points to the web manual", () => {
  const doc = join(repoDir, "docs", "diseno.md");
  assert.ok(existsSync(doc), "docs/diseno.md is missing");
  const text = readFileSync(doc, "utf8");
  for (const want of ["angeles_bakery_web", "#FAF6F1", "#735523", "Playfair Display", "DM Sans", "contraste", "salvia"]) {
    assert.ok(text.includes(want), `docs/diseno.md should mention ${want}`);
  }
});
