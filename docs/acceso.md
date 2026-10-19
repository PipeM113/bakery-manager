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

## Cambiar tu contraseña (Mi cuenta)

En la app, **Mi cuenta** (menú "Más" en el celular, barra lateral en el computador): escribes la
contraseña actual, la nueva y la repites.

- La nueva debe tener **al menos 10 caracteres**, no pasar de 72 bytes (el límite de bcrypt) y ser
  distinta de la actual. Si la actual no es correcta, se rechaza y **cuenta como intento fallido**
  (ver abajo).
- Solo cambia la contraseña de quien tiene la sesión abierta.
- Las sesiones que ya están abiertas (en otro aparato, por ejemplo) **siguen valiendo hasta que su
  token vence (24 horas)**: cambiar la contraseña no las cierra. Si hay que cortarlas ya, cambia el
  `JWT_SECRET` (cierra todas las sesiones).
- Si olvidaste la contraseña y no puedes entrar, la otra cuenta no puede cambiártela desde la app;
  usa el hash y el `update` de SQL: `cd backend && go run ./cmd/hashpassword` y
  `update manager.users set password = '<hash>' where email = '<correo>';`.

## Intentos de entrada (bloqueo)

Después de **5 intentos fallidos con el mismo correo en 15 minutos**, el login responde **429**
("Demasiados intentos fallidos. Vuelve a intentar en N minutos"), con la cabecera `Retry-After`, **aunque
la contraseña sea la correcta**. El bloqueo termina cuando pasan 15 minutos desde los fallos; un
ingreso correcto antes de llegar a 5 pone el contador en cero.

- Se cuenta por **correo** (sin distinguir mayúsculas), no por IP, y los intentos viven en la base
  (`manager.login_attempts`: solo el correo y el momento, nunca la contraseña), porque cada instancia de
  Vercel es un proceso distinto. Un correo que no existe cuenta igual y responde igual que una clave
  mala: la respuesta no dice qué correos existen.
- Costo conocido: alguien que adivine tu correo puede dejarte 15 minutos sin entrar. Para desbloquear
  antes: `delete from manager.login_attempts where email = '<correo en minúsculas>';`.
- No frena a quien prueba muchos correos distintos; con dos cuentas conocidas, es suficiente.

## Tamaño de lo que se envía y registros del servidor

- Los endpoints JSON rechazan cuerpos de más de **1 MiB** con 413. La subida de fotos (4 MB) y la
  importación de ingredientes (4 MiB) tienen su propio límite.
- Los registros (logs) del servidor llevan ids, montos y fechas, **nunca texto escrito por personas**
  (descripciones, notas, nombres de clientes, correos).

## Quitar el acceso a alguien

`delete from manager.users where email = '<correo>';` Sus registros se quedan. Su token sigue
valiendo hasta que venza; si hay que cortarlo ya, cambia el `JWT_SECRET` (esto cierra la sesión de
todas las cuentas).
