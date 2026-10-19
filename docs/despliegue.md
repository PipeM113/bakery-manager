# Despliegue: Supabase + Vercel

Procedimiento para dejar el bakery-manager vivo en un proyecto de Supabase compartido (esquema `manager`) y dos proyectos de Vercel (backend y frontend). Decisión y riesgos: `docs/adr/001-esquema-manager-en-supabase-compartido.md`. Fuentes y fechas: `docs/fuentes.md`.

**Regla que no se rompe:** ninguna contraseña, URL con contraseña ni secreto se pega en el chat, en un commit ni en este archivo. Los valores reales viven solo en el panel de Vercel y en tu terminal. Aquí van marcadores como `<contraseña>`.

El ensayo local de este procedimiento (rol, esquema, migraciones, primer usuario, login) corre en los tests de `backend/internal/deploy`. Lo que ese ensayo **no** puede probar (Supabase y Vercel reales) está en la sección "Verificación en el mundo real".

## 0. Antes de empezar

- Proyecto de Supabase con acceso al SQL editor como `postgres` y a "Connect" (cadenas de conexión).
- Go instalado en tu máquina (el repo pide `go 1.25.0`; Go baja el toolchain solo).
- Dos tipos de conexión, que no se mezclan:
  - **pooler de sesión** (puerto 5432, IPv4): lo usas para migrar y para el respaldo.
  - **pooler de transacciones** (puerto 6543): lo usa la aplicación en Vercel. No admite sentencias preparadas; el backend ya lo maneja.
- El usuario del pooler tiene la forma `manager_app.<project_ref>` (rol, punto, referencia del proyecto).
- Genera la contraseña del rol en hexadecimal para no tener que escapar caracteres en la URL:

```sh
openssl rand -hex 24
```

## 1. Rol y esquema

1. Abre `backend/db/setup/01_manager_schema.sql`, copia su contenido al SQL editor de Supabase y reemplaza las dos apariciones de `__MANAGER_APP_PASSWORD__` por la contraseña generada. **No guardes el archivo con la contraseña puesta.**
2. Ejecútalo. Se puede repetir: reutiliza el rol y gana la última contraseña.
3. Crea el rol `manager_app` (sin superusuario, sin crear roles ni bases, sin saltar RLS), el esquema `manager` a su nombre, su `search_path` por defecto y quita permisos a `public` y a los roles de la Data API (`anon`, `authenticated`, `service_role`).

Verifica que la Data API no alcanza el esquema:

```sql
select r.rolname,
       has_schema_privilege(r.rolname, 'manager', 'USAGE') as usage
from pg_roles r
where r.rolname in ('anon', 'authenticated', 'service_role');
```

Las tres filas deben decir `false`. Además, en el panel de Supabase (Data API, configuración) **`manager` no debe estar en "Exposed schemas"**. Esa lista es la que decide qué ve la API REST; déjala como está.

## 2. Migraciones

Corre desde la raíz del repo, con el **pooler de sesión**:

```sh
export MIGRATE_URL='postgresql://manager_app.<project_ref>:<contraseña>@<host-del-pooler>:5432/postgres'
go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.20.1 \
  -path backend/db/migrations -database "$MIGRATE_URL" up
```

- El rol ya cae en el esquema `manager`, por eso no hace falta `search_path` en la URL; la tabla de control `schema_migrations` también queda ahí.
- La migración 15 activa RLS en todas las tablas, sin políticas.
- La migración 16 pasa el dinero a pesos enteros (los montos con decimales se redondean al peso más cercano) y crea las categorías de gasto (`expense_categories`, con RLS). Los gastos que ya existían conservan su categoría. Haz una copia de seguridad antes de correrla en producción (sección de respaldo).
- La migración 17 agrega `ingredients.kind` (`ingredient` o `supply`, para cajas y cartones); lo que ya existe queda como `ingredient`.
- La migración 18 pasa las columnas `TIMESTAMP` (usuarios, ingredientes, historial de precios y recetas) a `TIMESTAMPTZ`; los valores existentes se leen como UTC. Haz una copia de seguridad antes de correrla.
- La migración 19 pasa el monto de los costos fijos a la tabla `fixed_cost_versions` (un monto por mes) y reemplaza `is_active` por `ended_from` (ver `docs/costos-fijos.md`). Hazla **antes** de desplegar el backend nuevo: el backend anterior lee las columnas que la migración quita.
- La migración 20 agrega `sales.sold_as` y `sale_ingredients.stock_deducted` (ver `docs/ventas-y-stock.md`). Solo agrega columnas: se puede aplicar antes o después del deploy, pero el backend nuevo las necesita, así que aplícala **antes**.
- Verifica: `select version, dirty from manager.schema_migrations;` debe dar `20` y `false` (la 19 dejó `fixed_cost_versions`, ver arriba), y `select count(*) from manager.fixed_cost_versions;` debe igualar la cantidad de costos fijos que tenías.
- Deshacer la última: mismo comando con `down 1`.
- Cuando termines, `unset MIGRATE_URL`.

