# Cotizaciones

Una cotización es lo que se le entrega a un cliente antes de que confirme: uno o varios productos, un
delivery y un PDF con la marca. Es de la tienda, no de una cuenta: las dos cuentas ven, crean,
confirman, cancelan y descargan todo (`docs/acceso.md`).

## Qué lleva

- **Cliente** (hasta 100 caracteres) y **de 1 a 30 líneas**.
- Cada **línea** es una receta con una cantidad entera (1 a 100.000) y un modo: `recipe` (recetas
  completas, por defecto) o `portion` (porciones, igual que en las ventas). Se puede ajustar por línea el
  margen, los costos indirectos, la mano de obra y el cobro extra, sin tocar la receta. Lo que no se
  manda sale de la receta (el cobro extra, en cambio, vale 0 si no se manda, igual que en el endpoint de
  costos).
- **Un solo delivery** por cotización (pesos enteros, 0 si no hay). No va dentro de las líneas.
- **Total** = suma de las líneas + delivery, en pesos enteros.

## Cómo se calcula el precio de una línea

Con el mismo cálculo de la pantalla de costos (ver `docs/glosario-dinero.md`), redondeado hacia arriba a
$500:

- `recipe`: precio de una receta completa × cantidad. El cobro extra queda dentro del precio, no se
  muestra aparte.
- `portion`: precio de la receta hecha para esa cantidad de porciones (los ingredientes se escalan). El
  precio unitario que se muestra es ese total dividido por las porciones, redondeado al peso.

## Se guarda todo, y el PDF sale de lo guardado

- `POST /quotations` valida y calcula todas las líneas y guarda la cotización entera, o nada. Responde
  JSON (201), no el PDF.
- Cada línea guarda una **copia** del nombre de la receta, su foto y sus precios: si después cambian los
  ingredientes o la receta, la cotización guardada no cambia.
- `GET /quotations/{id}/pdf` arma el PDF de lo guardado, las veces que haga falta, desde la lista. La fecha
  del PDF y del archivo es la de la cotización (día de Santiago): `cotizacion_<cliente>_<fecha>.pdf`.
- Otras rutas: `GET /quotations?status=`, `GET /quotations/{id}`, `PUT /quotations/{id}/confirm` y
  `/cancel`. Confirmar o cancelar solo cambia el estado; no crea pedido ni toca el stock (el pedido se
  ingresa a mano con los precios cotizados).
- `POST /quotations/generate` (el PDF de una sola receta en un paso) ya no existe.

## El PDF

Logo en color, cliente y fecha, una fila por línea (foto pequeña si la receta tiene, nombre, cantidad,
precio unitario y subtotal), delivery si hay, total, datos de transferencia, contacto y el aviso de 48
horas. Si las filas no caben pasa a otra página; el total y los datos van juntos en la última.

- Colores `crema` (fondo), `oro` y `negro`; tipografías Playfair Display y DM Sans, incluidas en el
  servidor (`backend/internal/quotations/service/assets/`; fuente y licencia en `docs/fuentes.md`).
- Las fotos pueden ser JPEG, PNG o WebP (el tipo se lee de los bytes, no de la dirección). Cada foto se
  descarga con un máximo de 5 MB y 5 segundos, todas a la vez; una foto que no se puede descargar o leer
  deja esa fila sin foto, nunca detiene el PDF. La misma foto en dos filas se incluye una vez.

## Datos del negocio

El PDF imprime tres tipos de datos del negocio:

- **De la marca, fijos en el código** (`backend/internal/quotations/service/pdf.go`): el nombre, **Angeles Bakery**,
  y el Instagram, **@angelescoffee_bakery**, que en el PDF es un enlace a
  `https://www.instagram.com/angelescoffee_bakery`. No se editan ni pueden faltar. Si cambian, se cambian en el
  código.
- **Editables en la app**, en **Datos del negocio** (menú "Más" en el celular, barra lateral en el computador).
  Las dos cuentas pueden editarlos y el cambio vale desde el siguiente PDF, también para cotizaciones antiguas
  (son datos del negocio, no de la cotización).
- **En Vercel**: el RUT y el número de cuenta. Nunca llegan a la pantalla ni a la base.

| Dato | Dónde | Notas |
|---|---|---|
| Teléfono | App | Números, espacios, `+`, `-` y paréntesis; al menos 8 dígitos |
| Banco | App | Hasta 60 caracteres |
| Tipo de cuenta | App | Cuenta Corriente, Cuenta Vista, Cuenta RUT o Cuenta de Ahorro |
| Titular de la cuenta | App | Hasta 100 caracteres |
| Correo | App | Se guarda en minúsculas |
| **RUT** | **Variable `BIZ_RUT` en Vercel** | Nunca llega a la pantalla ni a la base |
| **Número de cuenta** | **Variable `BIZ_ACCOUNT` en Vercel** | Igual |

`PUT /business` solo acepta esos cinco campos: **uno que no existe (por ejemplo `name` o `instagram`) se rechaza con
400**, no se ignora. Las variables de Vercel las lee el servidor al arrancar: después de crearlas o cambiarlas hay
que **volver a desplegar**.

Ningún dato viene de relleno: la ficha parte vacía. Mientras falte algo, el PDF responde 503 y el mensaje dice qué
falta y dónde completarlo (los campos de la app o las variables de Vercel), y la pantalla de cotizaciones muestra un
aviso con un enlace a "Datos del negocio". Crear cotizaciones funciona igual.

## Migraciones

- **24:** crea `quotation_lines` (con RLS y sin políticas, como las demás), agrega
  `quotations.delivery_cost` y hace opcionales `recipe_id` y `margin_pct` de `quotations`. Cada cotización
  que ya existía pasa a tener una línea (cantidad 1, su receta, su total); su total conserva el significado.
  La bajada devuelve la receta de la primera línea y pierde las demás líneas.
- **25:** crea `business_profile`, una sola fila que nace vacía.
- **27:** elimina las columnas `name` e `instagram` de `business_profile` (son de la marca y viven en el código). Los valores que tuvieran se pierden; la bajada las devuelve vacías.

## Lo que no hace (todavía)

- Crear el pedido al confirmar la cotización.
- Editar una cotización guardada: se crea otra.
- Aplicar a la cotización los datos del negocio que tenía el día en que se hizo.
