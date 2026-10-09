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
- Verifica: `select version, dirty from manager.schema_migrations;` debe dar `15` y `false`.
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
2. Framework: "Other". No cambies el comando de build.
3. El archivo `backend/vercel.json` envía todas las rutas a `api/index.go`.

## 5. Variables de entorno

En el panel del proyecto del backend (Settings, Environment Variables). Nada de esto va en archivos del repo:

| Variable | Valor |
|---|---|
| `DATABASE_URL` | Cadena del **pooler de transacciones** (6543) con el usuario `manager_app.<project_ref>`: `postgresql://manager_app.<project_ref>:<contraseña>@<host-del-pooler>:6543/postgres` |
| `JWT_SECRET` | Aleatorio, de al menos 32 caracteres. Genéralo con `openssl rand -base64 48` y pégalo directo en el panel. |
| `CORS_ORIGINS` | La URL exacta del frontend, sin `/` final y sin `*`, por ejemplo `https://<tu-frontend>.vercel.app`. Varias, separadas por coma. |
| `DB_MAX_CONNS` | `4` (valor por defecto). Cada instancia abre como máximo ese número de conexiones. |
| `CLOUDINARY_URL` | Opcional. Sin ella la subida de fotos queda desactivada. |

En el proyecto del **frontend** agrega `VITE_API_URL` con la URL pública del backend, `https://<tu-backend>.vercel.app`. El build del frontend falla si falta o no es una URL válida. Es una variable de build: cambiarla exige un nuevo despliegue.

Despliega los dos proyectos.

## 6. Verificación

1. En el log de build del backend busca la línea `go version` / versión de Go que usó Vercel y anótala en `docs/fuentes.md` (hoy figura como no verificada).
2. Abre `https://<tu-backend>.vercel.app/health`: debe responder `200`.
3. Abre el frontend, inicia sesión con el usuario del paso 3 y crea un insumo.
4. Si `/health` responde `500` con "configuración del servidor inválida", falta o está mal una variable (el log de la función dice cuál, sin mostrar valores).
5. Si el login falla con error de conexión, revisa que `DATABASE_URL` use el puerto 6543 y el usuario con `.<project_ref>`.

## 7. Respaldo manual

El plan Free de Supabase **no hace respaldos automáticos** y pausa el proyecto tras una semana sin actividad. Antes de cambios riesgosos y cada tanto, con el pooler de sesión:

```sh
pg_dump "$MIGRATE_URL" --schema=manager --no-owner --no-privileges -Fc -f manager-$(date +%F).dump
```

Guarda el archivo fuera del repo (contiene los datos del negocio y los hashes de contraseñas). Para restaurar en un proyecto vacío: ejecuta el paso 1 y `pg_restore --no-owner -d "$MIGRATE_URL" manager-<fecha>.dump`.

## Verificación en el mundo real (pendiente)

Lo siguiente solo se puede confirmar contra Supabase y Vercel reales; el ensayo local no lo cubre. Anota el resultado en `docs/fuentes.md` cuando lo hagas:

- [ ] Las migraciones corren por el pooler de sesión con el rol `manager_app`.
- [ ] El login funciona por el pooler de transacciones.
- [ ] Versión de Go en el log de build del backend.
- [ ] La ruta que recibe la función (con o sin prefijo `/api`); el handler acepta ambas.
- [ ] `manager` no figura en "Exposed schemas" y las tres filas de la consulta del paso 1 dan `false`.
