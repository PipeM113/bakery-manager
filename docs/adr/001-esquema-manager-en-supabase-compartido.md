# ADR-001: bakery-manager usa el esquema `manager` del Supabase del sitio, desplegado en Vercel

- **Estado:** Aceptado (Felipe, 2026-10-08; el borrador lo redactó Claude)
- **Fecha:** 2026-10-08

## Contexto

bakery-manager es la herramienta interna de la pastelería (ingredientes, recetas, costos, cotizaciones, ventas, gastos, analítica). La usan dos personas, ambas administradoras: Felipe y su mamá. Hoy no corre: el backend y la base estaban en Railway y la prueba terminó. No hay datos que migrar.

El sitio público (repo `angeles_bakery_web`) ya funciona sobre un proyecto de Supabase (Postgres, Auth con Google, Storage) y dos proyectos de Vercel (su ADR-006). Se quiere que ambos proyectos se coordinen más adelante, pero sin mezclarlos ahora.

Hechos verificados en `docs/fuentes.md`: Supabase Free se pausa tras una semana de inactividad y no tiene backups automáticos; el pooler de transacciones (puerto 6543) no soporta prepared statements; Vercel Hobby limita el cuerpo de una solicitud a 4,5 MB y la duración a 300 s.

## Opciones consideradas

| Opción | Costo | Riesgo | Reversibilidad |
|---|---|---|---|
| A. Mismo proyecto de Supabase, esquema propio `manager` | Sin costo adicional | El proyecto compartido es un punto único: pausa, borrado o error afectan a ambos | Media: se exporta el esquema con `pg_dump` y se mueve |
| B. Segundo proyecto de Supabase Free | Sin costo (Free permite 2 proyectos activos) | Dos conjuntos de usuarios: la etapa 1 (identidad única) obligaría a migrar cuentas | Alta al inicio, cara después |
| C. Tablas del manager en `public` junto a las del sitio | Sin costo | `public` se expone por la Data API por defecto; choque de nombres y de tablas de migración | Baja |
| D. Otro host para el backend Go | No evaluado: sin fuentes verificadas | Más piezas que operar | — |
| E. Volver a Railway | Descartada por el propietario: la prueba terminó | — | — |

## Decisión

Opción A, con el backend y el frontend en dos proyectos de Vercel (igual que el sitio), y este plan por etapas:

- **Etapa 0, reactivar sin integrar.** Dejar el manager funcionando con login propio, ordenado en mini sprints (ver `docs/backlog.md`).
- **Etapa 1, identidad única.** Supabase Auth con Google, verificando el token con JWKS como hace el sitio. Reemplaza el login propio.
- **Etapa 2, receta y producto.** `recipe_id` opcional en el producto del sitio; el precio sugerido del manager es solo una referencia.
- **Etapa 3, pedidos.** Los pedidos del sitio crean cotizaciones en estado `pending`.

## Razón

- Una sola base y una sola identidad hacen posibles las etapas 1 a 3 sin migrar cuentas.
- El esquema propio separa tablas, migraciones y permisos del sitio sin pagar otro proyecto.
- Mantener Go evita reescribir lo que ya funciona; reescribir está fuera de alcance.

## Reglas del esquema (consecuencias de la decisión)

- Todas las tablas y la tabla de control de migraciones viven dentro de `manager`.
- RLS activado en todas las tablas, sin políticas, como defensa en profundidad.
- `manager` no se agrega a "Exposed schemas" de la API de Supabase (por defecto solo `public` se expone; ver `docs/fuentes.md`).
- El detalle de roles y permisos de base de datos se decide en el Sprint 1.

## Riesgos y consecuencias

1. **Proyecto compartido y plan Free.** El manager hereda la pausa por inactividad y la falta de backups. Mitigación: respaldo manual del esquema `manager` con `pg_dump`, escrito en el runbook.
2. **Vercel Hobby.** La política dice que Hobby es solo para uso personal no comercial (cita en `docs/fuentes.md`). Decisión del propietario: es una herramienta interna con dos usuarios, sin publicidad ni pagos, así que Hobby sirve por ahora. Es una interpretación suya en una zona gris, no una autorización de Vercel.
3. **Login propio entre la etapa 0 y la 1.** Token HS256 con secreto compartido y sin límite de intentos de login. Mitigación mínima: exigir `JWT_SECRET` al arrancar (Sprint 1). El resto se reemplaza en la etapa 1, no se repara.
4. **Pooler.** El backend en Vercel usa el pooler de transacciones, así que pgx debe funcionar sin prepared statements (Sprint 1). Las migraciones se corren por el pooler de sesión o la conexión directa.
5. **Datos personales.** Las cotizaciones guardan nombres de clientes. La Ley 21.719 está sin verificar (pendiente antes de la etapa 3).
6. **Alcance.** Este ADR no cambia el login ni toca el sitio; solo fija dónde vive el manager.

## Supuestos

| Supuesto | Estado |
|---|---|
| Dos usuarios, ambos administradores | Confirmado por Felipe |
| Hobby alcanza por ahora | Decisión de Felipe |
| Un entrypoint en `/api` que envuelve el router chi funciona en Vercel | Inferido: la firma `Handler(w, r)` está verificada; falta probar el despliegue |
| Vercel usa la última versión de Go por defecto | Observación de Felipe, sin verificar; se mira en el log del primer despliegue |
| Nombre y permisos del rol de base de datos del manager | Pendiente (Sprint 1) |

## Revisar si

- Vercel limita o rechaza el uso del manager.
- El manager empieza a guardar datos de clientes reales a escala, o se activa la etapa 3.
- Supabase pausa el proyecto con frecuencia o hace falta un backup automático.
- Se inicia la etapa 1.
