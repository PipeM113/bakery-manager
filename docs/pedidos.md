# Pedidos con varios productos

Un pedido agrupa la venta de varios productos: "una torta con 3 postres", "un surtido de postres".
Se registra desde **Ventas → Nuevo pedido** y se ve en la pestaña **Pedidos**.

## Cómo se guarda

- Tabla `orders`: cliente (opcional), nota, fecha y `total_price`. RLS activo, sin políticas.
- **Cada línea del pedido es una venta** (`sales`) con su `order_id`. Así el stock, los costos y los
  reportes funcionan como siempre.
- `sales.line_total` es lo que esa línea realmente aportó. Para una venta suelta es
  `cantidad × precio unitario`; en un pedido con total propio es su parte del reparto. Los ingresos
  del dashboard, de la tendencia y de la rentabilidad por receta suman `line_total`.

## Precio de un pedido

- **Sin `total_price`:** el total es la suma de `cantidad × precio unitario` de cada línea.
- **Con `total_price`** (por ejemplo un descuento de combo): ese es el total, y se hace el
  **reparto** entre las líneas en proporción a su precio normal, en pesos enteros y de modo que la
  suma de las líneas sea exactamente el total. Lo que sobra al redondear hacia abajo va, de a un
  peso, a las líneas con mayor resto (la primera gana un empate). Si todas las líneas tienen
  precio normal 0, el total se reparte en partes iguales. Un total de 0 es un regalo.

## API

| Llamada | Qué hace |
|---|---|
| `POST /orders` | Crea el pedido. Cuerpo: `customer_name`, `notes`, `order_date` (hoy en Santiago por defecto, no futura), `total_price` opcional, y `lines`: de 1 a 50 con `recipe_id`, `quantity` (entero > 0), `unit_price` y `sold_as` (`recipe` por defecto, o `portion`). |
| `GET /orders?month=AAAA-MM` | Los pedidos del mes (el actual por defecto), el más reciente primero, con sus líneas. |
| `GET /orders/{id}` | Un pedido con sus líneas. 404 si no existe o no es del usuario. |
| `DELETE /orders/{id}` | **Anula el pedido completo** y devuelve al stock lo que cada línea descontó de verdad. |

- Es **todo o nada**: si una línea es inválida (receta inexistente, `sold_as` desconocido, unidades
  incompatibles, cantidad 0 o con decimales, total negativo, fecha futura…), responde 400 y no
  queda nada guardado: ni ventas ni stock descontado.
- El stock se descuenta línea por línea como en una venta (ver `docs/ventas-y-stock.md`): conversión
  de unidades, porciones, y si falta stock el pedido se registra igual, el stock queda en 0 y la
  respuesta trae `stock_shortfalls` (sumado por ingrediente, aunque dos líneas lo usen).
- Una línea de un pedido **no se puede anular sola**: `DELETE /sales/{id}` responde 409 y manda a
  anular el pedido. Si te equivocaste, anula el pedido y vuelve a ingresarlo.
- Pedidos al mismo tiempo: todos los ingredientes de todas las líneas se bloquean una vez, en un
  orden fijo, así no hay bloqueos cruzados ni stock negativo.

## Lo que no hace (todavía)

- Surtidos guardados con nombre (hoy el surtido se arma con líneas libres en cada pedido).
- Estado de pago (por cobrar / pagado).
- Editar un pedido: se anula y se vuelve a ingresar.

## Migración 22

Crea `orders` y agrega a `sales` las columnas `order_id` (opcional) y `line_total` (en las ventas
existentes se calcula como `cantidad × precio`). No borra nada.
