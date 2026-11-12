# ADR-004: un ingrediente es nombre + marca + proveedor, y las equivalencias entre unidades las pone la dueña

- **Estado:** Propuesto (borrador de Claude, 2026-11-01; recoge lo decidido por Felipe en el sprint 14)
- **Fecha:** 2026-11-01

## Contexto

La mamá de Felipe compra el mismo producto (por ejemplo, mantequilla Soprole) en lugares distintos y a precios distintos, y la leche o el aceite en litros mientras la receta pide gramos. Hasta el sprint 13 el nombre era único y la unidad no se podía cruzar entre familias (peso, volumen, unidad).

## Opciones consideradas

| Opción | Efecto | Problema |
|---|---|---|
| A. Nombre único (lo que había) | Simple | No guarda el mismo producto con dos precios |
| B. Producto con varias ofertas (una tabla de productos y otra de ofertas por proveedor) | La receta apunta al producto y el costo sale de la oferta elegida | Migración grande: recetas, ventas, stock y costos cambian; más pantallas |
| C. La llave es nombre + marca + proveedor (sin distinguir mayúsculas) | Pocos cambios; cubre lo pedido | La receta queda ligada a un proveedor concreto (ver Consecuencias) |

Para las unidades: (1) precargar densidades, (2) pedir el peso de 1 litro y de 1 unidad a la dueña, (3) prohibir el cruce.

## Decisión

- **Llave:** opción C. Índice único sobre `lower(name)`, `lower(brand)`, `lower(supplier)` (migración 28). Repetir los tres responde 409 con un mensaje que los nombra.
- **Equivalencias:** opción (2). Cada ingrediente puede tener `grams_per_liter` y `grams_per_unit` (0 = no se sabe; máximos 20 000 y 100 000 gr). Con ellos, `kernel.Convert` pasa entre gr, ml y und; sin ellos, el cruce se rechaza diciendo cuál dato falta.
- **No hay densidades precargadas:** no existe una fuente oficial para los productos que compra la pastelería (`docs/fuentes.md` exige fuente para datos de terceros).
- **Se pide solo cuando hace falta:** al cambiar de familia la unidad de un ingrediente que una receta usa, la API responde 409 con el nombre de la receta y el dato que falta; si ninguna receta lo usa, el cambio es libre. Quitar una equivalencia que una receta necesita también es 409.
- **No se convierten solos** el stock ni el tamaño del paquete al cambiar de familia: los edita la persona.

## Razón

- Es lo mínimo que resuelve lo que pidió la dueña, sin la migración de B.
- No inventar densidades evita costos falsos que parecen exactos.
- Preguntar solo cuando una receta lo necesita no bloquea el trabajo diario (decisión de Felipe).

## Supuestos

| Supuesto | Estado |
|---|---|
| Cada receta usa siempre el mismo producto de un proveedor | **Pendiente**: validarlo con el piloto (ver Consecuencias) |
| La dueña sabe o puede averiguar el peso de 1 litro y de 1 unidad | Pendiente |
| Marca vacía y proveedor vacío cuentan como un valor más de la llave | Confirmado (tests E1) |

## Consecuencias

- **Una receta apunta a una fila concreta.** Si un día compra la misma mantequilla en otro lugar, el stock y el costo están en otra fila y hay que cambiar la línea de la receta o repartir el stock. Con B no pasaría.
- Cada módulo que convierte unidades debe usar `kernel.Convert` (hoy: costos, ventas, ajustes de stock y validación de recetas). La regla "cuánto necesita una receta" sigue duplicada en costos y ventas (ver `docs/revision-arquitectura-2026-11.md`, H3).
- Un cambio posterior a B sería una migración grande con datos reales.

## Revisar si

- La dueña pide elegir el proveedor al preparar una receta, o ver el costo de una receta con cada proveedor.
- Aparece una fuente oficial de densidades que valga la pena precargar.
- Hay más de unas decenas de ingredientes repetidos por proveedor.
