# Fechas, horas y meses

La pastelería vive en Chile, así que "hoy" y "este mes" son los de **America/Santiago**, sin
importar la hora del servidor (Vercel corre en UTC). Dos horas distintas pueden dar días
distintos: a las 23:30 del 31 de marzo en Santiago ya es 1 de abril en UTC.

## El reloj

- Toda la app lee la hora a través del paquete `internal/shared/clock` (el **reloj**). Nadie más
  llama a `time.Now()` (lo verifica `internal/hygiene/time_test.go`); la única excepción es el
  vencimiento del token de sesión, que usa la hora real.
- El reloj da `Today()` (fecha de Santiago, `YYYY-MM-DD`) y `Month()` (mes y año de Santiago).
- La base de datos de zonas horarias viaja dentro del programa (`time/tzdata`): no depende de que
  el servidor la tenga. Si faltara, el programa no volvería a UTC en silencio.
- En los tests se pasa un reloj fijo (`clock.Fixed`), así se prueban las horas límite y los
  cambios de horario (Chile retrasa el reloj la primera noche de sábado de abril y lo adelanta la
  de septiembre).

## Dónde se usa

- Fecha por defecto de una venta y fecha de vigencia del historial de precios.
- Mes por defecto del dashboard, de la rentabilidad por receta y del listado de gastos.
- Un gasto no puede tener fecha futura: se compara con hoy en Santiago, a cualquier hora.
- Fecha del PDF de cotización y nombre de los archivos (`cotizacion_..._2026-03-31.pdf`,
  `insumos_20260331.xlsx`).

## Meses de la tendencia

`GET /analytics/trends?months=N` (de 1 a 24; 6 si no se indica; fuera de rango responde 400) lista
N meses consecutivos que terminan en el mes actual de Santiago, del más antiguo al más reciente.
Los meses se cuentan desde el **primer día del mes**: restar un mes al 31 de marzo ya no cae en
marzo otra vez. La tendencia hace cuatro consultas en total, sin importar los meses, y cada punto
es igual al dashboard mensual de ese mes.

## Columnas

Las fechas calendario (`sale_date`, `expense_date`, vigencia de precios) son `DATE`. Los instantes
(`created_at`, `updated_at`) son `TIMESTAMPTZ` desde la migración 18; los valores que ya existían se
leyeron como UTC, que es como los guardaba el servidor.