## 3. Primer usuario

La API no tiene registro: sin un usuario en `manager.users` nadie entra. La contraseña no debe pasar por el chat ni por el historial de la terminal.

1. Genera el hash. El programa lee la contraseña de la entrada estándar (escríbela y cierra con Ctrl-D) y escribe solo el hash:

```sh
cd backend && go run ./cmd/hashpassword
```

   (Se admiten hasta 72 bytes; es el límite de bcrypt.)

2. En el SQL editor, con el hash pegado (empieza con `$2a$`):

```sql
insert into manager.users (name, email, password, role)
values ('<nombre>', '<correo>', '<hash>', 'owner');
```

Ambas administran el negocio: crea una fila por persona, las dos con rol `owner`.

## 4. Proyecto de Vercel del backend

Ya existe un proyecto de Vercel para el frontend. El backend es un **segundo proyecto** conectado al mismo repositorio:

1. Crea el proyecto, elige el repo y en **Root Directory** pon `backend`.
2. **Framework Preset: Other.** Vercel detecta Go solo y propone el preset "Go", que espera un servidor (`main.go` escuchando en `PORT`) y no la función de `api/`: con ese preset el build falla con `unused_function`. Cámbialo en Settings, Build & Development Settings, y haz Redeploy. No cambies el comando de build.
3. El archivo `backend/vercel.json` envía todas las rutas a `api/index.go`. La versión de Go la fija `go.mod` (1.25.0); Vercel la descarga en el build.
4. En Settings, Deployment Protection, **desactiva Vercel Authentication** (hace lo mismo en el proyecto del frontend). Activa, la API pide iniciar sesión en Vercel y el navegador recibe un 401. La API sigue protegida por su propio login con JWT.

## 5. Variables de entorno

En el panel del proyecto del backend (Settings, Environment Variables). Nada de esto va en archivos del repo:

| Variable | Valor |
|---|---|
| `DATABASE_URL` | Cadena del **pooler de transacciones** (6543) con el usuario `manager_app.<project_ref>`: `postgresql://manager_app.<project_ref>:<contraseña>@<host-del-pooler>:6543/postgres`. Sin corchetes (los de la plantilla de Supabase), sin comillas y sin espacios. La contraseña, la de `manager_app` (no la de la base), generada con `openssl rand -hex 24`: con `#`, `/`, `?` o espacios la URL es inválida y el backend responde `500` en la verificación del paso 6. |
| `JWT_SECRET` | Aleatorio, de al menos 32 caracteres. Genéralo con `openssl rand -base64 48` y pégalo directo en el panel. |
| `CORS_ORIGINS` | La URL exacta del frontend, sin `/` final y sin `*`, por ejemplo `https://<tu-frontend>.vercel.app`. Varias, separadas por coma. |
| `DB_MAX_CONNS` | `4` (valor por defecto). Cada instancia abre como máximo ese número de conexiones. |
| `SUPABASE_URL` | Opcional, junto con las dos siguientes. El origen del proyecto: `https://<project_ref>.supabase.co`, sin ruta. |
| `SUPABASE_SECRET_KEY` | La clave secreta propia del manager (ver "Fotos de las recetas"). Se pega directo en el panel. |
| `SUPABASE_STORAGE_BUCKET` | `recipe-photos` |

Las tres de Storage van juntas o ninguna: si falta alguna, el backend no arranca y el log dice cuál. Sin ellas la subida de fotos queda desactivada.

