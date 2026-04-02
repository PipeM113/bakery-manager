import { useEffect, useState } from "react";
import { fixedCostService } from "../api/fixedCostService";
import type { IFixedCost } from "../types/fixedCost";

const clp = (n: number) => `$${Math.round(n).toLocaleString("es-CL")}`;

const emptyForm = { name: "", monthly_amount: "" };

export default function FixedCostsPage() {
  const [costs, setCosts]         = useState<IFixedCost[]>([]);
  const [form, setForm]           = useState(emptyForm);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [error, setError]         = useState("");
  const [loading, setLoading]     = useState(false);

  const load = () =>
    fixedCostService.getAll().then(setCosts).catch(() => setError("Error cargando costos fijos"));

  useEffect(() => { load(); }, []);

  const totalActive = costs
    .filter((c) => c.is_active)
    .reduce((sum, c) => sum + c.monthly_amount, 0);

  const startEdit = (fc: IFixedCost) => {
    setEditingId(fc.id);
    setForm({ name: fc.name, monthly_amount: String(fc.monthly_amount) });
    setError("");
  };

  const cancelEdit = () => {
    setEditingId(null);
    setForm(emptyForm);
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
        const updated = await fixedCostService.update(editingId, {
          name: form.name.trim(),
          monthly_amount: amount,
          is_active: existing.is_active,
        });
        setCosts((prev) => prev.map((c) => (c.id === editingId ? updated : c)));
        cancelEdit();
      } else {
        const created = await fixedCostService.create({
          name: form.name.trim(),
          monthly_amount: amount,
        });
        setCosts((prev) => [...prev, created]);
        setForm(emptyForm);
      }
    } catch {
      setError("Error guardando costo fijo");
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("¿Eliminar este costo fijo?")) return;
    try {
      await fixedCostService.delete(id);
      setCosts((prev) => prev.filter((c) => c.id !== id));
    } catch {
      setError("Error eliminando costo fijo");
    }
  };

  const handleToggle = async (fc: IFixedCost) => {
    try {
      const updated = await fixedCostService.update(fc.id, {
        name: fc.name,
        monthly_amount: fc.monthly_amount,
        is_active: !fc.is_active,
      });
      setCosts((prev) => prev.map((c) => (c.id === fc.id ? updated : c)));
    } catch {
      setError("Error actualizando estado");
    }
  };

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">

      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <div className="h-px w-8 bg-gold opacity-40" />
          <span className="text-gold text-xs tracking-[0.3em] uppercase font-light">Gestión</span>
        </div>
        <h1 className="font-display text-4xl text-stone-800">Costos Fijos</h1>
      </div>

      {/* Formulario */}
      <form onSubmit={handleSubmit} className="bg-white border border-stone-200 p-6 mb-8">
        <div className="flex items-center gap-4 mb-5">
          <h2 className="font-display text-xl text-stone-800">
            {editingId ? "Editar costo fijo" : "Nuevo costo fijo"}
          </h2>
          <div className="flex-1 h-px bg-stone-200" />
        </div>

        <div className="flex flex-col sm:flex-row gap-3 mb-4">
          <div className="flex-1">
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Nombre
            </label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="ej. Arriendo"
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            />
          </div>
          <div className="sm:w-48">
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Monto mensual
            </label>
            <input
              type="number"
              value={form.monthly_amount}
              onChange={(e) => setForm((f) => ({ ...f, monthly_amount: e.target.value }))}
              placeholder="ej. 100000"
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            />
          </div>
        </div>

        {error && (
          <p className="text-red-500 text-xs mb-3">{error}</p>
        )}

        <div className="flex gap-2">
          <button
            type="submit"
            disabled={loading}
            className="bg-gold text-white text-xs tracking-widest uppercase font-medium
                       px-6 py-2.5 hover:bg-gold-light transition-all disabled:opacity-50"
          >
            {loading ? "Guardando..." : editingId ? "Actualizar" : "Agregar"}
          </button>
          {editingId && (
            <button
              type="button"
              onClick={cancelEdit}
              className="text-xs tracking-widest uppercase font-medium px-6 py-2.5
                         border border-stone-200 text-stone-400 hover:border-stone-400 transition-all"
            >
              Cancelar
            </button>
          )}
        </div>
      </form>

      {/* Tabla */}
      <div className="bg-white border border-stone-200">
        <div className="flex items-center justify-between px-6 py-4 border-b border-stone-100">
          <h2 className="font-display text-xl text-stone-800">Lista</h2>
          {costs.length > 0 && (
            <span className="text-xs text-stone-400 font-light">
              Total activo:{" "}
              <span className="text-gold font-medium">{clp(totalActive)}/mes</span>
            </span>
          )}
        </div>

        {costs.length === 0 ? (
          <p className="text-stone-400 text-sm font-light text-center py-10">
            No hay costos fijos registrados
          </p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-stone-100">
                <th className="text-left px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Nombre</th>
                <th className="text-right px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Monto/mes</th>
                <th className="text-center px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Estado</th>
                <th className="px-6 py-3" />
              </tr>
            </thead>
            <tbody>
              {costs.map((fc) => (
                <tr key={fc.id} className="border-b border-stone-100 last:border-0 hover:bg-vanilla-100 transition-colors">
                  <td className="px-6 py-4 text-stone-800 font-light">{fc.name}</td>
                  <td className="px-6 py-4 text-right text-stone-800 font-light">{clp(fc.monthly_amount)}</td>
                  <td className="px-6 py-4 text-center">
                    <button
                      onClick={() => handleToggle(fc)}
                      className={`text-xs tracking-widest uppercase font-medium px-3 py-1 border transition-all
                        ${fc.is_active
                          ? "border-gold text-gold bg-vanilla-100"
                          : "border-stone-200 text-stone-400 hover:border-stone-400"}`}
                    >
                      {fc.is_active ? "Activo" : "Inactivo"}
                    </button>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex gap-3 justify-end">
                      <button
                        onClick={() => startEdit(fc)}
                        className="text-xs text-stone-400 hover:text-gold tracking-widest uppercase transition-colors"
                      >
                        Editar
                      </button>
                      <button
                        onClick={() => handleDelete(fc.id)}
                        className="text-xs text-stone-400 hover:text-red-500 tracking-widest uppercase transition-colors"
                      >
                        Eliminar
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
