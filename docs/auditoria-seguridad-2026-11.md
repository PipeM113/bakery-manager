# Auditoría mínima de seguridad pre-lanzamiento (2026-11-10)

Hecha con el checklist de 20 puntos (`checklist-seguridad-prelanzamiento`), sobre `chore/sprint-15` (después del sprint 15) y sobre Supabase por consultas **de solo lectura**. No se probó nada contra la app desplegada. **No cambia código**: los arreglos son propuestas.

Estados: OK, FALLA, PARCIAL, N/A (no aplica a esta app) y NO VERIFICABLE (depende de algo fuera del repo; dice quién lo confirma).

| # | Punto | Estado | Evidencia | Arreglo propuesto |
|---|---|---|---|---|
| 1 | Keys ocultas | OK | El frontend solo lee `VITE_API_URL` (`frontend/src/services/api.ts:5`); las claves de Supabase y `JWT_SECRET` solo se leen en `backend/internal/config`. Ningún patrón de clave en el historial de git. | — |
| 2 | Secretos fuera de Git | OK | `.env` en `.gitignore`; solo `*.env.example` versionados; sin secretos en el historial (búsqueda de `sb_secret_` y de JWT). | — |
| 3 | Key pública para la DB | OK | El frontend no usa Supabase; la clave secreta (`SUPABASE_SECRET_KEY`) vive solo en el backend (ADR-001, riesgo 6). | Mantener una clave propia del manager y revocable. |
| 4 | RLS en todas las tablas | OK | Consulta: 18 de 18 tablas de `manager` con RLS, 0 políticas (sin política = sin acceso), 0 grants a `anon`/`authenticated`, dueño `manager_app` en todas. `manager` no figura en los esquemas expuestos de la API. | — |
| 5 | Datos sensibles y PII en logs | PARCIAL | Los logs no llevan correo, nombre de cliente ni teléfono (búsqueda en `log.Printf`). El cifrado en reposo y en tránsito lo da Supabase. | Confirmar que `DATABASE_URL` use `sslmode=require` (hoy el valor por defecto de pgx es `prefer`). |
| 6 | Auth forzada | OK | Todo cuelga de `mid.Auth` salvo `/auth/login` y `/health` (`internal/app/app.go`). | — |
| 7 | Acceso a registros (IDOR) | N/A | Un solo negocio con dos cuentas administradoras (ADR-002): todo se comparte a propósito. | Revisar si algún día hay una persona con permisos reducidos. |
| 8 | Campos manipulables | OK | `created_by` y `id` los pone el servidor; el `role` del token no viene del body; `PUT /business` rechaza campos desconocidos; `POST /recipes` rechaza `photo_url` (S15). | — |
| 9 | Sesión | PARCIAL | No hay cookies: el token viaja en `Authorization` y vive en `localStorage` (24 h, sin cierre en el servidor; un usuario borrado conserva su token hasta que expire). Ya está en ADR-001 (riesgo 3) y en el backlog. | Se reemplaza en la etapa 1 (Supabase Auth). Mientras tanto, no hay nada más barato que valga la pena. |
| 10 | Contraseñas | OK | bcrypt con costo por defecto; mínimo y máximo de largo (`auth.go`); hash ficticio para igualar el tiempo con un correo inexistente. Tabla propia `users`, a propósito (login propio hasta la etapa 1). | — |
| 11 | Rate limiting | PARCIAL | Bloqueo por correo en el login: 5 intentos en 15 minutos (`login_attempts`). No hay límite global por IP ni por endpoint. | Bajo riesgo con dos usuarios; si se publica el backend, usar el firewall de Vercel. Anotado en el backlog (bloqueo por IP). |
| 12 | Bots | N/A | No hay formularios públicos, registro ni recuperación de contraseña. | — |
| 13 | Queries parametrizadas | OK | Sin concatenación de entrada en SQL (búsqueda de `Sprintf` y `+` con `WHERE`/`SELECT`); todo va con `$n`. | — |
| 14 | Validación en el servidor | OK | Cada handler valida tipo, rango y largo (unidad, precios, tamaños, equivalencias, proveedor ≤ 100); cuerpos JSON ≤ 1 MiB (413). No hay una librería de esquemas: son validaciones a mano, cubiertas por tests. | — |
| 15 | Contenido sanitizado | OK | Sin `dangerouslySetInnerHTML` ni `innerHTML` en el frontend; React escapa por defecto. | — |
| 16 | Archivos | PARCIAL | Foto: tipo por el contenido (`http.DetectContentType`), nombre aleatorio, máximo 4 MB. El bucket `recipe-photos` es público a propósito (se ve en la app y en el PDF), con límite de 3 MB y solo jpeg/png/webp. **El límite del bucket (3 MB) es menor que el del código (4 MB)**: una foto de 3 a 4 MB pasa el backend y la rechaza Supabase con un error genérico. | Alinear los dos límites (bajar el del código a 3 MB o subir el del bucket). Es un cambio de una línea. |
| 17 | Devolver solo lo necesario | PARCIAL | El login devuelve solo el token. Dos consultas de gastos usan `RETURNING *` (`operational_expenses/repository/repository.go:61,104`); no exponen datos sensibles, pero dependen de que la tabla no crezca con columnas privadas. | Nombrar las columnas. Baja prioridad. |
| 18 | Security headers y CORS | FALLA (endurecimiento) | Ni el backend ni `frontend/vercel.json` ponen HSTS, CSP, `X-Content-Type-Options` ni `Referrer-Policy`. CORS está bien: lista de orígenes exactos, sin `*` y sin credenciales (`app.go`, `config.go`). | Agregar `headers` en `frontend/vercel.json` (`X-Content-Type-Options: nosniff`, `Referrer-Policy`, `X-Frame-Options`/`frame-ancestors`, CSP que permita solo el backend y el bucket). Vercel puede añadir HSTS por su cuenta: confirmarlo (abajo). |
| 19 | HTTPS | NO VERIFICABLE | `SUPABASE_URL` se exige `https` (`config.go`) y las fotos del PDF solo salen del bucket por `https` (P2). La redirección de HTTP y HSTS los resuelve Vercel. | Felipe: `curl -sI http://<frontend>` debe redirigir a `https`. |
| 20 | Dependencias | **FALLA** | `npm audit --omit=dev`: **1 vulnerabilidad alta** en `follow-redirects` 1.16.0 a 1.16.1 (viene de `axios`; GHSA-mrwp-f3q8-3g9p: las credenciales pueden seguir una redirección). Hay arreglo con `npm audit fix`. Un aviso moderado más, en `minimatch` (desarrollo). Lockfiles commiteados (`package-lock.json`, `go.sum`). Sin `dependabot.yml`. `govulncheck` no se pudo correr (el proxy bloquea `vuln.go.dev`). | Actualizar con `npm audit fix` y probar; agregar Dependabot (npm, gomod, github-actions); correr `govulncheck` desde otro entorno. En el navegador `axios` usa XHR, no `follow-redirects`, así que el riesgo práctico es bajo, pero el aviso es de producción y hay arreglo. |

