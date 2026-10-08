# Backlog

Ordenado por sprint. Cada sprint sigue el mismo método: contrato, reglas de aceptación aprobadas, tests congelados, implementación con evidencia, documentación. Última actualización: 2026-10-08.

## Sprint 0, fundaciones (en curso)

- [x] Helper de Postgres local de pruebas que rechaza cualquier host no local (AC1).
- [x] Tests de caracterización del cálculo de precios y de las unidades (AC2, AC3).
- [x] Migraciones 1 a 14: aplicar, revertir y aplicar de nuevo (AC4).
- [x] Higiene del repo y `tmp/build-errors.log` fuera de git (AC5).
- [x] `docker-compose` de pruebas y CI (AC6).
- [x] ADR-001 (borrador), fuentes y este backlog (AC7).

## Sprint 1, vivo en Vercel

- [ ] Handler de Vercel en `api/`, manteniendo `cmd/server` para desarrollo local, y `vercel.json`.
- [ ] pgx compatible con el pooler de transacciones (sin prepared statements), con límite de conexiones y timeouts.
- [ ] CORS con lista de orígenes por variable de entorno (hoy hay un solo `FRONTEND_URL`); quitar `AllowCredentials`, el token va en el header.
- [ ] Exigir `JWT_SECRET` al arrancar (hoy un secreto vacío podría aceptar tokens; probarlo).
- [ ] Migraciones dentro del esquema `manager`, con la tabla de control en ese esquema, RLS activado y sin tocar `public`. Revisar `CREATE EXTENSION pgcrypto` en Supabase.
- [ ] Rol de base de datos propio del manager, con permisos solo sobre `manager`.
- [ ] Frontend: URL de la API por variable de entorno y rewrite de SPA (ya existen; verificar con el build).
- [ ] Runbook `docs/despliegue.md`, con el respaldo manual del esquema.
- [ ] Verificar en el log del primer despliegue la versión de Go.

## Sprint 2, dinero

- [ ] **Defecto con evidencia:** insumos $1.000, mano de obra 20%, margen 25%, rinde 9 deben dar $1.500 (hoy $2.000). Cambia el test `TestAC2_KnownDefectFloatDriftRaisesThePriceStep`, con aprobación.
- [ ] **Margen aplicado dos veces** en `Simulate` cuando el desglose ya incluye el margen (1300 pasa a 1690). Cambia `TestAC2_SimulateAppliesMarginOnTopOfTheBreakdown`, con aprobación.
- [ ] Montos finales en CLP enteros (ventas, gastos, costos fijos, cotizaciones, precio sugerido), con expand/contract.
- [ ] Precios unitarios y cantidades con aritmética exacta (probablemente una librería decimal, a justificar antes de agregarla); redondear solo al final.

## Sprint 3, tiempo

- [ ] Una sola función de "hoy" en `America/Santiago` para ventas, gastos y analytics.
- [ ] `sale_date DATE DEFAULT CURRENT_DATE` usa la zona del servidor.
- [ ] Gasto con fecha de hoy rechazado como futuro: `Truncate(24h)` corta en UTC, no en Santiago (por lectura del código, falta el test).
- [ ] `parseMonthYear` de analytics usa `time.Now()` sin zona.
- [ ] `users.created_at` e `ingredients.created_at/updated_at` son `TIMESTAMP` sin zona; pasarlos a `TIMESTAMPTZ`.

## Sprint 4, consistencia de ventas

- [ ] La venta descuenta stock en la unidad de la receta sin convertirla a la del ingrediente (por lectura; los costos sí usan `ConvertToBase`). Verificar con test.
- [ ] Chequeo de stock y descuento sin bloqueo de fila: dos ventas simultáneas pueden pasar el chequeo.
- [ ] Stock insuficiente por la restricción de base responde 500; debe ser 409.
- [ ] `recipe_id` de una venta sin validar contra la receta.
- [ ] Cotización que se entrega sin guardarse: el error de `repo.Save` se ignora.
- [ ] Unidades con mayúscula (`Kg`) no se reconocen (hoy fijado en `TestAC3_ConvertToBaseRejectsIncompatibleUnits`).

## Sprint 5, acceso y roles

- [ ] ADR-002: el negocio es uno solo y ambas personas administran. `user_id` pasa a ser `created_by` (quién lo registró).
- [ ] Hoy ventas, gastos, costos fijos, cotizaciones y analytics filtran por usuario: cada uno vería solo su mitad del negocio. Quitar ese filtro.
- [ ] Ingredientes sin `created_by`; recetas se leen todas pero solo el creador las edita. Unificar.
- [ ] El rol `helper` queda sin uso y documentado.
- [ ] Aserciones de tipo sin chequeo en el middleware (`claims["sub"].(string)`): un token sin esos campos produce 500 en vez de 401.
- [ ] Límite de intentos de login (se reemplaza en la etapa 1; evaluar si vale la pena antes).

## Sprint 6, calidad

- [ ] `GetAll` de recetas hace N+1 consultas.
- [ ] Límites de tamaño del cuerpo en los endpoints JSON (solo hay límite en los de archivos).
- [ ] Logs de gastos incluyen la descripción: sacar datos personales de los logs.
- [ ] `xlsx@0.18.5` en el frontend tiene vulnerabilidades conocidas en la versión del registro npm (por verificar).
- [ ] `go.mod` con todas las dependencias marcadas `// indirect`: correr `go mod tidy` en un commit aparte, con aprobación.
- [ ] Token en `localStorage` (se reemplaza en la etapa 1).

## Etapas siguientes (después del Sprint 6)

- [ ] **Etapa 1:** identidad única con Supabase Auth (Google), verificación JWKS ES256.
- [ ] **Etapa 2:** `recipe_id` opcional en el producto del sitio; precio sugerido solo de referencia.
- [ ] **Etapa 3:** pedidos del sitio crean cotizaciones `pending`. Antes, verificar la Ley 21.719.
