# Backlog

Cada sprint sigue el mismo método: contrato, reglas de aceptación aprobadas, tests escritos y vistos en rojo, aprobación, implementación con evidencia y documentación. "Aceptado" significa que el propietario lo probó en la app real. Última actualización: 2026-10-19 (cierre del sprint 13).

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
- [x] Dependencias de producción del frontend actualizadas (`axios`, `react-router-dom`).

## Pendiente

- [ ] **`xlsx@0.18.5` (decisión del propietario):** dos avisos de gravedad alta sin versión corregida en npm (ver `docs/fuentes.md`). Opciones: quitar la vista previa del archivo (el servidor ya valida), o cambiar de biblioteca.
- [ ] Probar S6 a S10, S12 y S13 en la app real, sobre todo desde el celular, y marcarlos como aceptados.
- [ ] Medir en Vercel el tiempo de una cotización de 30 líneas con fotos (nunca se midió).
- [ ] Cargar los datos reales con su mamá, escribirle un manual corto de una página y hacer un piloto de 1 a 2 semanas.
- [ ] Respaldo del esquema con `pg_dump` y verificar "Exposed schemas" en Supabase (`docs/despliegue.md`).

## Ideas anotadas, no decididas

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
