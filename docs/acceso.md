# Acceso y cuentas

El manager es **un solo negocio** con **cuentas personales** (ADR-002). Felipe y su mamá entran
cada uno con su correo y su contraseña, y ven y administran exactamente lo mismo.

## Qué significa "negocio único"

- Ventas, pedidos, gastos, costos fijos, cotizaciones, ajustes de stock, recetas, ingredientes, el
  dashboard, la tendencia y la rentabilidad cuentan lo que registró **cualquiera** de las dos
  cuentas. Las dos ven los mismos números.
- Cualquiera de las dos puede editar, anular o borrar lo que registró la otra, incluidas las
  recetas.
- Cada registro recuerda quién lo creó en `created_by` (no hay `user_id`). La pantalla no lo
  muestra. Si ese usuario se borra, **los registros se quedan** y `created_by` queda vacío.
- El rol (`owner`) no limita nada: las dos cuentas son administradoras.
- Un token mal formado (sin `sub` o `role`, o de otro tipo) responde 401.

## Crear la cuenta de tu mamá

La API no tiene registro. La contraseña no debe pasar por el chat ni por el historial.

1. Genera el hash sin que la contraseña salga de tu terminal: `cd backend && go run ./cmd/hashpassword`.
2. En el SQL Editor de Supabase: `insert into manager.users (name, email, password, role) values ('<nombre>', '<correo>', '<hash>', 'owner');`
3. Comprueba que entra y que ve los mismos datos que tú.

## Cambiar una contraseña

Genera un hash nuevo con `hashpassword` y ejecuta
`update manager.users set password = '<hash>' where email = '<correo>';`. Cambiarla desde la app
no está hecho todavía.

## Quitar el acceso a alguien

`delete from manager.users where email = '<correo>';` Sus registros se quedan. Su token sigue
valiendo hasta que venza; si hay que cortarlo ya, cambia el `JWT_SECRET` (esto cierra la sesión de
todas las cuentas).
