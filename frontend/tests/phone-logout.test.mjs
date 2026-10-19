// Bug found by the owner on his phone: the sidebar (desktop) has "Cerrar sesión" but the bottom bar
// ("Más" menu) had none, so nobody could switch accounts from a phone.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const src = join(dirname(fileURLToPath(import.meta.url)), "..", "src");
const bottomNav = readFileSync(join(src, "components/layout/BottomNav.tsx"), "utf8");

test("the phone menu has a logout button wired to the auth context", () => {
  assert.match(bottomNav, /useAuth/);
  assert.match(bottomNav, /Cerrar sesión/);
  assert.match(bottomNav, /data-testid="nav-logout"/);
  assert.match(bottomNav, /onClick=\{[^}]*logout/);
});
