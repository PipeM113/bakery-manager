#!/usr/bin/env bash
# El comando único de calidad: lo mismo que corre el CI, más formato y búsqueda de secretos.
#
#   ./scripts/check.sh          revisión rápida (la que va antes de cada commit)
#   ./scripts/check.sh --full   además las pruebas de navegador (responsive y CSP; tardan varios minutos)
#
# Para las pruebas de base de datos exporta TEST_DATABASE_URL (ver README). Sin ella esas pruebas se saltan
# y el script lo avisa: en el CI su ausencia es un error.
set -uo pipefail
cd "$(dirname "$0")/.."

FULL=0
[ "${1:-}" = "--full" ] && FULL=1
failed=()

step() { # step "name" command...
  local name="$1"; shift
  printf '\n== %s ==\n' "$name"
  if "$@"; then printf 'ok: %s\n' "$name"; else printf 'FALLÓ: %s\n' "$name" >&2; failed+=("$name"); fi
}

gofmt_clean() {
  local out
  out=$(cd backend && gofmt -l .)
  [ -z "$out" ] || { printf 'gofmt quiere cambiar:\n%s\n' "$out" >&2; return 1; }
}

# Secretos: nada de claves privadas, llaves de servicio ni URLs de base con una contraseña real en lo versionado.
# Las URLs de la base desechable de pruebas (postgres:postgres@127.0.0.1 / localhost) están permitidas.
no_secrets() {
  # Los archivos de prueba usan claves y URLs de mentira a propósito (TESTONLY, s3cr3t-pass): se dejan fuera.
  local skip=(':!scripts/check.sh' ':!*_test.go' ':!frontend/tests/*')
  local hits urls
  hits=$(git grep -nIE -e '-----BEGIN [A-Z ]*PRIVATE KEY-----' -e 'sb_secret_[A-Za-z0-9_]{8,}' -e 'service_role[" :=]+eyJ' -- . "${skip[@]}" || true)
  urls=$(git grep -nIE 'postgres(ql)?://[A-Za-z0-9_.-]+:[^@/ <>$]{4,}@' -- . "${skip[@]}" \
    | grep -vE 'postgres:postgres@(127\.0\.0\.1|localhost)|user:password@' || true)
  if [ -n "$hits$urls" ]; then printf 'posible secreto en lo versionado:\n%s\n%s\n' "$hits" "$urls" >&2; return 1; fi
}

if [ -z "${TEST_DATABASE_URL:-}" ]; then
  echo "aviso: TEST_DATABASE_URL no está definida; las pruebas de base de datos se saltarán." >&2
fi

step "gofmt (backend)"      gofmt_clean
step "go vet"               bash -c 'cd backend && go vet ./...'
step "go test"              bash -c 'cd backend && go test -count=1 ./...'
step "eslint (frontend)"    bash -c 'cd frontend && npm run lint --silent'
step "tipos (tsc -b)"       bash -c 'cd frontend && npx tsc -b'
step "npm test"             bash -c 'cd frontend && VITE_API_URL="${VITE_API_URL:-https://api.example.test}" npm test --silent'
step "build del frontend"   bash -c 'cd frontend && VITE_API_URL="${VITE_API_URL:-https://api.example.test}" npm run build --silent'
step "npm audit (producción)" bash -c 'cd frontend && npm audit --omit=dev --audit-level=high'
step "secretos"             no_secrets

if [ "$FULL" = 1 ]; then
  : "${CHROMIUM_PATH:?define CHROMIUM_PATH con el navegador que usan las pruebas (ver docs/operacion.md)}"
  step "pruebas responsive" bash -c 'cd frontend && VITE_API_URL="${VITE_API_URL:-http://127.0.0.1:4000}" npm run test:responsive --silent'
  step "pruebas CSP"        bash -c 'cd frontend && VITE_API_URL="${VITE_API_URL:-https://api.example.test}" npm run test:csp --silent'
fi

printf '\n'
if [ "${#failed[@]}" -gt 0 ]; then
  printf 'REVISIÓN FALLIDA en: %s\n' "${failed[*]}" >&2
  exit 1
fi
echo "Todo en orden."
