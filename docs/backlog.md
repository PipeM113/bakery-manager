# Backlog

Cada sprint sigue el mismo método: contrato, reglas de aceptación aprobadas, tests escritos y vistos en rojo, aprobación, implementación con evidencia y documentación. "Aceptado" significa que el propietario lo probó en la app real. Última actualización: 2026-10-26 (cierre del sprint 14).

## Estado de los sprints

| Sprint | Qué | Código | Aceptado | Documento |
|---|---|---|---|---|
| 0 | Fundaciones: tests de caracterización, migraciones reversibles, CI, higiene del repo | Hecho | — | `docs/adr/001-*` |
| 1, 1b, 1c | En vivo en Vercel y Supabase, esquema `manager`, RLS, fotos en Storage | Hecho; el despliegue y la subida de fotos se verificaron a mano | Sí | `docs/despliegue.md` |
| 2 | Dinero en pesos enteros y aritmética exacta | Hecho | — | `docs/glosario-dinero.md` |
| 3 | Tiempo: el día y el mes son los de Santiago | Hecho | — | `docs/tiempo.md` |
| 4 | Consistencia de ventas | Hecho (se completó con el sprint 7) | — | `docs/ventas-y-stock.md` |
| 6 | Costos fijos con historial por mes | Hecho | Pendiente | `docs/costos-fijos.md` |
| 7 | Ventas: stock, unidades y porciones | Hecho | Pendiente | `docs/ventas-y-stock.md` |
| 8 | Ajustes de stock y merma | Hecho | Pendiente | `docs/ajustes-de-stock.md` |
| 9 | Diseño responsivo (celular y computador) | Hecho | Pendiente | `docs/diseno.md` |
| 10 | Pedidos con varios productos | Hecho | Pendiente | `docs/pedidos.md` |
| 11 | Un solo negocio con cuentas personales | Hecho | Sí | `docs/acceso.md`, `docs/adr/002-*` |
| 12 | Cotizaciones con varios productos y PDF con la marca; datos del negocio editables | Hecho | Pendiente | `docs/cotizaciones.md` |
| 13 | Calidad y detalles | Hecho | Pendiente | `docs/acceso.md`, `docs/despliegue.md` |
| 14 | Ingredientes: proveedor, orden, stock rápido y equivalencias entre unidades | Hecho (migración 28) | Pendiente | `docs/ingredientes.md` |
| 15 | Borrados con motivo, origen de las fotos, una sola regla de necesidades de receta, pruebas de flujos | Hecho | Pendiente | `docs/ingredientes.md` |
| 16 | Tipos de receta (opcionales), buscador y costo en la lista de recetas | Hecho (migración 29) | Pendiente | `docs/recetario.md` |

Los sprints 0 a 4 usan la numeración antigua y son anteriores al registro de aceptación (por eso "—"). El antiguo "sprint 5, acceso y roles" es hoy el sprint 11, y el antiguo "sprint 6, calidad", el 13. La paleta y las tipografías de la marca están en `docs/diseno.md`. Las casillas viejas de este archivo se reemplazaron por esta tabla.

## Sprint 12, cotizaciones (cerrado en código)

- [x] Cotizaciones de 1 a 30 líneas con un delivery, guardadas con copia de precios y nombres; el PDF sale de lo guardado y se puede volver a descargar (migración 24).
- [x] PDF con la marca (colores, Playfair Display y DM Sans, logo), fotos JPEG, PNG y WebP.
- [x] Datos del negocio editables en la app (migración 25); el RUT y el número de cuenta, en `BIZ_RUT` y `BIZ_ACCOUNT` de Vercel.
- [x] Corrección de S9: en el celular la barra inferior tapaba el botón principal de las ventanas.

## Sprint 13, calidad (cerrado en código)

