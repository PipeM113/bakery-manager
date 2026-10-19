# Ajustes de stock y merma

Todo cambio del stock que no es una venta deja un registro con su motivo (tabla
`stock_adjustments`). Los ajustes no se borran ni se editan: un error se arregla con otro ajuste.

## Los tres motivos

| Motivo | Qué hace | Pérdida |
|---|---|---|
| `merma` | Algo se perdió o se echó a perder. Baja el stock en `quantity`. | Sí: `quantity × precio por unidad` |
| `correccion` | Contaste el stock real. Deja el stock en `counted_quantity` y guarda la diferencia (positiva o negativa). | No |
| `otro` | Regalo, muestra u otro caso. Sube o baja el stock en `quantity_delta` (con signo). **Exige una nota.** | No |

- La cantidad puede venir en otra unidad de la misma clase (kg↔gr, lt↔ml) con `unit`; se convierte
  a la unidad del ingrediente. Unidades incompatibles responden 400.
- Una merma mayor al stock disponible responde 400 (para corregir primero se usa una
  `correccion`). Ningún ajuste deja el stock negativo.
- La pérdida se guarda en `loss_amount` (pesos enteros, 0 si no es merma). Es `cantidad × precio por unidad` con el precio exacto (precio del envase ÷ tamaño
  del envase), redondeada **una sola vez** a pesos enteros, mitades hacia arriba.
- La fecha (`adjusted_on`, por defecto hoy en Santiago) puede ser pasada, no futura.
- Dos ajustes al mismo tiempo sobre un ingrediente se esperan entre sí: el stock no queda
  negativo y no se pierde ninguno.

## Cambiar el stock desde la edición del ingrediente

`PUT /ingredients/{id}` que cambia `stock_quantity` deja un ajuste automático de tipo `correccion`
(con la diferencia). Si el stock no cambia (por ejemplo, solo se renombra), no deja nada. Importar
ingredientes desde Excel no cambia el stock, así que tampoco deja ajustes.

## API

| Llamada | Qué hace |
|---|---|
| `POST /ingredients/{id}/stock-adjustments` | Registra un ajuste. Cuerpo: `reason`, y según el motivo `quantity`, `counted_quantity` o `quantity_delta`; opcionales `unit`, `note`, `adjusted_on`. |
| `GET /ingredients/{id}/stock-adjustments` | Historial del ingrediente, el más reciente primero. |
| `GET /stock-adjustments?month=AAAA-MM` | Lo ajustado en un mes (el actual por defecto), con el nombre del ingrediente. Un mes mal escrito responde 400. |

## Dónde se ve la pérdida

`total_waste_cost` en `/analytics/monthly` suma las **mermas** del mes y se resta de la ganancia
neta (`net_profit`). La tendencia (`/analytics/trends`) hace lo mismo mes a mes, dentro de las
mismas consultas que los gastos. La pantalla de Rentabilidad muestra "Mermas del mes" y la línea
"− Mermas" en el desglose.

## Lo que no hace (todavía)

- Un porcentaje de pérdida dentro del costo de la receta.
- Anular un ajuste.
- Ajustar por envases en vez de por unidad.

## Migración 21

Crea `stock_adjustments` (RLS activo, sin políticas). Solo agrega una tabla.
