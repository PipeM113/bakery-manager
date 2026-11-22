# CLAUDE.md — reglas del proyecto

Herramienta interna de la pastelería de la mamá de Felipe (costeo, recetas, cotizaciones, ventas) y pieza de portafolio. La usa una persona real con datos reales: cuidamos los datos primero.

## Idioma
- **Texto para personas** (docs, mensajes de la app, mensajes de error, commits explicados al usuario): español neutro de Chile, directo.
- **Código, nombres técnicos, comentarios y commits:** inglés (commits en formato convencional: `feat:`, `fix:`, `docs:`, `test:`, `chore:`).

## Cómo se trabaja (no se salta)
1. **Contrato y reglas primero.** Una funcionalidad parte con reglas de aceptación numeradas (Dado/Cuando/Entonces) que Felipe aprueba.
2. **Tests en rojo antes de implementar.** Se escriben, se muestran fallando y Felipe los aprueba. Después se implementa hasta que pasen.
3. **Los tests aprobados no se cambian** sin evidencia y sin pedirle aprobación a Felipe, explicándole qué cambia y por qué. Un cambio permitido siempre se declara (en el commit y en el PR).
4. **Una rama por trabajo** (`chore/sprint-N-...` o `chore/...`), PR pequeño, y no se mergea con el CI en rojo.
5. **Decisiones caras de revertir llevan ADR** en `docs/adr/` (estados: Propuesto, Aceptado, Reemplazado). Un ADR aceptado no se reescribe: se reemplaza.
6. **Datos de terceros** (precios, límites, normas, densidades): solo con fuente oficial citada en `docs/fuentes.md`. Si no hay fuente, se pregunta; no se inventa.
7. **Antes de dar algo por terminado** corre `./scripts/check.sh` (y `--full` si tocaste pantallas) y di qué corrió y con qué resultado.

## Reglas del dominio
- **Dinero en pesos enteros** (CLP), con fracciones exactas dentro de los cálculos y un solo redondeo al final. Nunca punto flotante para mostrar o guardar un monto. Ver `docs/glosario-dinero.md`.
- **Fechas del negocio en hora de Santiago** (`internal/shared/clock`), ver `docs/tiempo.md`.
- **Un ingrediente es nombre + marca** (ADR-005). El proveedor, el tamaño del paquete y el precio pagado viven en cada compra; el costo de una receta usa el **último precio pagado**.
- **Unidades:** todo cruce de unidades pasa por `kernel.Convert`; no hay densidades precargadas (ADR-004).
- **La regla de «cuánto necesita una receta»** vive en `internal/recipeneeds`; costos y ventas la usan, no la copian.
- **Borrar:** lo que tiene historial se protege con un 409 que dice por qué (`kernel.InUseError`).
- **Datos del negocio:** marca en el código, perfil editable en la base, RUT y cuenta en variables de entorno (ADR-003).

## Base de datos y migraciones
- Esquema `manager` en un Supabase compartido (ADR-001). La app entra como `manager_app` por el pooler de transacciones; **todo objeto del esquema debe ser de `manager_app`**.
- Migraciones **aditivas** y con `down`; si un `down` no puede deshacer algo (una fusión), se escribe en el archivo.
- Todo cambio sobre datos reales: **respaldo antes** (`infra/backup/`) y una consulta de dueños después (`docs/despliegue.md`, sección 2).
- Las tablas llevan RLS activo y sin políticas (la app no usa la Data API de Supabase).
- Cada migración nueva se prueba con datos que la hagan trabajar (no solo vacía).

## Seguridad
- Ningún secreto, contraseña ni URL con contraseña en el repo, en un commit, en un log o en el chat. Los valores viven en Vercel y en la terminal de Felipe.
- Las pruebas usan solo la base desechable local y se niegan a correr contra otra (`pgtest`).
- Entrada externa se valida una vez, en la frontera (handler); dentro se confía en los tipos. Mensajes de error al usuario sin SQL ni trazas.
- Cambios a la CSP o a los headers: `frontend/vercel.json` y `docs/despliegue.md`.

## Interfaz y marca
- Colores y tipografías del manual de marca (`docs/diseno.md`): crema, lino, salvia (solo fondo), rosa (solo fondo), negro, oro, oro claro (solo sobre oscuro). Playfair Display y DM Sans. Las fuentes de pago o de uso no comercial no entran al repo sin licencia confirmada.
- Avisos con `useToast()` y confirmaciones con `useConfirm()`; nunca `alert`/`confirm` del navegador.
- Pantallas que cargan: skeletons, no un texto «Cargando…». Táctiles de al menos 44 px en celular.
- Las pantallas se prueban a 360, 390, 768 y 1280 px (`npm run test:responsive`).

## Comandos
- Revisión rápida (lo del CI + formato + secretos): `./scripts/check.sh`. Completa, con navegador: `./scripts/check.sh --full`.
- Operación (correr local, desplegar, respaldar, qué hacer si algo falla): `docs/operacion.md`.
