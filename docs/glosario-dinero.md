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

## Escalar una receta

`GET /recipes/{id}/cost?portions=N` calcula el precio de la receta hecha para N porciones
(mínimo 1): los ingredientes se multiplican por N / rendimiento. El cobro extra y el delivery
son por pedido y no se multiplican. "Guardar escalado" crea una copia con las cantidades
escaladas y los mismos porcentajes, cobro extra y delivery; su precio coincide con el de la
vista previa.

## Pesos enteros

Todo el dinero que se guarda y se muestra (ventas, costos fijos, gastos, cotizaciones y el
dashboard) son pesos enteros, de 1 hasta 2.000.000.000. La API rechaza decimales, cero (salvo
una venta regalada, que puede valer 0), negativos y texto. El costo de los ingredientes del
mes se suma exacto y se redondea una sola vez al final. El margen de ganancia lleva un decimal.

## Categorías de gasto

Cada gasto pertenece a una categoría. Parten gas, luz, agua, arriendo, envases, transporte,
sueldos y otros, y cualquier persona puede crear, renombrar o borrar categorías (no se borra
una que tenga gastos). El nombre es único sin distinguir mayúsculas.

Lo que va dentro de la receta (ingredientes, y más adelante los insumos como cajas o moldes)
sale del costo de ventas del dashboard, así que **no se registra además como gasto**: se
contaría dos veces. Por eso no se puede crear una categoría "ingredientes". Gas, luz,
arriendo y similares, que no son líneas de receta, sí son gastos y se restan de la utilidad.

El dashboard mensual trae `expenses_by_category`: cuánto se gastó en cada categoría, de mayor a
menor.

## Insumos de receta

En la app, **ingredientes** son lo que se cocina (harina, huevos) e **insumos de receta** son lo
que acompaña al producto final y se consume por unidad: cajas de transporte, cartones, bases
para tortas. Son filas del mismo catálogo (`kind` = `supply`, los demás `ingredient`), con
precio por paquete, unidad y stock.

- Entran a la receta con una cantidad, como un ingrediente, y **escalan con las porciones**
  (una caja por torta: para 10 tortas, 10 cajas).
- Son un costo directo: el costo total es `ingredientes × (1 + indirectos + mano de obra) + insumos`.
  Los porcentajes de indirectos y mano de obra se aplican solo a los ingredientes; los insumos se
  suman tal cual, sin porcentajes. El desglose los muestra aparte (`supplies_total`).
- Una venta descuenta su stock y su costo entra al costo de ventas del dashboard
  (`total_ingredients_cost` suma ingredientes e insumos). Por eso no se registran además como
  gasto: se contaría dos veces. La categoría de gasto "Envases" queda para lo que no va dentro de
  una receta.
- Los moldes quedan fuera por ahora: dependen del tamaño y a veces ya se tienen.

## Costo fijo

Un gasto que se repite cada mes (arriendo, gas, luz, contador). Tiene un **monto por mes**: cuando
cambia, el nuevo monto vale desde el mes que se indique ("Vigente desde") y el histórico no cambia.
Terminar un costo lo saca desde ese mes en adelante pero conserva los meses anteriores. Ver
`docs/costos-fijos.md`.

## Porción

Una venta puede ser de **recetas completas** o de **porciones**. Una porción es `1 ÷ rendimiento`
de la receta: en una torta que rinde 12 porciones, 3 porciones descuentan 3/12 de cada
ingrediente. El precio unitario de una venta por porción es el precio de una porción. Ver
`docs/ventas-y-stock.md`.
