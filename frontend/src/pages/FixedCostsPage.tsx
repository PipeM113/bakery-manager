import { useEffect, useState } from "react";
import { fixedCostService } from "../api/fixedCostService";
import type { IFixedCost } from "../types/fixedCost";
import { todayInSantiago } from "../utils/businessDate";
import { formatCLP } from "../utils/money";


const thisMonth = () => todayInSantiago().slice(0, 7);
const emptyForm = (month: string) => ({ name: "", monthly_amount: "", from: month });

export default function FixedCostsPage() {
  const [costs, setCosts]         = useState<IFixedCost[]>([]);
  const [month, setMonth]         = useState(thisMonth);
  const [form, setForm]           = useState(() => emptyForm(thisMonth()));
  const [editingId, setEditingId] = useState<string | null>(null);
  const [error, setError]         = useState("");
  const [loading, setLoading]     = useState(false);

  const load = () =>
    fixedCostService.getAll(month).then(setCosts).catch(() => setError("Error cargando costos fijos"));

  useEffect(() => { load(); }, [month]);

  const totalActive = costs
    .filter((c) => c.is_active)
    .reduce((sum, c) => sum + c.monthly_amount, 0);

  const startEdit = (fc: IFixedCost) => {
    setEditingId(fc.id);
    setForm({ name: fc.name, monthly_amount: String(fc.monthly_amount), from: month });
    setError("");
  };

  const cancelEdit = () => {
    setEditingId(null);
    setForm(emptyForm(month));
    setError("");
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    const amount = parseFloat(form.monthly_amount);
    if (!form.name.trim() || isNaN(amount) || amount <= 0) {
      setError("Nombre y monto son requeridos");
      return;
    }
    setLoading(true);
    try {
      if (editingId) {
        const existing = costs.find((c) => c.id === editingId)!;
        await fixedCostService.update(editingId, {
          name: form.name.trim(),
          monthly_amount: amount,
          is_active: existing.is_active,
          effective_from: form.from || month,
        });
        cancelEdit();
        await load();
      } else {
        await fixedCostService.create({
          name: form.name.trim(),
          monthly_amount: amount,
          valid_from: form.from || month,
        });
        setForm(emptyForm(month));
        await load();
      }
    } catch {
      setError("Error guardando costo fijo");
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("¿Terminar este costo fijo desde este mes? Los meses anteriores no cambian.")) return;
    try {
      await fixedCostService.delete(id);
      await load();
    } catch {
      setError("Error eliminando costo fijo");
    }
  };

  const handleToggle = async (fc: IFixedCost) => {
    try {
      await fixedCostService.update(fc.id, {
        name: fc.name,
        monthly_amount: fc.monthly_amount,
        is_active: !fc.is_active,
        effective_from: month,
      });
      await load();
    } catch {
      setError("Error actualizando estado");
    }
  };

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">

      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <div className="h-px w-8 bg-oro/40" />
          <span className="text-oro text-xs tracking-[0.3em] uppercase font-light">Gestión</span>
        </div>
        <h1 className="font-display text-4xl text-negro">Costos Fijos</h1>
        <div className="mt-4 flex items-center gap-3">
          <label className="text-xs tracking-widest uppercase text-negro/70 font-light" htmlFor="fixed-month">
            Mes
          </label>
          <input
            id="fixed-month"
            type="month"
            value={month}
            onChange={(e) => e.target.value && setMonth(e.target.value)}
            className="bg-white border border-linea text-negro px-3 py-1.5 text-sm font-light focus:outline-none focus:border-oro"
          />
        </div>
      </div>

      {/* Formulario */}
      <form onSubmit={handleSubmit} className="bg-white border border-linea p-6 mb-8">
        <div className="flex items-center gap-4 mb-5">
          <h2 className="font-display text-xl text-negro">
            {editingId ? "Editar costo fijo" : "Nuevo costo fijo"}
          </h2>
          <div className="flex-1 h-px bg-lino" />
        </div>

        <div className="flex flex-col sm:flex-row gap-3 mb-4">
          <div className="flex-1">
            <label className="block text-xs tracking-widest uppercase text-negro/70 mb-1.5 font-light">
              Nombre
            </label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="ej. Arriendo"
              className="w-full bg-crema border border-linea text-negro
                         px-4 py-2.5 text-sm font-light focus:outline-none focus:border-oro transition-all"
            />
          </div>
          <div className="sm:w-44">
            <label className="block text-xs tracking-widest uppercase text-negro/70 mb-1.5 font-light">
              Vigente desde
            </label>
            <input
              type="month"
              value={form.from}
              onChange={(e) => setForm((f) => ({ ...f, from: e.target.value }))}
              className="w-full bg-crema border border-linea text-negro
                         px-4 py-2.5 text-sm font-light focus:outline-none focus:border-oro transition-all"
            />
          </div>
          <div className="sm:w-48">
            <label className="block text-xs tracking-widest uppercase text-negro/70 mb-1.5 font-light">
              Monto mensual
            </label>
            <input
              type="number"
              value={form.monthly_amount}
              onChange={(e) => setForm((f) => ({ ...f, monthly_amount: e.target.value }))}
              placeholder="ej. 100000"
              className="w-full bg-crema border border-linea text-negro
                         px-4 py-2.5 text-sm font-light focus:outline-none focus:border-oro transition-all"
            />
          </div>
        </div>

        {error && (
          <p className="text-peligro text-xs mb-3">{error}</p>
        )}

        <div className="flex gap-2">
          <button
            type="submit"
            disabled={loading}
            className="bg-negro text-crema text-xs tracking-widest uppercase font-medium
                       px-6 py-2.5 hover:bg-black transition-all disabled:opacity-50"
          >
            {loading ? "Guardando..." : editingId ? "Actualizar" : "Agregar"}
          </button>
          {editingId && (
            <button
              type="button"
              onClick={cancelEdit}
              className="text-xs tracking-widest uppercase font-medium px-6 py-2.5
                         border border-linea text-negro/70 hover:border-oro transition-all"
            >
              Cancelar
            </button>
          )}
        </div>
      </form>

      {/* Tabla */}
      <div className="bg-white border border-linea">
        <div className="flex items-center justify-between px-6 py-4 border-b border-linea">
          <h2 className="font-display text-xl text-negro">Lista</h2>
          {costs.length > 0 && (
            <span className="text-xs text-negro/70 font-light">
              Total del mes:{" "}
              <span className="text-oro font-medium">{formatCLP(totalActive)}/mes</span>
            </span>
          )}
        </div>

        {costs.length === 0 ? (
          <p className="text-negro/70 text-sm font-light text-center py-10">
            No hay costos fijos en este mes
          </p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-linea">
                <th className="text-left px-6 py-3 text-xs tracking-widest uppercase text-negro/70 font-light">Nombre</th>
                <th className="text-right px-6 py-3 text-xs tracking-widest uppercase text-negro/70 font-light">Monto/mes</th>
                <th className="text-center px-6 py-3 text-xs tracking-widest uppercase text-negro/70 font-light">Estado</th>
                <th className="px-6 py-3" />
              </tr>
            </thead>
            <tbody>
              {costs.map((fc) => (
                <tr key={fc.id} className="border-b border-linea last:border-0 hover:bg-lino transition-colors">
                  <td className="px-6 py-4 text-negro font-light">{fc.name}</td>
                  <td className="px-6 py-4 text-right text-negro font-light">{formatCLP(fc.monthly_amount)}</td>
                  <td className="px-6 py-4 text-center">
                    <button
                      onClick={() => handleToggle(fc)}
                      className={`text-xs tracking-widest uppercase font-medium px-3 py-1 border transition-all
                        ${fc.is_active
                          ? "border-oro text-oro bg-crema"
                          : "border-linea text-negro/70 hover:border-oro"}`}
                    >
                      {fc.is_active ? "Activo" : "Inactivo"}
                    </button>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex gap-3 justify-end">
                      <button
                        onClick={() => startEdit(fc)}
                        className="text-xs text-negro/70 hover:text-oro tracking-widest uppercase transition-colors"
                      >
                        Editar
                      </button>
                      <button
                        onClick={() => handleDelete(fc.id)}
                        className="text-xs text-negro/70 hover:text-peligro tracking-widest uppercase transition-colors"
                      >
                        Terminar
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
