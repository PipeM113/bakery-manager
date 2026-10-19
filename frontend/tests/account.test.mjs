// Sprint 13 (C1, C2, front side): "Mi cuenta" changes the password; the login says when it is locked.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const src = (p) => readFileSync(join(root, "src", p), "utf8");

test("the service changes the password through PUT /auth/password", () => {
  const path = join(root, "src/api/authService.ts");
  assert.ok(existsSync(path), "api/authService.ts does not exist");
  const s = readFileSync(path, "utf8");
  assert.match(s, /changePassword/);
  assert.match(s, /api\.put[^(]*\(\s*["']\/auth\/password["']/);
  assert.match(s, /current_password/);
  assert.match(s, /new_password/);
});

test("the screen has the three fields and the button", () => {
  const path = join(root, "src/pages/AccountPage.tsx");
  assert.ok(existsSync(path), "pages/AccountPage.tsx does not exist");
  const s = readFileSync(path, "utf8");
  assert.match(s, /data-testid="account-form"/);
  for (const label of ["Contraseña actual", "Nueva contraseña", "Repite la nueva contraseña"]) {
    assert.ok(s.includes(label), `no field "${label}"`);
  }
  assert.match(s, /Cambiar contraseña/);
  assert.match(s, /type="password"/);
  assert.match(s, /autoComplete="current-password"/);
  assert.match(s, /autoComplete="new-password"/);
  assert.match(s, /10 caracteres/, "it should say the minimum length");
});

test("the route and both navigations lead to Mi cuenta", () => {
  assert.match(src("App.tsx"), /path="\/account"/);
  for (const nav of ["components/layout/Sidebar.tsx", "components/layout/BottomNav.tsx"]) {
    assert.match(src(nav), /\/account/, nav);
    assert.match(src(nav), /Mi cuenta/, nav);
  }
});

test("the login explains a lock (429) and when to retry", () => {
  const s = src("pages/LoginPage.tsx");
  assert.match(s, /429/);
  assert.match(s, /Retry-After|retry-after/i);
  assert.match(s, /intentos/);
});
