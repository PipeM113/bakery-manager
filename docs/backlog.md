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

- [x] Handler de Vercel en `backend/api/`, manteniendo `cmd/server` para desarrollo local (S1a). `backend/vercel.json` agregado en S1b.
- [x] pgx compatible con el pooler de transacciones (modo `Exec`), con límite de conexiones y timeouts (S1a). Falta probarlo contra el pooler real.
- [x] CORS con lista de orígenes por `CORS_ORIGINS`, sin `AllowCredentials` (S1a).
- [x] Exigir `JWT_SECRET` de al menos 32 caracteres al arrancar, inyectado en el middleware (S1a).
- [x] Migraciones dentro del esquema `manager`, con la tabla de control en ese esquema, RLS activado (migración 15) y sin tocar `public`; pgcrypto lo instala el script de setup (S1b). Falta correrlo en Supabase real.
- [x] Rol `manager_app` con permisos solo sobre `manager` (`backend/db/setup/01_manager_schema.sql`, S1b). Falta aplicarlo en Supabase real.
- [x] Frontend: el build falla sin `VITE_API_URL` y ya no cae a localhost; job de CI del frontend (S1b).
- [x] Runbook `docs/despliegue.md`, con el respaldo manual del esquema (S1b).
- [ ] Verificar en el log del primer despliegue la versión de Go y los demás puntos de "Verificación en el mundo real" de `docs/despliegue.md` (los ejecuta Felipe).
- [x] **Primer usuario (`cmd/hashpassword` y paso del runbook, S1b):** no hay registro en la API; sin un usuario en `manager.users` nadie puede iniciar sesión. Herramienta para generar el hash de la contraseña sin que pase por el chat, y paso en el runbook.

## Sprint 1c, fotos en Supabase Storage (reemplaza a Cloudinary)

Decidido el 2026-10-09 (opción 3 sobre Vercel Blob; se parte sin fotos previas, no hay nada que migrar). Va después de mergear #2 y #3; el despliegue del Sprint 1b no depende de esto (la subida de fotos ya es opcional).

- [ ] Spike: confirmar contra la documentación de Supabase Storage el contrato REST de subida (ruta, cabeceras, respuesta, URL pública). Hoy solo está verificado el modelo de buckets (público/privado, límites por bucket, tipos permitidos) y las claves.
- [ ] Config: `SUPABASE_URL`, `SUPABASE_SECRET_KEY` y `SUPABASE_STORAGE_BUCKET`, las tres o ninguna; ningún error repite la clave.
- [ ] Subida desde el backend con la clave secreta; nombre único por subida (evita caché de la foto vieja); devuelve la URL pública.
- [ ] Validar en el handler: máximo 4 MB (la función de Vercel admite 4,5 MB de cuerpo) y tipo real por contenido (jpeg, png, webp), no por el nombre ni por la cabecera.
- [ ] Quitar Cloudinary: paquete `internal/cloudinary`, dependencia `cloudinary-go`, `CLOUDINARY_URL` en config, `.env.example`, runbook y test de higiene (que pase a vigilar `SUPABASE_SECRET_KEY`).
- [ ] Runbook: crear el bucket, la clave propia del manager y las variables en Vercel.
- [ ] Limpieza de la foto anterior al reemplazarla (best effort), si se decide que vale la pena.
- [ ] Verificación real (Felipe): subir una foto, abrir su URL pública, ver la foto en el PDF de la cotización, y comprobar que el bucket rechaza un archivo de más de 4 MB y uno que no es imagen.

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
