# Recetario: tipos de receta, búsqueda y costo en la lista

Sprint 16. Migración 29 (se aplica antes del deploy, ver `docs/despliegue.md`).

## Tipos de receta

Cada receta puede tener **un tipo** ("Tortas bizcocho", "Hojarasca", "Merengue"...) que la dueña crea y administra.
Es opcional: sin tipo, todo funciona como antes (crear, editar, costear, vender y cotizar).

- **T1.** `POST /recipe-categories` crea un tipo: el nombre se recorta, se colapsan los espacios de adentro y tiene de 1 a 50
  caracteres. Uno repetido, sin distinguir mayúsculas ni tildes, da 409. `GET /recipe-categories` los lista en orden
  alfabético, sin distinguir mayúsculas ni tildes, con `recipe_count` (cuántas recetas tiene cada uno).
- **T2.** `PUT /recipe-categories/{id}` renombra (409 si el nombre ya existe; 404 si no existe).
- **T3.** `DELETE /recipe-categories/{id}` borra el tipo y sus recetas **quedan sin tipo**: no se borran ni se bloquea el
  borrado. La ventana lo avisa antes ("2 recetas quedarán sin tipo").
- **T4.** Las recetas aceptan `category_id`, opcional. En un `PUT`, sin el campo se conserva el tipo, con `null` se quita, y un
  id que no existe da 400. `GET /recipes` y `GET /recipes/{id}` devuelven `category_id` y `category` (el nombre, o vacío).
  "Guardar como" y "Guardar escalada" copian el tipo.
- **T5.** Una receta sin tipo se costea, se vende y se cotiza igual.
- **T6.** Migración 29: tabla `recipe_categories` (con RLS y nombre único sin distinguir mayúsculas) y `recipes.category_id`
  opcional con `ON DELETE SET NULL`.

## Costo en la lista

- **L1.** Cada receta de `GET /recipes` trae `total_cost` y `suggested_price` (pesos enteros), iguales a los de
  `GET /recipes/{id}/cost` con sus valores por defecto. Si no se puede costear (falta una equivalencia, o el rendimiento es 0)
  vienen como `null` y la lista no falla: la tarjeta dice "Sin costo".
- **L2.** La cantidad de consultas no crece con la cantidad de recetas (3 en total).

## Pantalla

- **U1.** Buscador por nombre o por tipo, sin distinguir mayúsculas ni tildes. Cada receta muestra su miniatura, "Costo total" y
  "Precio sugerido", y toda la fila abre el detalle.
- **U2.** Si alguna receta tiene tipo, la lista se agrupa con un encabezado por tipo (con contador) y "Sin tipo" al final; con dos
  o más tipos en uso aparecen chips para filtrar. Si ninguna tiene tipo, la lista es plana, sin encabezados ni chips.
- **U3.** En el formulario, "Tipo (opcional)": elige uno, "Sin tipo" o "+ Crear tipo…", que lo crea ahí mismo.
- **U4.** El botón "Tipos" abre la ventana para crear, renombrar y borrar tipos.
- **U5.** Los selectores de recetas de Costos, Ventas, Pedidos y Cotización agrupan por tipo cuando hay tipos.
- **U6.** Las ventanas nuevas caben en los cuatro anchos de referencia, con el botón principal a la vista.

## Qué no hace

No fusiona tipos ni permite varios por receta. Para cambiar el nombre de un tipo se usa "Renombrar" (cambia en todas sus recetas).
