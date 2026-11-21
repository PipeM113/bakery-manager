# ADR-005: un ingrediente es nombre + marca; el proveedor, el tamaño del paquete y el precio pagado viven en cada compra, y el costo usa el último precio pagado

- **Estado:** Aceptado (2026-11-19, aprobado por Felipe; borrador de Claude del 2026-11-18). Reemplaza la decisión de **llave** del ADR-004; el resto del ADR-004 (equivalencias entre unidades) sigue vigente.
- **Fecha:** 2026-11-18

## Contexto

El ADR-004 (sprint 14) fijó que un ingrediente es nombre + marca + proveedor, y dejó como supuesto pendiente que cada receta usa siempre el mismo proveedor. El piloto lo refutó: la mamá de Felipe compra el mismo producto (por ejemplo, mantequilla Soprole) en el mayorista o en el súper según la oferta, y el tamaño del paquete cambia con el lugar (mayorista 25 kg, súper 1 kg). Con la llave de tres campos, ese producto son dos filas con dos stocks y dos precios, y la receta apunta a una sola.

Datos de producción (consulta de solo lectura, 2026-11-18): 71 ingredientes, 0 casos de mismo nombre + marca con más de un proveedor, y solo 2 ingredientes con proveedor escrito. Hoy no hay nada que fusionar.

Restricciones: una sola persona mantiene el sistema; los datos reales ya están cargados (migración 29 en producción); costos, ventas y la regla de "cuánto necesita una receta" (`recipeneeds`) funcionan y no deben cambiar.

## Opciones consideradas

| Opción | Costo | Riesgo | Reversibilidad | Resultado |
|---|---|---|---|---|
| A. Dejar la llave del ADR-004 (nombre + marca + proveedor) | Ninguno | El mismo producto queda en dos filas: dos stocks, y la receta mide el costo de una sola | Alta | Descartada: no resuelve lo que se observó en el piloto |
| B. Producto con ofertas por proveedor (tablas `products` y `offers`; la receta apunta al producto) | Alto: cambian recetas, ventas, stock y costos | Migración grande con datos reales | Baja | Descartada: más de lo que hace falta hoy |
| C. «B ligero»: la llave es nombre + marca; cada compra guarda proveedor, cantidad, tamaño del paquete y precio pagado; el ingrediente guarda el último precio | Medio: una migración con columnas nuevas, un índice nuevo y una fusión (hoy sin filas que fusionar) | Que el último precio sea una oferta y subestime el costo de reponer | Media: el historial de compras permite cambiar la política de costo sin perder datos | **Elegida** |

Para el costo de la receta se compararon tres políticas: último precio pagado (elegida por Felipe), el más alto de los últimos N días y el promedio ponderado. Las otras dos quedan como posibles cambios futuros sobre el mismo historial de compras.

## Decisión

- **Llave del ingrediente:** `lower(name)` + `lower(coalesce(brand, ''))`, índice único (migración 31). Repetir ambos responde 409 con un mensaje que los nombra. Marca vacía cuenta como un valor más.
- **La compra se guarda en `stock_adjustments`** con el motivo `compra` que ya existe, más tres columnas opcionales: `supplier`, `package_size` y `package_price`. Una compra lleva proveedor (puede ir vacío), cantidad, tamaño del paquete y precio pagado por ese paquete. El tamaño va en la compra porque cambia con el lugar.
- **Último precio pagado:** al registrar una compra con fecha igual o posterior a la de la última compra registrada, el ingrediente pasa a tener el tamaño de paquete, el precio del paquete, el precio por unidad y el proveedor de esa compra, y se agrega una fila a `ingredient_price_history`. Una compra con fecha anterior a la última se guarda y entra al historial, pero no cambia el precio vigente. Una merma, una corrección o un ajuste "otro" no cambian el precio.
- **El stock es una sola fila por ingrediente:** la compra suma cantidad, como hoy.
- **Costos, ventas y `recipeneeds` no cambian:** siguen leyendo `ingredients.price_per_unit`.
- **La columna `ingredients.supplier` se conserva** y pasa a significar "proveedor de la última compra" (informativo; ya no es parte de la llave).
- **Equivalencias (ADR-004):** `grams_per_liter` y `grams_per_unit` siguen en el ingrediente; ahora valen para todos sus proveedores.
- **Migración 31, fusión de duplicados** (mismo nombre + marca, distinto proveedor): conserva la fila con el precio más reciente; suma el stock; repunta `recipe_ingredients`, `sale_ingredients`, `stock_adjustments` e `ingredient_price_history` a la fila que queda; el `alert_threshold` es el mayor de los dos; las equivalencias son las de la fila que queda, o las de la otra si aquella tiene 0. **Si las filas difieren en unidad, en tipo (ingrediente/insumo) o en una equivalencia distinta de 0, la migración se detiene con un mensaje que nombra el ingrediente** y no cambia nada: lo resuelve una persona.
- **El `down` de la migración 31 no puede deshacer una fusión:** restaura el índice de tres campos y quita las columnas nuevas, pero las filas fusionadas siguen fusionadas. Antes de aplicarla en producción hay un respaldo.

## Razón

- Resuelve lo observado (un producto, varios lugares de compra) sin cambiar costos, ventas ni `recipeneeds`.
- El ingrediente guarda un precio vigente simple; la complejidad de las compras queda en una tabla que ya existe y en tres columnas.
- Detener la migración ante un conflicto evita decidir en silencio con datos reales (una densidad o una unidad equivocadas dan costos falsos que parecen exactos).
- La política de costo se puede cambiar después porque cada compra queda guardada con su precio.

## Supuestos

| Supuesto | Estado |
|---|---|
| Para costear, el último precio pagado es lo que la dueña quiere | Confirmado por Felipe (2026-11-18) |
| El tamaño del paquete cambia con el lugar de compra | Confirmado por Felipe |
| Producción no tiene duplicados por nombre + marca | Confirmado (consulta del 2026-11-18); puede cambiar antes de aplicar la migración |
| La densidad o el peso de una unidad no depende del proveedor | Inferido: pendiente de validar con la dueña |
| Las compras casi siempre se registran el mismo día o en orden | Inferido: por eso una compra de fecha anterior no mueve el precio |

## Consecuencias

- **Una compra en oferta fija el costo:** si el último precio fue de oferta, el costo de las recetas queda por debajo de lo que costará reponer. El historial permite mostrar el precio más alto reciente junto al último (idea en `docs/backlog.md`).
- **Un cambio de proveedor ya no cambia la receta ni parte el stock.**
- **El historial de ajustes de las filas fusionadas** conserva sus cifras `stock_before` y `stock_after` de origen; después de una fusión no suman en secuencia con el stock nuevo.
- **El importador de Excel** deja de crear una fila por proveedor: una fila con el mismo nombre + marca actualiza el ingrediente existente.
- **Cada compra necesita más datos** (proveedor, tamaño del paquete, precio): el modal "+ Stock" pasa a "Registrar compra" y se vuelve un poco más largo.
- Cambian las pruebas de identidad E1, que fijaban la llave de tres campos; se cambian solo con aprobación de Felipe.

## Revisar si

- La dueña pide ver el costo de una receta por proveedor o elegir el proveedor al preparar una receta (iría a la opción B).
- El último precio oscila más de, por ejemplo, 30 % entre compras seguidas y el costo de las recetas deja de ser creíble (cambiar la política: más alto reciente o promedio).
- Se registran compras fuera de orden con frecuencia.
- Aparecen ingredientes con más de tres proveedores o la dueña pide comparar precios por proveedor.
