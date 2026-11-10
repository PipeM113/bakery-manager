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

## Eliminar (sprint 15)

Un ingrediente o una receta con historia no se borra: la API responde **409** con el motivo y la pantalla
(Ingredientes y Recetas) lo muestra en un aviso. Antes respondía un 500 genérico y el botón "Eliminar"
parecía roto.

- **D1.** Un ingrediente que usan recetas: "No se puede eliminar: se usa en las recetas Alfajor, Brownie,
  Cheesecake y 2 más" (hasta tres nombres, en orden alfabético).
- **D2.** Un ingrediente que ya no está en ninguna receta pero aparece en el historial de ventas: "aparece en
  el historial de N ventas".
- **D3.** Una receta con ventas o cotizaciones: "tiene 2 ventas y 1 cotización registradas". Su foto no se borra
  del bucket. Los pedidos cuentan como ventas.
- **D4.** Lo que no tiene uso se sigue borrando (204), con la foto de la receta.
- **D5.** El motivo aparece en la pantalla (`role="alert"`), con una ×  para cerrarlo.

Quien quiera retirar una receta con ventas no puede hacerlo hoy: archivarla (esconderla sin borrar el
historial) queda como idea en `docs/backlog.md`.

## Fotos de las recetas (sprint 15)

- **P1.** `POST /recipes` rechaza `photo_url` (400): la foto entra solo por la subida
  (`POST /recipes/{id}/photo`). "Guardar como" y "Guardar escalada" siguen copiando la foto de la receta original.
- **P2.** El PDF de una cotización solo descarga fotos cuya dirección empieza con la dirección pública del
  bucket del proyecto (`<SUPABASE_URL>/storage/v1/object/public/<bucket>/`) y no sigue redirecciones. Sin
  almacenamiento configurado no descarga ninguna. Una foto que no cumple se omite: la línea sale sin foto y el
  PDF se genera igual.

## Una sola regla para "cuánto necesita una receta" (sprint 15)

N1. `internal/recipeneeds` lee las líneas de una receta y las convierte a la unidad de cada ingrediente con sus
equivalencias. Costos (y, por lo tanto, cotizaciones) y ventas la usan; antes cada uno tenía su copia. Un test de
higiene (`internal/hygiene/recipeneeds_s15_test.go`) limita desde qué archivos se llama a `kernel.Convert`.
