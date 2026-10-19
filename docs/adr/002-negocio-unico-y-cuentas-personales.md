# ADR-002: el negocio es uno solo y cada persona tiene su cuenta

- **Estado:** Aceptado (Felipe, 2026-10-19; el borrador lo redactó Claude)
- **Fecha:** 2026-10-19

## Contexto

La usan dos personas, ambas administradoras: Felipe y su mamá. Hasta el sprint 10 cada registro
llevaba un `user_id` y casi todo se filtraba por él: con dos cuentas, cada una habría visto solo su
mitad del negocio (ventas, gastos, costos fijos, cotizaciones, pedidos, ajustes de stock y el
dashboard), y las recetas solo las habría podido editar quien las creó. Además, al borrar un
usuario se borraban en cascada todos sus registros.

## Opciones consideradas

| Opción | Efecto | Problema |
|---|---|---|
| A. Una sola cuenta compartida | Nada que cambiar | Se comparte una contraseña; no queda claro quién hizo qué; no se puede quitar el acceso a una persona |
| B. Filtrar por usuario (lo que había) | Cada cuenta ve lo suyo | Dos mitades de un mismo negocio, que es justo lo que no se quiere |
| C. Un negocio, cuentas personales | Ambas ven y administran todo; el registro recuerda quién lo creó | Hay que migrar la columna y quitar los filtros |

## Decisión

Opción C.

- La columna `user_id` pasa a llamarse **`created_by`** en `recipes`, `fixed_costs`, `sales`,
  `orders`, `operational_expenses`, `quotations` y `stock_adjustments`. Solo dice quién lo
  registró; no da permisos.
- Si se borra un usuario, **sus registros se quedan**: `created_by` queda vacío (`ON DELETE SET
  NULL`). Antes se borraban en cascada.
- Todo se lee del negocio completo y cualquiera de las dos cuentas puede editar, anular o borrar
  lo del otro. Los ingredientes ya eran compartidos.
- El rol (`owner` / `helper`) no cambia lo que se puede hacer: las dos personas son `owner`. Si algún
  día se necesita una persona con permisos reducidos, será una decisión nueva.
- Un token que no trae `sub` y `role` como texto es un 401, no un error del servidor.

## Consecuencias

- La API deja de decir `user_id` y dice `created_by` (que puede ser vacío).
- La pantalla no muestra quién registró cada cosa (decisión del propietario: da igual quién lo hizo);
  el dato queda guardado por si hace falta.
- Crear la cuenta de la segunda persona y cambiar una contraseña se hace desde la base, con
  `cmd/hashpassword` (ver `docs/despliegue.md`). Cambiar la propia contraseña desde la app queda
  para un sprint de calidad.
- Etapa 1 del plan (identidad única con Google) reemplazará el login propio sin tocar esto.
