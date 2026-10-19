import type { IOrder } from "../types/order";
import { useIsPhone } from "../hooks/useIsPhone";
import { formatCLP } from "../utils/money";

const summary = (o: IOrder) =>
  o.lines.map((l) => `${l.quantity_sold} × ${l.recipe_name}${l.sold_as === "portion" ? " (porción)" : ""}`);

interface Props {
  orders: IOrder[];
  onCancel: (order: IOrder) => void;
}

/** Orders as cards on a phone and as a table on bigger screens. */
export default function OrdersList({ orders, onCancel }: Props) {
  const phone = useIsPhone();

  if (orders.length === 0) {
    return <p className="text-negro/70 text-sm font-light text-center py-10">No hay pedidos este mes</p>;
  }

  if (phone) {
    return (
      <div data-testid="orders-list" data-view="cards" className="p-3 space-y-3">
        {orders.map((o) => (
          <div key={o.id} data-card className="border border-linea p-4 space-y-3">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="text-negro text-sm font-light break-words">{o.customer_name || "Sin cliente"}</p>
                <p className="text-xs text-negro/70 font-light">{o.order_date}</p>
              </div>
              <p className="font-medium text-negro shrink-0">{formatCLP(o.total_price)}</p>
            </div>
            <ul className="text-xs text-negro/70 font-light space-y-0.5">
              {summary(o).map((s, i) => <li key={i} className="break-words">{s}</li>)}
            </ul>
            {o.notes && <p className="text-xs text-negro/70 font-light break-words">{o.notes}</p>}
            <button onClick={() => onCancel(o)}
              className="w-full text-xs tracking-widest uppercase font-medium border border-linea text-negro/70 hover:border-peligro hover:text-peligro transition-all">
              Anular pedido
            </button>
          </div>
        ))}
      </div>
    );
  }

  return (
    <div data-testid="orders-list" data-view="table" className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-linea">
            {["Fecha", "Cliente", "Productos", "Total", ""].map((h, i) => (
              <th key={i} className={`px-6 py-3 text-xs tracking-widest uppercase text-negro/70 font-light ${h === "Total" ? "text-right" : "text-left"}`}>{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {orders.map((o) => (
            <tr key={o.id} className="border-b border-linea last:border-0 hover:bg-lino transition-colors align-top">
              <td className="px-6 py-4 text-negro/70 font-light whitespace-nowrap">{o.order_date}</td>
              <td className="px-6 py-4 text-negro font-light">{o.customer_name || "Sin cliente"}</td>
              <td className="px-6 py-4 text-negro/70 font-light">
                <ul className="space-y-0.5">{summary(o).map((s, i) => <li key={i}>{s}</li>)}</ul>
              </td>
              <td className="px-6 py-4 text-right font-medium text-negro whitespace-nowrap">{formatCLP(o.total_price)}</td>
              <td className="px-6 py-4 text-right whitespace-nowrap">
                <button onClick={() => onCancel(o)}
                  className="text-xs text-negro/70 hover:text-peligro tracking-widest uppercase transition-colors">
                  Anular pedido
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