## Arreglar antes de lanzar (críticos del checklist: puntos 1 a 4, 6 a 8)

Ninguno falla: los puntos 1, 2, 3, 4, 6 y 8 están OK; el 7 no aplica por el ADR-002.

## Mejorar después, por orden

1. **Punto 20 (alta, pero de bajo impacto práctico):** `npm audit fix` y Dependabot.
2. **Punto 18:** headers de seguridad en `frontend/vercel.json`.
3. **Punto 16:** alinear el límite de la foto (bucket 3 MB, código 4 MB).
4. **Punto 5:** `sslmode=require` en `DATABASE_URL`.
5. Puntos 9 y 11: ya están en el backlog (etapa 1 y bloqueo por IP).
6. Punto 17: columnas explícitas en `RETURNING`.

## No verificable desde el repo: quién lo confirma

| Qué | Quién | Cómo |
|---|---|---|
| Redirección de HTTP a HTTPS y HSTS en los dos dominios | Felipe | `curl -sI` a cada dominio |
| `DATABASE_URL` con `sslmode=require` | Felipe | Variables del proyecto backend en Vercel |
| Que `manager` siga fuera de "Exposed schemas" | Felipe | Panel de Supabase, Settings, API |
| Cifrado en reposo y respaldos de Supabase | Supabase (Free: sin backups automáticos) | `docs/despliegue.md`, sección 7 |
| `govulncheck` sin hallazgos | Felipe o un entorno con acceso a `vuln.go.dev` | `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` |

## Qué no cubre esta auditoría

No es una prueba de penetración ni cubre el sitio público (`angeles_bakery_web`), que comparte el proyecto de Supabase. Para una revisión completa del código queda la opción B del plan (skill `security-audit`).
