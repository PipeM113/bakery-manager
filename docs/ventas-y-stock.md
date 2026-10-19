# Ventas y stock

Qué descuenta una venta del stock, y cómo.

## Receta completa o porción

Una venta lleva `sold_as`:

| Valor | Qué descuenta |
|---|---|
| `recipe` (por defecto) | `cantidad vendida × cantidad de la receta` |
| `portion` | `cantidad vendida × cantidad de la receta ÷ rendimiento` |

- La cantidad vendida es un número entero mayor a 0. El `unit_price` es el precio de una receta
  o de una porción, según `sold_as`; el total es `cantidad × unit_price`.
- Si la receta rinde "12 porciones", una porción es 1/12 de la receta. Si rinde otra cosa
  (por ejemplo "2 kg"), una porción es una unidad de lo que rinde (1 kg, la mitad de la receta).
- Una receta con rendimiento 0 no se puede vender por porción (400).
- Un `sold_as` distinto de `recipe` o `portion` responde 400.

## Unidades

El stock de un ingrediente se guarda en **su** unidad (`default_unit`). Antes de descontar, la
cantidad de la receta se convierte a esa unidad: kg↔gr y lt↔ml (`kernel.ConvertToBase`).

- Receta con 200 gr de un ingrediente que se guarda en kg: se descuentan 0,2 kg y el costo es
  el 20% del precio de un kg. No se descuenta un envase entero por usar una parte.
- Unidades que no se pueden convertir (por ejemplo gr con ml): la venta se rechaza con 400, el
  mensaje nombra las dos unidades y no se descuenta nada.
- `quantity_used` y `price_at_time` de la venta quedan en la unidad del ingrediente.

## Vender sin stock suficiente

La venta se registra igual (201). El stock baja hasta 0 y nunca queda negativo. La respuesta
trae `stock_shortfalls`: una lista de `{ingredient, missing, unit}` con lo que faltó (vacía si
alcanzó). El costo de la venta usa la cantidad completa.

Cada ingrediente de la venta guarda `stock_deducted`: lo que **realmente** se descontó. Anular la
venta devuelve exactamente eso, no la cantidad completa.

## Ventas simultáneas

El stock de cada ingrediente se bloquea mientras se descuenta, siempre en el mismo orden. Dos
ventas al mismo tiempo se esperan una a la otra; entre las dos descuentan como máximo el stock
que había y ninguna falla.

## Lo que no hace (todavía)

- Una venta es de **una** receta. Una torta con 3 postres se registra como varias ventas; los
  pedidos con varios productos y los combos son otro sprint.
- Ajustes de stock y merma (lo que se pierde o se corrige a mano) son otro sprint.

## Migración 20

Agrega `sales.sold_as` (por defecto `recipe`) y `sale_ingredients.stock_deducted` (en las filas
existentes se copia de `quantity_used`). Solo agrega columnas.