- [x] Límite de intentos de login por correo (5 en 15 minutos) y cambio de contraseña desde "Mi cuenta" (migración 26).
- [x] Cuerpos JSON de más de 1 MiB rechazados con 413; la importación de ingredientes tiene 4 MiB y las fotos 4 MB.
- [x] Los registros del servidor no llevan texto escrito por personas.
- [x] `GET /recipes` con consultas fijas; la cotización consulta una vez por receta distinta.
- [x] Al reemplazar o borrar una receta se borra su foto del bucket (si ninguna otra receta ni cotización la usa).
- [x] Una sola función de formato de dinero (`-$271.000`); `App.css` eliminado; ESLint sin errores y en el CI.
- [x] `go mod tidy` limpio; el CI instala `poppler-utils` para los tests de texto del PDF.
- [x] Dependencias de producción del frontend actualizadas (`axios`, `react-router-dom`) y `xlsx` eliminado junto con la vista previa del archivo de "Importar ingredientes" (decisión del propietario): `npm audit --omit=dev` da 0 avisos.

## Sprint 14, ingredientes (cerrado en código)

- [x] Un ingrediente se identifica por nombre + marca + proveedor: la misma mantequilla en La Vega y en Líder son dos ingredientes (migración 28).
- [x] Listas ordenadas sin distinguir mayúsculas ni tildes; selector de orden en Ingredientes; marca y proveedor al elegir productos en las recetas.
- [x] Botón "+ Stock" para agregar una compra o contar el stock sin entrar a editar.
- [x] Peso de 1 litro y de 1 unidad (opcionales) para usar gr con productos en ml y und con gr; se pide solo cuando una receta lo necesita y sin bloquear lo demás.

## Sprint 15, borrados y fotos (cerrado en código)

- [x] Borrar algo con historia responde 409 con el motivo y la pantalla lo muestra (D1 a D5).
- [x] `POST /recipes` rechaza `photo_url`; el PDF solo descarga fotos del bucket del proyecto y no sigue redirecciones (P1, P2).
- [x] `internal/recipeneeds`: una sola regla de necesidades de receta para costos y ventas (N1).
- [x] Pruebas de flujo en el navegador: stock rápido, cambio de unidad con equivalencia, aviso en la receta y borrado rechazado (F1 a F3, D5).

## Auditoría mínima de seguridad (2026-11)

- [x] Puntos 5 (documentación), 16, 17, 18 y 20 arreglados (`docs/auditoria-seguridad-2026-11.md`).
- [ ] Poner `?sslmode=require` en `DATABASE_URL` en Vercel y comprobar HSTS y la redirección a HTTPS (los confirma Felipe).
- [ ] Correr `govulncheck` desde un entorno con acceso a `vuln.go.dev`.

## Pendiente

- [ ] Probar S6 a S10, S12 y S13 en la app real, sobre todo desde el celular, y marcarlos como aceptados.
- [ ] Medir en Vercel el tiempo de una cotización de 30 líneas con fotos (nunca se midió).
- [ ] Cargar los datos reales con su mamá, escribirle un manual corto de una página y hacer un piloto de 1 a 2 semanas.
- [ ] Correr `infra/backup/backup.sh` contra Supabase y `restore-check.sh` (los scripts están ensayados en local; falta la corrida real) y verificar "Exposed schemas" en Supabase (`docs/despliegue.md`).

## Ideas anotadas, no decididas

- [ ] Archivar recetas e ingredientes con historia en vez de bloquear el borrado (hoy responde 409).
- [ ] Crear el pedido al confirmar una cotización (hoy solo cambia el estado).
- [ ] Editar una cotización guardada (hoy se crea otra).
- [ ] Cerrar las sesiones abiertas al cambiar la contraseña (hoy duran hasta 24 horas).
- [ ] Bloqueo de login también por IP (hoy solo por correo).
- [ ] Restringir de qué servidor descarga fotos el PDF (hoy acepta cualquier dirección `http(s)` guardada en la receta; riesgo bajo, la guardan solo las dos cuentas).
- [ ] Surtidos guardados con nombre y estado de pago en los pedidos.
- [ ] Token de sesión en `localStorage`: se reemplaza en la etapa 1.

