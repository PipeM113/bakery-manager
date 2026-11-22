# Revisión de arquitectura y decisiones (2026-11-01)

Revisión del manager en el estado de `main` después del sprint 14, hecha con las skills `disenar-arquitectura` (modo revisión) y `revisar-contra-adr`. **Es un análisis: no cambia código.** Escala: sistema completo, pero solo lectura y una sonda local con datos ficticios; no se probó nada contra producción.

## Cómo se hizo y qué no cubre

- Se leyeron los ADR-001 y ADR-002, `docs/`, la estructura de `backend/internal`, `app.go`, los repositorios y servicios que tocamos en los sprints 11 a 14, el PDF, la configuración y el CI.
- Un hallazgo se confirmó con una sonda local (test temporal, ya borrado): ver H2.
- **No hay `CLAUDE.md` ni `.claude/rules/` en el repo**: las únicas reglas son los dos ADR y los documentos de `docs/`. Lo que aquí se marca como "defecto" viola algo escrito; el resto es `sugerencia`.
- No se hizo auditoría de seguridad completa (opción B del plan), no se revisó el sitio público ni se probó la app desplegada.
- La revisión independiente con un subagente de contexto limpio (que pide la skill para sistemas completos) **no se hizo**: lo que sigue es mi lectura, no una refutación externa.

## Lo que está bien (y conviene no tocar)

- **Decisiones con evidencia:** dinero en pesos enteros con fracciones exactas y un solo redondeo (`docs/glosario-dinero.md`), el reloj inyectable con "hoy" de Santiago y un test que prohíbe `time.Now()` suelto (`docs/tiempo.md`), RLS sin políticas con `manager_app` como dueño. Cada una tiene su documento y su test.
- **Método:** reglas de aceptación, tests en rojo, aprobación, implementación. Los 316 tests de Go (con Postgres real) y la suite responsive atrapan regresiones reales; en S14 detectaron un choque con el test de humo.
- **Un solo `app.New`** para el servidor local y Vercel: no hay dos versiones de las rutas.

## Hallazgos (ordenados por impacto)

