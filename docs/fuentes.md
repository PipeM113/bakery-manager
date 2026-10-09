# Fuentes verificadas

Consultadas el 2026-10-08. Precios y límites cambian: reverificar antes de decidir algo que dependa de ellos. Las filas marcadas "heredada" vienen de `docs/fuentes.md` del repo `angeles_bakery_web`, con su fecha original; se reutilizan sin reconsultarlas.

| Tema | Fuente | Dato usado |
|---|---|---|
| Vercel: runtime de Go | https://vercel.com/docs/functions/runtimes/go (2026-10-08, vía la búsqueda de documentación de Vercel; la página web está bloqueada en el entorno de trabajo) | Una función va en `/api` y exporta `Handler(w http.ResponseWriter, r *http.Request)` en el paquete `handler`. El build se puede personalizar con `buildCommand` o `GO_BUILD_FLAGS` en `vercel.json`. **No obtuve las versiones de Go soportadas.** |
| Vercel Hobby (heredada, 2026-10-03) | https://vercel.com/docs/plans/hobby | Gratis; duración máxima de función 300 s. |
| Vercel: límites de Functions (heredada, 2026-10-03) | https://vercel.com/docs/functions/limitations | El cuerpo de una solicitud o respuesta no puede pasar de **4,5 MB** (si no, 413 `FUNCTION_PAYLOAD_TOO_LARGE`). |
| Vercel: uso comercial (heredada, 2026-10-03) | https://vercel.com/docs/limits/fair-use-guidelines | "Hobby teams are restricted to non-commercial personal use only." Comercial incluye "advertising the sale of a product or service" y procesar pagos. **Decisión del propietario (2026-10-08):** el manager es una herramienta interna con dos usuarios; Hobby sirve por ahora. Es su interpretación, no una autorización de Vercel. |
| Vercel: proyectos por repositorio (heredada, 2026-10-03) | https://vercel.com/docs/limits | Hobby admite hasta 25 proyectos conectados al mismo repositorio; en un monorepo se crea un proyecto por directorio, cada uno con su *Root Directory*. |
| Supabase: tipos de conexión (2026-10-08, confirma la heredada del 2026-10-03) | https://supabase.com/docs/guides/database/connecting-to-postgres | **Directa** (`db.<ref>.supabase.co:5432`): IPv6 (IPv4 solo con add-on de pago). **Pooler de sesión** (5432): IPv4 en todos los planes. **Pooler de transacciones** (6543): IPv4, pensado para serverless; **"does not support prepared statements"**. Para Vercel se usa el pooler de transacciones; para migraciones, el de sesión o la directa. |
| Supabase Free (heredada, 2026-10-02) | https://supabase.com/pricing | 500 MB de base de datos, 2 proyectos activos. **Se pausa tras 1 semana de inactividad. Sin backups automáticos.** |
| Supabase: esquemas propios (2026-10-08) | https://supabase.com/docs/guides/api/using-custom-schemas | `public` está expuesto por la Data API por defecto; un esquema propio solo se expone si se agrega a "Exposed schemas" en la configuración de la API. |
| Supabase: Row Level Security (2026-10-08) | https://supabase.com/docs/guides/database/postgres/row-level-security | RLS debe estar activado en las tablas de un esquema expuesto. Las claves de servicio saltan RLS y nunca van al navegador. |
| Acciones de GitHub (heredada, 2026-10-03) | releases de https://github.com/actions/checkout | `actions/checkout` v7.0.1, commit `3d3c42e5aac5ba805825da76410c181273ba90b1`. Se fija por SHA completo en el CI. |
| `actions/setup-node` (2026-10-09) | etiquetas de https://github.com/actions/setup-node (`git ls-remote --tags`) | `v7.1.0` apunta al commit `949feb2413d6458794dcd2491c4babbbce0c15c1`; se fija por SHA completo en el job del frontend. |
| Go y su toolchain (observación propia, 2026-10-08) | comprobado en la sesión de trabajo | Con Go 1.24.7 instalado y `go 1.25.0` en `go.mod`, el comando `go` baja y usa el toolchain 1.25.0 solo (`GOTOOLCHAIN=auto`). El CI se apoya en esto. |
| GitHub Actions: primera ejecución del CI (observación propia, 2026-10-08) | check `backend-tests` del PR #1 de `PipeM113/bakery-manager` | El job terminó con conclusión `success` en unos 40 s (vet y tests con Postgres 16 de servicio). Por lo tanto el runner `ubuntu-24.04` trae Go y usa el toolchain que pide `go.mod`. No leí el log del job; la conclusión sale del estado del check. |
| Defecto de redondeo con `float64` (observación propia, 2026-10-08) | script aparte y `TestAC2_KnownDefectFloatDriftRaisesThePriceStep` | Insumos $1.000, mano de obra 20%, margen 25%, rinde 9: el precio exacto es $1.500, pero `float64` calcula 1500,000000000000227 y `ceilTo500` lo sube a $2.000. Hay otras combinaciones con el mismo problema. |
| Supabase Storage: buckets (2026-10-09) | https://supabase.com/docs/guides/storage/buckets/fundamentals | Un bucket público permite leer con la URL a cualquiera; subir, borrar y mover siguen exigiendo autorización. Las restricciones de tamaño y tipos se definen por bucket. |
| Supabase Storage: límites (2026-10-09) | https://supabase.com/docs/guides/storage/uploads/file-limits | En Free el límite global de archivo no puede superar 50 MB; el del bucket no puede superar el global. |
| Supabase Storage: tamaño total (2026-10-09) | https://supabase.com/docs/guides/platform/manage-your-usage/storage-size | Free incluye 1 GB de almacenamiento. |
| Supabase: claves de API (2026-10-09) | https://supabase.com/docs/guides/getting-started/api-keys | Las claves secretas (`sb_secret_...`) dan acceso completo al proyecto y se saltan RLS; solo para backend; se recomienda una por componente; se rotan creando otra y borrando la comprometida. |
| Supabase Storage: control de acceso (2026-10-09) | https://supabase.com/docs/guides/storage/security/access-control | Una clave de servicio se salta las políticas de Storage. Sin ella, subir exige una política `INSERT` en `storage.objects`. |
| Vercel Blob (2026-10-09, descartado) | https://vercel.com/docs/vercel-blob/private-storage | Existe acceso público y privado; el SDK oficial es de JavaScript. No se encontraron límites ni precios de Hobby ni un cliente oficial para Go. |

## No verificado todavía

- Versiones de Go soportadas por Vercel y cuál usa por defecto (Felipe cree que la última; confirmarlo en el log del primer despliegue).
- Que un solo `api/index.go` que envuelva el router chi, con un rewrite de todas las rutas, funcione en Vercel. Es una inferencia a partir de la firma verificada.
- Comportamiento exacto de pgx 5.8.0 contra el pooler de transacciones (qué modo de ejecución usar). Verificar en el código de la librería y con una prueba en el Sprint 1.
- Ley 21.719 (datos personales, Chile): vigencia y obligaciones, antes de la etapa 3. Las cotizaciones guardan nombres de clientes.
- Interpretación de Vercel sobre "uso comercial" para una herramienta interna.
- Contrato REST de subida de Supabase Storage (ruta, cabeceras, respuesta, forma de la URL pública): se verifica en el spike del Sprint 1c.
- Precio y límites de egreso de Supabase Storage en Free: no consultados.
