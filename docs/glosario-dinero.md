# Glosario del dinero

Qué significa cada palabra de precios, costos y utilidad en la app y en el código. Si una
pantalla o una respuesta de la API usa estas palabras de otra forma, es un error.

## Precio de una receta

| Palabra en pantalla | Nombre en la API / base | Qué es |
|---|---|---|
| **Recargo sobre costo** | `margin_pct` | Fracción que se suma al costo: 0.30 significa costo × 1.30. Se guarda por receta. Una receta nueva parte en 30%. |
| Costos indirectos (%) | `indirect_cost_pct` | Fracción del costo de ingredientes que se suma por gas, luz, moldes, etc. Por receta; nueva parte en 30%. |
| Mano de obra (%) | `labor_cost_pct` | Fracción del costo de ingredientes que se suma por trabajo. Es una **estimación para fijar precio**, no un sueldo pagado: la dueña se paga con su utilidad. Por receta; nueva parte en 30%. |
| Cobro extra | `extra_charge` | Pesos fijos que se suman al precio sin mostrarse en la cotización. |
| Delivery | `delivery_cost` | Pesos fijos que se suman al precio y se muestran en la cotización. |
| Precio sugerido | `suggested_price` | `(costo total × (1 + recargo)) + cobro extra + delivery`, subido al siguiente múltiplo de $500. |

Los nombres `margin_pct` en la API y en la base **no cambian** (hay cotizaciones guardadas con
ese nombre). En pantalla y en el código nuevo se dice *recargo sobre costo*.

## Dashboard

| Palabra | Qué es |
|---|---|
| **Margen de ganancia** | Utilidad dividida por ingresos, en porcentaje (utilidad / ventas × 100). No tiene relación con el recargo de una receta: 30% de recargo no es 30% de margen. |
| Utilidad | Ingresos − costo de ingredientes de lo vendido − costos fijos − gastos. |

Los porcentajes de indirectos y mano de obra de una receta ayudan a fijar el precio; la
utilidad se calcula con costos reales (gastos y costos fijos), no con esos porcentajes.

## Redondeo

- Los cálculos de precio usan fracciones exactas; no hay errores de coma flotante. El redondeo es lo único que quita decimales.
- Se redondea una sola vez, al final, y nunca se calcula con números ya redondeados.
- Los totales que se muestran (ingredientes, indirectos, mano de obra y costo total) van en
  pesos enteros; la mitad sube (0,5 → 1).
- El precio sugerido sube al siguiente múltiplo de $500.
- El costo por porción y la base se muestran sin redondear en la API; la pantalla los formatea.

## Números en el Excel de ingredientes

Se leen como se escriben en Chile: el punto agrupa miles y la coma separa decimales
(`1.500` = mil quinientos, `1.500,50`, `2,5`), y puede llevar `$` delante. Un `2.5` suelto es
2,5. Una fila con tamaño o precio inválido, vacío o menor o igual a 0 se rechaza con su
motivo; las demás filas se importan.
