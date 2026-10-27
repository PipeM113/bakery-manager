# Ingredientes: llave, orden, stock rápido y equivalencias

Sprint 14, a partir de las pruebas de la mamá de Felipe. Migración 28 (se aplica antes del deploy,
ver `docs/despliegue.md`).

## Reglas

- **E1. La llave es nombre + marca + proveedor**, sin distinguir mayúsculas ni espacios al borde. La misma
  mantequilla Soprole en La Vega y en Líder (a distinto precio) son dos ingredientes. Repetir los tres da
  409 y el mensaje los nombra. Renombrar sobre otro existente también es 409 (antes era un error 500).
- **E2. El proveedor es opcional**, se recorta y llega hasta 100 caracteres.
- **E3. Excel.** Importar reconoce el ingrediente por nombre + marca + proveedor; la columna se llama
  "Proveedor" y, si el archivo no la trae, el proveedor queda vacío. El archivo exportado la escribe en la
  columna F (el precio unitario, informativo, pasó a la G). Al importar, el nombre, la marca y el proveedor
  que ya existen conservan cómo estaban escritos.
- **E4. Orden alfabético** sin distinguir mayúsculas ni tildes, en ingredientes y recetas (después, marca y
  proveedor). En Ingredientes hay un selector "Ordenar por" (nombre, marca, proveedor, stock más bajo).
- **E5. Marca y proveedor al elegir productos.** Las líneas de una receta traen `brand` y `supplier`, y los
  selectores de la receta muestran "Nombre · Marca · Proveedor".
- **E6. Equivalencias.** Cada ingrediente puede tener el *Peso de 1 litro (gr)* y el *Peso de 1 unidad (gr)*
  (0 = no se sabe; el máximo es 20 000 y 100 000). Con ellos, una receta puede usar gr con un producto que se
  compra en litros, o und con uno que se compra en gramos. Los costos, la cotización (que coincide con la
  pantalla de costos), el descuento de stock de las ventas y los ajustes de stock convierten con ellas.
  **No hay densidades precargadas**: no hay una fuente oficial para los productos que compra la pastelería, y
  cada una se completa a mano.
- **E7. Unidad de una receta.** La unidad de cada línea se elige de una lista (gr, kg, ml, lt, und). Si es de
  otra familia que la del ingrediente y falta la equivalencia, la receta se rechaza (400) y el mensaje dice
  qué dato completar ("peso de 1 litro" o "peso de 1 unidad").
- **E8. Cambiar la unidad de un ingrediente.** Dentro de la familia (kg a gr) no se pregunta nada. Si el
  ingrediente no lo usa ninguna receta, cambia libre. Si una receta lo usa en otra familia y falta la
  equivalencia, responde 409 con el nombre de la receta y el dato que falta, y la pantalla abre las
  equivalencias para completarlas ahí mismo; con el dato, el cambio se guarda en la misma petición. Quitar una
  equivalencia que una receta necesita también es 409. El stock y el tamaño del paquete **no se convierten
  solos**: los edita la persona.
- **E9. Stock rápido.** El botón "+ Stock" abre una ventana con "Agregar" (motivo `compra`: suma al stock, no es
  pérdida) y "Contar" (corrección). Queda en el historial. Acepta otra unidad de la misma familia (1 kg a un
  ingrediente en gr); con la equivalencia, también de otra familia desde la API.

## Qué no hace

- No convierte el stock ni el precio al cambiar de familia: son números que se revisan a mano.
- La ventana de la receta avisa cuando falta la equivalencia, pero la completa en Ingredientes.
