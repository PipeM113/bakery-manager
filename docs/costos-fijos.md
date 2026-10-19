# Costos fijos con histórico

Un costo fijo (arriendo, gas, luz, contador…) tiene **un monto por mes**. Si el gas sube, el
cambio vale desde ese mes en adelante y los meses anteriores no se tocan.

## Cómo se guarda

- `fixed_costs`: la identidad del costo (nombre, dueño) y `ended_from`, el primer mes en que el
  costo **deja de contar** (vacío si sigue vigente).
- `fixed_cost_versions` (`fixed_cost_id`, `valid_from`, `monthly_amount`): una fila por cambio de
  monto. `valid_from` es siempre el día 1 de un mes; no puede haber dos versiones del mismo mes.
- El monto de un mes es el de la versión más reciente con `valid_from` menor o igual a ese mes. Un
  costo cuya primera versión es posterior a ese mes no existía todavía y no aparece.
- Ambas tablas tienen RLS activo y ninguna policy, como el resto.

## API

| Llamada | Qué hace |
|---|---|
| `GET /fixed-costs?month=AAAA-MM` | Los costos que existen ese mes, con el monto de ese mes y `is_active` (si cuenta ese mes). Sin `month`, el mes actual de Santiago. |
| `POST /fixed-costs` `{name, monthly_amount, valid_from?}` | Crea el costo desde `valid_from` (`AAAA-MM`; por defecto el mes actual). Puede ser un mes pasado, por si se olvidó anotarlo. |
| `PUT /fixed-costs/{id}` `{name, monthly_amount, is_active, effective_from?}` | Pone una versión del monto en `effective_from` (por defecto el mes actual) **solo si cambió** respecto del monto vigente ese mes, o reemplaza la versión de ese mes. `is_active:false` termina el costo desde ese mes; `true` quita el término. |
| `DELETE /fixed-costs/{id}` | Termina el costo desde el mes actual. No borra nada: los meses anteriores siguen contando. |

- Un mes con formato distinto de `AAAA-MM` responde 400. Un costo de otro usuario responde 404.
- Se pueden corregir meses pasados: cambiar el monto de un mes pasado solo afecta a ese mes y a
  los siguientes hasta la próxima versión.
- Renombrar no crea versión: el nombre es la identidad y vale para todo el histórico.
- Quitar el término (`is_active:true`) hace que el costo vuelva a contar en todos los meses desde
  su inicio, incluso los que estaban terminados.

## Dónde se usa

- El dashboard (`/analytics/monthly`) suma los costos fijos que existen y cuentan ese mes, cada
  uno con el monto de ese mes.
- La tendencia (`/analytics/trends`) lo hace mes a mes en una sola consulta.
- La pantalla **Costos fijos** tiene un selector de mes y pide "Vigente desde" al crear o
  cambiar un monto (por defecto el mes de Santiago).

## Migración 19

Cada costo existente queda con una versión desde su mes de creación (en hora de Santiago); los que
estaban inactivos quedan terminados desde el mes de su última actualización. El `down` deja en
`fixed_costs` el último monto de cada costo y `is_active` según el término: **se pierde el
histórico**.