## Etapas siguientes (integración con el sitio)

- [ ] **Etapa 1:** identidad única con Supabase Auth (Google), verificación JWKS ES256.
- [ ] **Etapa 2:** `recipe_id` opcional en el producto del sitio; precio sugerido solo de referencia.
- [ ] **Etapa 3:** pedidos del sitio crean cotizaciones `pending`. Antes, verificar la Ley 21.719 (las cotizaciones guardan nombres de clientes).


## Sprint 18, cotizaciones para persona y para empresa (cerrado en código)

- [x] Tipo de cotización (persona o empresa) al crearla; el PDF de persona muestra solo precios finales y el de
  empresa el detalle y las condiciones del pedido (fecha de entrega, forma de pago, validez).
- [ ] Aceptar en la app real (crear una de cada tipo y revisar los PDF).
- Ideas: razón social y RUT del cliente en las de empresa; IVA (neto, IVA y total) si la pastelería emite
  factura; poder editar el tipo y las condiciones de una cotización ya guardada.


## Sprint 19, compras y llave nombre + marca (ADR-005; cerrado en código)

- [x] La llave del ingrediente es nombre + marca; el proveedor, el tamaño del paquete y el precio pagado van en
  cada compra; el costo usa el último precio pagado. Migración 31.
- [x] «+ Stock» pasa a «+ Compra» / «Registrar compra».
- [ ] Antes de aplicar la migración 31 en producción: `pg_dump` completo con `infra/backup/backup.sh` (hay un
  respaldo de datos del 2026-11-18) y revisar que no haya duplicados que necesiten decisión.
- [ ] Aceptar en la app real: registrar una compra con otro tamaño y otro proveedor y ver el costo de la receta.
- Ideas: mostrar junto al último precio el más alto de los últimos 90 días y avisar cuando la última compra fue
  una oferta (el costo de reponer sería mayor); costo por proveedor en una receta si la dueña lo pide; poder
  corregir una compra mal anotada.


## Iteración 0, forma de trabajo y operación (cerrada)

- [x] `CLAUDE.md` con las reglas del proyecto, `scripts/check.sh` (un comando de calidad, con formato y búsqueda de secretos) y `docs/operacion.md` (cómo se corre, se publica, se respalda y qué hacer si algo falla).
- [ ] Confirmar o ajustar los niveles de servicio propuestos en `docs/operacion.md`.
- [x] El CI revisa también el formato de Go (`gofmt`) y la búsqueda de secretos (job `repo-checks`, con `check.sh --only secrets`). Los demás pasos siguen explícitos en el workflow porque las pruebas de `hygiene` y de seguridad leen `ci.yml`.

## Revisión del 2026-11-22: lo que quedó anotado (ver `docs/revision-arquitectura-2026-11.md`)

- [x] Identificador de solicitud: cada respuesta trae `X-Request-Id` y el registro lo imprime en la línea de la solicitud (`pkg/middleware/requestid.go`); el que mande el cliente se ignora.
- [x] Las listas que crecen (cotizaciones y ventas) vienen en páginas: `?limit=` (1 a 1000, 200 por defecto) y `?offset=`, más recientes primero, con `X-Has-More`. Gastos ya se lee por mes. **Falta** que la pantalla avise cuando hay más de 200 (hoy mostraría las 200 más recientes sin decirlo).
- [x] Título del navegador por pantalla (`utils/titles.ts`).
- [x] Los 8 avisos del lint: seis se arreglaron (dependencias de efectos) y dos casos intencionales (guardar al cerrar, cargar una sola vez) quedaron con su motivo escrito. Lint sin avisos. La regla `react-hooks` 7.1 sigue pendiente (4 efectos que llaman `setState`).
- [ ] Prueba de uso con una persona que no conozca el sistema: la pauta está en `docs/prueba-de-uso.md`; falta hacerla.