| # | Tipo | Severidad | Qué | Evidencia | Por qué importa | Pregunta para Felipe |
|---|---|---|---|---|---|---|
| H1 | defecto | Alta | El piloto con datos reales empieza **sin respaldo probado**. | `docs/fuentes.md:13`: Supabase Free "sin backups automáticos" y "se pausa tras 1 semana de inactividad". `docs/despliegue.md:158` tiene el comando de `pg_dump`, pero `despliegue.md:176` y `docs/backlog.md` ("Pendiente") dejan "respaldo probado al menos una vez" sin marcar. ADR-001, riesgo 1, lo fijó como mitigación. | Si se pierde o se corrompe la base, se pierde lo que cargó tu mamá. La mitigación es barata y está sin dueño ni fecha. | ¿Hacemos hoy un `pg_dump`, lo restauramos en una base local para probarlo y lo dejamos programado (por ejemplo, semanal, guardado fuera de Supabase)? |
| H2 | defecto | Media | Borrar un ingrediente que usa una receta responde **500 "error eliminando insumo"**. | Sonda local: `DELETE /ingredients/{id}` con el ingrediente en la receta "Pan" → `500 {"error":"error eliminando insumo"}`. `handler.go` (Delete) no distingue el error de clave foránea; `recipe_ingredients.ingredient_id` es `ON DELETE RESTRICT` (migración 4). | Es el mismo patrón que corregimos en S14 con el nombre repetido (500 en vez de 409). Tu mamá vería un error genérico y el log se llenaría de "errores" que son una regla de negocio. | ¿Lo corregimos con un 409 que nombre las recetas que lo usan? **Incierto:** falta mirar si borrar una receta con ventas o cotizaciones tiene el mismo problema. |
| H3 | defecto | Media | La regla "cuánto necesita cada ingrediente una receta, en la unidad del ingrediente" está **duplicada**. | `costs/service/service.go` (`LoadRecipe`) y `sales/repository/repository.go` (`needsOf`) hacen la misma consulta (`recipe_ingredients` unido a `ingredients`) y la misma conversión con `kernel.Convert`. En S14 tuve que tocar las dos, más `stock_adjustments`, para que las equivalencias valieran en todas. | Un tercer consumidor, o un cambio de la regla, se puede olvidar en uno de los dos. Hoy los tests cruzados (E6) lo cubren, pero dependen de que alguien los escriba. | ¿Movemos la regla a una sola función del dominio (en `shared/kernel` o `costs/domain`) que ambos usen? |
| H4 | defecto | Media | **No hay tests de comportamiento del frontend**. | Los 22 archivos de `frontend/tests` leen texto fuente o importan utilidades; ninguno monta un componente. El comportamiento de pantallas lo cubre solo la suite Playwright con una API simulada (`mock-api.mjs`), que mide ancho, tamaños y que los textos existan. | En S14 verifiqué con expresiones regulares que existan "Equivalencias" y el aviso de la receta. **Nadie ha ejecutado de verdad** el flujo "cambiar de familia → 409 → se abre Equivalencias → guardar", ni el aviso de la receta ni "+ Stock" guardando. Pasar `npm test` no prueba que funcionen. | ¿Agregamos 3 o 4 pruebas e2e de flujos (stock rápido, cambio de unidad con 409, receta con unidad de otra familia) contra la API simulada? Mientras tanto, esos flujos son parte de lo que tienes que aceptar a mano. |
| H5 | sugerencia | Media | **El diseño de datos liga la receta a un proveedor concreto.** Cada combinación nombre + marca + proveedor es una fila de `ingredients`, y la receta apunta a esa fila. | ADR-004 propuesto; migración 28 (`ingredients_identity`). Una receta de galletas usa "Mantequilla · Soprole · La Vega"; si compras ese día en Líder, hay que cambiar la línea de la receta o cargar el stock en la otra fila. | Resuelve lo que pidió tu mamá (mismo producto, distinto precio), pero **cambia el stock y el costo de una receta según dónde compre**. El modelo normalizado (producto → ofertas por proveedor) lo evita, pero cuesta una migración grande. | ¿Tu mamá pide cambiar de proveedor dentro de una receta, o cada receta siempre usa el mismo? Si es lo primero, revisamos ADR-004 antes del piloto largo. **Resuelto el 2026-11-19:** compra el mismo producto en distintos lugares; el ADR-005 reemplazó la llave (nombre + marca) y el proveedor pasó a cada compra. |
| H6 | sugerencia | Media | **Las capas no tienen una regla.** Unos paquetes siguen `handler → service → repository`; otros van `handler → repository`; algunos servicios llevan SQL. | `ingredients`, `recipes`, `stock_adjustments`, `quotations` y `business`: el handler llama al repositorio. SQL en servicios: `auth/service`, `costs/service`, `analytics/service`, `ingredients/service/import.go`. Reglas de negocio dentro de repositorios: `ingredients/repository` (`checkRecipes`, la corrección de stock) y `sales/repository` (`needsOf`). | Para dos usuarios y un desarrollador, la mezcla funciona. El costo aparece al decidir **dónde va** una regla nueva (H3 es una consecuencia). | ¿Escribimos la regla mínima ("las reglas viven en domain/service; el repositorio solo persiste") en un ADR y la aplicamos **solo** a lo que se toque, sin reescribir lo demás? |
| H7 | sugerencia | Baja | El conocimiento de **unidades** está repetido: Go (`kernel`, `ingredients/handler`, `ingredients/service/import.go`) y frontend (`utils/units.ts`, `UNITS_OF` en dos modales, `PriceDisplay`). | `grep` de las listas `"gr"/"kg"/"ml"/"lt"/"und"`. `QuickStockModal` copia `UNITS_OF` de `StockAdjustmentModal`: es duplicación que agregué yo en S14. | Si aparece una unidad nueva o cambia una regla, hay siete sitios donde olvidarse. Además `normalize` distingue mayúsculas y un test (`CURRENT`) lo fija. | ¿Lo dejamos hasta que haga falta otra unidad? Recomiendo unificar solo los dos `UNITS_OF` con `unitsFor` cuando se toquen esas pantallas. |
| H8 | sugerencia | Baja | **Sesión:** el token vive en `localStorage` 24 horas, no se revoca y el rol del token no se usa. | `frontend/src/services/api.ts:9`, `auth.go:143`, `pkg/middleware/auth.go:49-54`. Un usuario borrado conserva su token hasta que expire. ADR-001 (riesgo 3) y `docs/backlog.md` ya lo registran. | Aceptado hasta la etapa 1 (Supabase Auth). Con dos cuentas de confianza, el riesgo real es un celular perdido. | Nada que decidir ahora: se reemplaza en la etapa 1. |
| H9 | sugerencia | Baja | **PDF: descarga una URL guardada en la receta, sin lista de servidores y siguiendo redirecciones.** | `quotations/service/pdf.go:371-374`. La URL llega en `photo_url` del cuerpo de `POST /recipes` (cualquier cuenta autenticada). | Los actores son solo las dos cuentas de confianza, así que no cruza una frontera real: es endurecimiento. Ya está en el backlog. | ¿Restringimos a los hosts de Supabase Storage? Es un cambio pequeño; puede ir junto a H2. |
| H10 | sugerencia | Baja | **Migraciones a mano y dueño de los objetos.** | `docs/despliegue.md:62`: seis migraciones aplicadas a mano dejaron tablas de `postgres` y el login dio 500 (2026-10-19). El CI no aplica migraciones a Supabase. | Ya se corrigió y se documentó. Sigue dependiendo de que quien aplique recuerde `OWNER TO manager_app`. | Para S14 usamos la consulta de dueños y salió limpia. Un script único que aplique y verifique evitaría depender de la memoria; no urge. |

## Decisiones que hoy solo viven en `docs/` o en la conversación

Se redactaron como **propuestos** (no son restricciones hasta que los apruebes):

- `docs/adr/003-datos-del-negocio.md`: marca fija en el código, cinco campos editables en la base y RUT y cuenta en variables de entorno.
- `docs/adr/004-identidad-de-ingredientes-y-equivalencias.md`: llave nombre + marca + proveedor y equivalencias entre unidades sin densidades precargadas (incluye el riesgo de H5).

