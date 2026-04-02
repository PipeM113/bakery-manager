import { useEffect, useRef, useState } from "react";
import { expenseService } from "../api/expenseService";
import type { IExpense, ExpenseFormData } from "../types/expense";

const clp = (n: number) => `$${Math.round(n).toLocaleString("es-CL")}`;

const CATEGORIES: { value: IExpense["category"]; label: string }[] = [
  { value: "ingredientes", label: "Ingredientes" },
  { value: "servicios",    label: "Servicios"    },
  { value: "otros",        label: "Otros"        },
];

const today = () => new Date().toISOString().slice(0, 10);

const emptyForm: ExpenseFormData = {
  description: "",
  amount: "",
  category: "otros",
  expense_date: today(),
};

const MONTHS = [
  "Enero","Febrero","Marzo","Abril","Mayo","Junio",
  "Julio","Agosto","Septiembre","Octubre","Noviembre","Diciembre",
];

type Toast = { id: number; type: "success" | "error"; message: string };

export default function ExpensesPage() {
  const now = new Date();
  const [expenses, setExpenses] = useState<IExpense[]>([]);
  const [form, setForm]         = useState<ExpenseFormData>(emptyForm);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [error, setError]       = useState("");
  const [loading, setLoading]   = useState(false);
  const [filterMonth, setFilterMonth] = useState(now.getMonth() + 1);
  const [filterYear, setFilterYear]   = useState(now.getFullYear());
  const [toasts, setToasts]     = useState<Toast[]>([]);
  const toastCounter            = useRef(0);

  const addToast = (type: Toast["type"], message: string) => {
    const id = ++toastCounter.current;
    setToasts((prev) => [...prev, { id, type, message }]);
    setTimeout(() => setToasts((prev) => prev.filter((t) => t.id !== id)), 4000);
  };

  const load = (month: number, year: number) => {
    expenseService
      .getByMonth(month, year)
      .then((data) => {
        console.log("Expenses loaded:", data);
        setExpenses(data);
      })
      .catch((err) => {
        console.error("Error loading expenses:", err);
        setError("Error cargando gastos");
      });
  };

  useEffect(() => {
    load(filterMonth, filterYear);
  }, [filterMonth, filterYear]);

  const totalMonth = expenses.reduce((sum, e) => sum + e.amount, 0);

  const startEdit = (e: IExpense) => {
    setEditingId(e.id);
    setForm({
      description: e.description,
      amount: String(e.amount),
      category: e.category,
      expense_date: e.expense_date,
    });
    setError("");
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  const cancelEdit = () => {
    setEditingId(null);
    setForm(emptyForm);
    setError("");
  };

  const handleSubmit = async (ev: React.FormEvent) => {
    ev.preventDefault();
    setError("");
    const amount = parseFloat(form.amount);
    if (!form.description.trim()) { setError("La descripción es requerida"); return; }
    if (isNaN(amount) || amount <= 0) { setError("El monto debe ser mayor a 0"); return; }
    if (!form.expense_date) { setError("La fecha es requerida"); return; }
    if (form.expense_date > today()) { setError("La fecha no puede ser futura"); return; }

    setLoading(true);
    const payload = {
      description: form.description.trim(),
      amount,
      category: form.category,
      expense_date: form.expense_date,
    };
    console.log("Expense form submitted:", payload);
    try {
      if (editingId) {
        const updated = await expenseService.update(editingId, payload);
        console.log("API response (update):", updated);
        setExpenses((prev) => prev.map((e) => (e.id === editingId ? updated : e)));
        addToast("success", "✅ Gasto actualizado");
        cancelEdit();
      } else {
        const created = await expenseService.create(payload);
        console.log("API response (create):", created);
        // Only add to list if it matches current filter
        const [cy, cm] = created.expense_date.split("-").map(Number);
        if (cm === filterMonth && cy === filterYear) {
          setExpenses((prev) => [created, ...prev]);
        }
        addToast("success", "✅ Gasto registrado");
        setForm(emptyForm);
      }
    } catch (err: unknown) {
      const apiMsg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      const msg = apiMsg ?? "Error desconocido al guardar";
      console.error("Error saving expense:", err);
      setError(`❌ No se pudo guardar el gasto: ${msg}`);
      addToast("error", `❌ No se pudo guardar el gasto: ${msg}`);
      // form is NOT cleared so the user can retry
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("¿Eliminar este gasto?")) return;
    try {
      await expenseService.delete(id);
      setExpenses((prev) => prev.filter((e) => e.id !== id));
      addToast("success", "✅ Gasto eliminado");
    } catch (err) {
      console.error("Error deleting expense:", err);
      setError("Error eliminando gasto");
      addToast("error", "❌ Error eliminando gasto");
    }
  };

  const yearOptions = [];
  for (let y = now.getFullYear(); y >= now.getFullYear() - 3; y--) yearOptions.push(y);

  const categoryLabel = (cat: string) =>
    CATEGORIES.find((c) => c.value === cat)?.label ?? cat;

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">

      {/* Toast notifications */}
      {toasts.length > 0 && (
        <div className="fixed top-4 right-4 z-50 flex flex-col gap-2">
          {toasts.map((t) => (
            <div
              key={t.id}
              className={`px-4 py-3 text-sm font-light shadow-md transition-all
                ${t.type === "success"
                  ? "bg-white border border-green-200 text-green-800"
                  : "bg-white border border-red-200 text-red-700"}`}
            >
              {t.message}
            </div>
          ))}
        </div>
      )}

      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <div className="h-px w-8 bg-gold opacity-40" />
          <span className="text-gold text-xs tracking-[0.3em] uppercase font-light">Gestión</span>
        </div>
        <h1 className="font-display text-4xl text-stone-800">Gastos Operacionales</h1>
      </div>

      {/* Form */}
      <form onSubmit={handleSubmit} className="bg-white border border-stone-200 p-6 mb-8">
        <div className="flex items-center gap-4 mb-5">
          <h2 className="font-display text-xl text-stone-800">
            {editingId ? "Editar gasto" : "Registrar Gasto"}
          </h2>
          <div className="flex-1 h-px bg-stone-200" />
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-4">
          {/* Description */}
          <div className="sm:col-span-2">
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Descripción
            </label>
            <input
              type="text"
              value={form.description}
              onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              placeholder="ej. Gas mensual"
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            />
          </div>

          {/* Amount */}
          <div>
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Monto (CLP)
            </label>
            <input
              type="number"
              min="1"
              value={form.amount}
              onChange={(e) => setForm((f) => ({ ...f, amount: e.target.value }))}
              placeholder="ej. 50000"
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            />
          </div>

          {/* Category */}
          <div>
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Categoría
            </label>
            <select
              value={form.category}
              onChange={(e) =>
                setForm((f) => ({ ...f, category: e.target.value as IExpense["category"] }))
              }
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            >
              {CATEGORIES.map((c) => (
                <option key={c.value} value={c.value}>{c.label}</option>
              ))}
            </select>
          </div>

          {/* Date */}
          <div>
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Fecha del gasto
            </label>
            <input
              type="date"
              value={form.expense_date}
              max={today()}
              onChange={(e) => setForm((f) => ({ ...f, expense_date: e.target.value }))}
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            />
          </div>
        </div>

        {error && <p className="text-red-500 text-xs mb-3">{error}</p>}

        <div className="flex gap-2">
          <button
            type="submit"
            disabled={loading}
            className="bg-gold text-white text-xs tracking-widest uppercase font-medium
                       px-6 py-2.5 hover:bg-gold-light transition-all disabled:opacity-50"
          >
            {loading ? "Guardando..." : editingId ? "Actualizar" : "Registrar Gasto"}
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

      {/* List */}
      <div className="bg-white border border-stone-200">
        <div className="flex flex-wrap items-center justify-between gap-3 px-6 py-4 border-b border-stone-100">
          <h2 className="font-display text-xl text-stone-800">
            Gastos de{" "}
            <span className="text-gold">
              {MONTHS[filterMonth - 1]} {filterYear}
            </span>
          </h2>

          <div className="flex items-center gap-2">
            <select
              value={filterMonth}
              onChange={(e) => setFilterMonth(Number(e.target.value))}
              className="bg-vanilla-100 border border-stone-200 text-stone-600 px-3 py-1.5
                         text-xs font-light focus:outline-none focus:border-gold transition-all"
            >
              {MONTHS.map((m, i) => (
                <option key={i + 1} value={i + 1}>{m}</option>
              ))}
            </select>
            <select
              value={filterYear}
              onChange={(e) => setFilterYear(Number(e.target.value))}
              className="bg-vanilla-100 border border-stone-200 text-stone-600 px-3 py-1.5
                         text-xs font-light focus:outline-none focus:border-gold transition-all"
            >
              {yearOptions.map((y) => (
                <option key={y} value={y}>{y}</option>
              ))}
            </select>

            <button
              type="button"
              onClick={() => load(filterMonth, filterYear)}
              title="Recargar"
              className="text-xs text-stone-400 hover:text-gold tracking-widest uppercase transition-colors px-2 py-1.5 border border-stone-200 hover:border-gold"
            >
              🔄 Recargar
            </button>

            {expenses.length > 0 && (
              <span className="text-xs text-stone-400 font-light ml-2">
                Total:{" "}
                <span className="text-gold font-medium">{clp(totalMonth)}</span>
              </span>
            )}
          </div>
        </div>

        {expenses.length === 0 ? (
          <p className="text-stone-400 text-sm font-light text-center py-10">
            No hay gastos registrados para este período
          </p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-stone-100">
                <th className="text-left px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Fecha</th>
                <th className="text-left px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Descripción</th>
                <th className="text-left px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Categoría</th>
                <th className="text-right px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Monto</th>
                <th className="px-6 py-3" />
              </tr>
            </thead>
            <tbody>
              {expenses.map((e) => (
                <tr key={e.id} className="border-b border-stone-100 last:border-0 hover:bg-vanilla-100 transition-colors">
                  <td className="px-6 py-4 text-stone-500 font-light whitespace-nowrap">
                    {e.expense_date}
                  </td>
                  <td className="px-6 py-4 text-stone-800 font-light">{e.description}</td>
                  <td className="px-6 py-4">
                    <span className="text-xs tracking-widest uppercase font-medium px-2 py-1 border border-stone-200 text-stone-500">
                      {categoryLabel(e.category)}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-right text-stone-800 font-light">{clp(e.amount)}</td>
                  <td className="px-6 py-4">
                    <div className="flex gap-3 justify-end">
                      <button
                        onClick={() => startEdit(e)}
                        className="text-xs text-stone-400 hover:text-gold tracking-widest uppercase transition-colors"
                      >
                        Editar
                      </button>
                      <button
                        onClick={() => handleDelete(e.id)}
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