En el proyecto del **frontend** agrega `VITE_API_URL` con la URL pública del backend, `https://<tu-backend>.vercel.app`. El build del frontend falla si falta o no es una URL válida. Es una variable de build: cambiarla exige un nuevo despliegue.

### Fotos de las recetas (Supabase Storage)

Las fotos se guardan en un bucket **público** de Supabase: cualquiera que tenga la URL de una foto la ve, pero solo el backend puede subir.

1. En Supabase: **Storage, New bucket**, nombre `recipe-photos`, marcado como **Public**. Activa **Restrict file size** con `4 MB` y **Allowed MIME types** con `image/jpeg, image/png, image/webp`. El límite de 4 MB es porque una función de Vercel admite cuerpos de hasta 4,5 MB.
2. En Supabase: **Settings, API Keys**, crea una **secret key** nueva con el nombre `manager-backend`. Una clave por componente permite revocarla sin afectar al sitio de Angeles Bakery. Esa clave da acceso completo al proyecto: se pega solo en Vercel, nunca en el chat, el repo ni el frontend.
3. En el proyecto del backend en Vercel agrega `SUPABASE_URL`, `SUPABASE_SECRET_KEY` y `SUPABASE_STORAGE_BUCKET` (tabla anterior) y haz Redeploy.
4. Verifica: sube una foto a una receta, abre su URL pública y genera el PDF de una cotización con foto.

Si la clave se filtra: en **Settings, API Keys** crea otra, cámbiala en Vercel, haz Redeploy y borra la comprometida (borrarla es irreversible).

Despliega los dos proyectos.

## 6. Verificación

1. En el log de build del backend aparece la versión de Go que usó Vercel (el equivalente de `go version`, en la línea `Downloading go: ...`); con `go.mod` en 1.25.0 baja esa.
2. Abre `https://<tu-backend>.vercel.app/health`: debe responder `200`.
3. Abre el frontend, inicia sesión con el usuario del paso 3 y crea un insumo.
4. Si `/health` responde `500` con "configuración del servidor inválida", falta o está mal una variable (el log de la función dice cuál, sin mostrar valores).
5. Si el login falla con error de conexión, revisa que `DATABASE_URL` use el puerto 6543 y el usuario con `.<project_ref>`.
6. Si al crear un usuario sale `relation "manager.users" does not exist`, todavía no corriste las migraciones (paso 2). El script del paso 1 solo crea el rol y el esquema vacío.
7. Cada despliegue conserva las variables que tenía al crearse: tras cambiar una variable hay que hacer Redeploy, y probar siempre una dirección estable del proyecto, no un enlace viejo de un despliegue anterior.
8. Los logs de funciones en Hobby se guardan solo una hora: si necesitas la causa de un error, reprodúcelo y míralos enseguida.

## 7. Respaldo manual

El plan Free de Supabase **no hace respaldos automáticos** y pausa el proyecto tras una semana sin actividad. Antes de cambios riesgosos y cada tanto, con el pooler de sesión:

```sh
pg_dump "$MIGRATE_URL" --schema=manager --no-owner --no-privileges -Fc -f manager-$(date +%F).dump
```

Guarda el archivo fuera del repo (contiene los datos del negocio y los hashes de contraseñas). Para restaurar en un proyecto vacío: ejecuta el paso 1 y `pg_restore --no-owner -d "$MIGRATE_URL" manager-<fecha>.dump`.

## Verificación en el mundo real

Confirmado contra Supabase y Vercel reales (2026-10-15 al 2026-10-18):

- [x] Las migraciones corren por el pooler de sesión con el rol `manager_app`.
- [x] El login funciona por el pooler de transacciones, desde el frontend (confirmado por Felipe).
- [x] Versión de Go en el log de build del backend: 1.25.0, la de `go.mod`.
- [x] La ruta que recibe la función llega tal cual se pidió, sin el prefijo `/api` (el handler acepta ambas).
- [x] Subida de fotos a Supabase Storage de punta a punta (confirmado por Felipe).

Pendiente de confirmar:

- [ ] `manager` no figura en "Exposed schemas" y las tres filas de la consulta del paso 1 dan `false`.
- [ ] Respaldo con `pg_dump` probado al menos una vez.