Pendientes de ADR, si quieres formalizarlos más adelante: reloj inyectable y hora de Santiago (hoy `docs/tiempo.md`), dinero en pesos enteros (hoy `docs/glosario-dinero.md`), y toda variación de stock deja un registro (hoy `docs/ajustes-de-stock.md`).

## Veredicto

**Bloquea una cosa, y no es de código:** H1 (el respaldo) antes de seguir cargando datos reales. Lo demás no bloquea: H2 y H3 son los siguientes arreglos con mejor relación costo-beneficio; H4 y H5 conviene decidirlos antes de dar el piloto por aceptado.

## Siguiente paso propuesto

1. Hoy: respaldo con `pg_dump`, restauración de prueba y programación (H1).
2. Un sprint pequeño (15): H2 + H9 (409 al borrar y lista de hosts de fotos), H3 (una sola regla de "necesidades de una receta"), y las pruebas e2e de flujos de H4.
3. Decidir H5 con lo que veas en el piloto, y si hay que cambiarlo, hacerlo antes de que haya muchos datos.

## Revisión del 2026-11-22 (diseño, arquitectura y buenas prácticas, tras los sprints 17 a 19)

Se revisó la interfaz (con `direccion-visual` y `make-interfaces-feel-better`), la arquitectura contra los ADR aceptados (001, 002, 003, 005 y las equivalencias del 004) y las buenas prácticas del backend y del frontend. Lo que cumple, y lo que se corrigió en el mismo PR, y lo que queda anotado.

**Cumple (con evidencia)**
- Las consultas con texto concatenado solo concatenan listas de columnas constantes (`orderColumns`, `quotationColumns`, `recipeColumns`, `expenseColumns`); los valores van siempre como parámetros.
- Los clientes HTTP salientes tienen timeout (`storage/supabase.go`, 20 s; descarga de fotos del PDF, 5 s) y las fotos del PDF solo salen del bucket del proyecto.
- Los cuerpos JSON tienen tope global (`mid.MaxBody`) y las cargas grandes, el suyo.
- Las listas de cotizaciones, líneas y recetas cargan sus detalles en una sola consulta (sin N+1).
- Las dependencias apuntan hacia adentro: el costo, las ventas y las cotizaciones usan `recipeneeds` y `kernel.Convert` en vez de copiar la regla.
- Paleta, tipografía y contraste siguen el manual de marca y están protegidos por `brand.test.mjs`.

**Corregido en este PR**
| Hallazgo | Corrección |
|---|---|
| 125 usos de `transition-all` (animan propiedades que no cambian y pueden saltar en pantallas lentas) | Ahora nombran lo que cambia (`transition-colors`, o color y opacidad donde el control cambia de opacidad) |
| `console.log` con los gastos y sus montos en la consola del navegador (`ExpensesPage`) | Quitados; quedan solo los `console.error` |
| Sin enlace para saltar la navegación; los dos `<nav>` sin nombre | «Saltar al contenido» (visible al enfocar con el teclado) y `aria-label="Navegación principal"` |
| Titulares con líneas desparejas, párrafos con palabras sueltas, cifras que no se alinean en columnas, fotos que se funden con el fondo crema | `text-wrap: balance` y `pretty`, cifras tabulares en tablas y cifras grandes, y un contorno fino en las fotos |
| `docs/despliegue.md` decía Go 1.25.0 y `go.mod` pide 1.26.0 | Corregido (lo histórico de `docs/fuentes.md` queda como observación fechada) |

**Anotado para después (no se hizo aquí)** — ver `docs/backlog.md`
- ~~**Sin identificador de solicitud en los registros**~~ **Resuelto el 2026-11-22:** `pkg/middleware/requestid.go` pone un identificador propio (el del cliente se ignora) en el encabezado `X-Request-Id` y en el contexto que imprime el `Logger` de chi.
- ~~**Listas sin tope ni paginación**~~ **Resuelto el 2026-11-22 para las que crecen:** cotizaciones y ventas se leen en páginas (`limit` 1 a 1000, 200 por defecto; `offset`; `X-Has-More`). Los gastos ya se leen por mes; ingredientes y recetas son catálogos de decenas de filas. La pantalla de Ventas avisa cuando una lista llega a 200.
- ~~**Título del navegador igual en todas las pantallas**~~ **Resuelto el 2026-11-22** (`utils/titles.ts`).
- ~~**8 avisos del lint**~~ **Resuelto el 2026-11-22:** seis corregidos y dos intencionales documentados; el lint queda sin avisos.
- ~~**El CI no revisaba formato ni secretos**~~ **Resuelto el 2026-11-22:** paso `gofmt` en el job del backend y job `repo-checks` (búsqueda de secretos).
- **La captura de la pantalla que más se usa a 360 px** sigue sin pasar por una persona real: la prueba con alguien que no conozca el sistema (paso 6.7 de `direccion-visual`) está pendiente.

**Veredicto:** no hay defectos de severidad alta. La deuda es de observabilidad y de crecimiento de datos, no de corrección ni de seguridad.
