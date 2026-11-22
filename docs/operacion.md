# Operación

Cómo se corre, se revisa y se cuida el bakery-manager. El procedimiento paso a paso de la primera instalación está en `docs/despliegue.md`; las decisiones, en `docs/adr/`.

## Qué hay en producción
- **Base:** Supabase, esquema `manager` (rol `manager_app`). Plan Free: **sin respaldos automáticos** y se pausa tras una semana sin actividad.
- **Backend:** Vercel, proyecto `bakery-manager-back`. **Frontend:** Vercel, proyecto `bakery-manager`. Cada merge a `main` despliega solo.
- **Fotos:** Supabase Storage (el bucket del proyecto).
- **Variables de entorno:** solo en Vercel (lista en `docs/despliegue.md`, sección 5). Nunca en el repo.

## Revisar el código antes de un commit
```sh
export TEST_DATABASE_URL=postgres://postgres:postgres@127.0.0.1:54330/postgres   # la base desechable (README)
./scripts/check.sh          # rápido: formato, vet, tests Go, lint, tipos, tests, build, audit, secretos
./scripts/check.sh --full   # además pruebas de navegador (responsive y CSP); necesita CHROMIUM_PATH
```
El CI corre lo mismo que la revisión rápida. Un PR no se mergea con el CI en rojo.

## Publicar un cambio
1. Rama, commits convencionales, PR. El CI debe quedar verde.
2. Si el cambio trae una **migración**: respaldo → (consulta de verificación si la migración la pide) → aplicar la migración → verificar versión y dueños → **recién ahí** mergear (el backend nuevo lee las columnas nuevas). Ver `docs/despliegue.md`, sección 2.
3. Mergear. Vercel despliega solo. Probar a mano lo que cambió.

## Respaldo
- **Cuándo:** una vez por semana mientras dure el piloto y **siempre antes de una migración**.
- **Cómo:** `infra/backup/backup.sh` y `restore-check.sh` (sección 7 de `docs/despliegue.md`). Necesita `pg_dump` 17 (Supabase corre PostgreSQL 17) y el pooler de sesión.
- Guarda una copia **fuera del computador** donde se hizo. Un respaldo que nunca se restauró no cuenta.
- Alternativa de emergencia, sin `pg_dump`: exportar cada tabla del esquema `manager` como JSON con una consulta de solo lectura (se hizo el 2026-11-18 y el 2026-11-21) y probar que se puede cargar en una base local con el esquema de las migraciones.
- **Estado:** `backup.sh` **aún no se corrió contra Supabase**.

## Qué hacer si algo falla
| Síntoma | Primero mira | Después |
|---|---|---|
| Cualquier error que vea tu mamá | El código de la solicitud: los avisos de error del servidor lo muestran («… (código d0ff621e1b63e18a)»); es el encabezado `X-Request-Id` de la respuesta y la línea del registro de Vercel lo lleva entre corchetes | Buscar ese identificador en los logs del proyecto `bakery-manager-back` |
| El login da 500 | Dueños de las tablas del esquema (consulta de `docs/despliegue.md`, sección 2): tiene que ser `manager_app` | `alter table manager.<tabla> owner to manager_app;` |
| Una pantalla no carga o da error | Los errores de ejecución del backend en Vercel (logs del proyecto `bakery-manager-back`) | Si empezó después de un deploy, revertir ese deploy en Vercel (rollback) |
| Se aplicó una migración y algo se rompió | **No toques la base.** Revisa `select version, dirty from manager.schema_migrations;` | Restaurar el último respaldo en una base local para comparar; el `down` no deshace una fusión de datos |
| El PDF de la cotización falla | Datos del negocio completos (la pantalla avisa) y `BIZ_RUT`/`BIZ_ACCOUNT` en Vercel | Ver `docs/cotizaciones.md` |
| No se suben fotos | Variables del bucket y la CSP del frontend (`frontend/vercel.json`) | Ver `docs/despliegue.md` |
| La base no responde tras días sin uso | El proyecto de Supabase puede estar pausado (plan Free) | Reactivarlo desde el panel |

## Niveles de servicio (propuesta, por confirmar con Felipe)
Una herramienta interna de una usuaria, sin guardia. Estos números son una **propuesta**, no una medición:
- **Disponibilidad:** se espera en horario de trabajo; una caída de un día es molesta, no grave.
- **Datos:** no se pierde más de una semana de trabajo (por eso el respaldo semanal); cada migración tiene respaldo propio.
- **Cambios:** una corrección urgente puede salir el mismo día; lo demás, por sprint.

## Pendientes de operación
- [ ] Correr `backup.sh` contra Supabase y restaurarlo en local (`docs/backlog.md`).
- [ ] Confirmar que `manager` no está en «Exposed schemas» y las tres filas de la consulta del paso 1 de `docs/despliegue.md` dan `false`.
- [ ] Activar las alertas de seguridad de Dependabot (Settings, Code security).
- [ ] Revisar cada mes los PRs de Dependabot (frecuencia mensual).
