import { useEffect, useState } from "react";
import api from "../services/api";

interface Ingredient {
  id: string;
  name: string;
  default_unit: string;
  price_per_unit: number;
  stock_quantity: number;
  alert_threshold: number;
  supplier: string;
}

const UNITS = ["gr", "kg", "ml", "lt", "und"];

const emptyForm = {
  name: "", default_unit: "gr", price_per_unit: 0,
  stock_quantity: 0, alert_threshold: 0, supplier: "",
};

export default function IngredientsPage() {
  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [search, setSearch]           = useState("");
  const [showForm, setShowForm]       = useState(false);
  const [editing, setEditing]         = useState<Ingredient | null>(null);
  const [form, setForm]               = useState(emptyForm);
  const [loading, setLoading]         = useState(true);
  const [saving, setSaving]           = useState(false);
  const [error, setError]             = useState("");

  const load = () => {
    api.get("/ingredients").then((r) => setIngredients(r.data)).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const filtered = ingredients.filter((i) =>
    i.name.toLowerCase().includes(search.toLowerCase())
  );

  const openCreate = () => {
    setEditing(null);
    setForm(emptyForm);
    setError("");
    setShowForm(true);
  };

  const openEdit = (i: Ingredient) => {
    setEditing(i);
    setForm({
      name: i.name, default_unit: i.default_unit,
      price_per_unit: i.price_per_unit, stock_quantity: i.stock_quantity,
      alert_threshold: i.alert_threshold, supplier: i.supplier,
    });
    setError("");
    setShowForm(true);
  };

  const handleSave = async () => {
    setError("");
    setSaving(true);
    try {
      if (editing) {
        await api.put(`/ingredients/${editing.id}`, form);
      } else {
        await api.post("/ingredients", form);
      }
      setShowForm(false);
      load();
    } catch (e: any) {
      setError(e.response?.data?.error || "Error al guardar");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("¿Eliminar este insumo?")) return;
    await api.delete(`/ingredients/${id}`);
    load();
  };

  if (loading) return (
    <div className="min-h-screen flex items-center justify-center">
      <span className="text-gold text-sm tracking-widest uppercase animate-pulse">Cargando...</span>
    </div>
  );

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">

      {/* Header */}
      <div className="flex items-end justify-between mb-8">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <div className="h-px w-8 bg-gold opacity-50" />
            <span className="text-gold text-xs tracking-[0.3em] uppercase font-light">Catálogo</span>
          </div>
          <h1 className="font-display text-4xl text-cream">Insumos</h1>
        </div>
        <button
          onClick={openCreate}
          className="bg-gold text-noir text-xs tracking-widest uppercase font-medium px-5 py-3
                     hover:bg-gold-light transition-all duration-200"
        >
          + Nuevo
        </button>
      </div>

      {/* Búsqueda */}
      <input
        type="text"
        placeholder="Buscar insumo..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="w-full bg-noir-700 border border-gold border-opacity-20 text-cream text-sm
                   px-4 py-3 mb-6 font-light tracking-wide placeholder-stone-600
                   focus:outline-none focus:border-gold focus:border-opacity-60 transition-all"
      />

      {/* Lista */}
      {filtered.length === 0 ? (
        <p className="text-center text-cream-muted text-sm font-light py-16 tracking-wide">
          No hay insumos registrados
        </p>
      ) : (
        <div className="space-y-2">
          {filtered.map((i) => {
            const lowStock = i.alert_threshold > 0 && i.stock_quantity < i.alert_threshold;
            return (
              <div
                key={i.id}
                className={`flex items-center justify-between p-4 bg-noir-700
                  border transition-all duration-200
                  ${lowStock ? "border-terracota-400 border-opacity-50" : "border-gold border-opacity-10 hover:border-opacity-30"}`}
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-cream text-sm font-light truncate">{i.name}</p>
                    {lowStock && <span className="text-terracota-400 text-xs">◆ Stock bajo</span>}
                  </div>
                  <p className="text-cream-muted text-xs mt-0.5 font-light">
                    ${i.price_per_unit.toFixed(2)} / {i.default_unit}
                    {i.supplier && <span className="ml-2 opacity-60">· {i.supplier}</span>}
                  </p>
                </div>
                <div className="flex items-center gap-3 ml-4">
                  <button
                    onClick={() => openEdit(i)}
                    className="text-xs text-cream-muted hover:text-gold tracking-wider transition-colors"
                  >
                    Editar
                  </button>
                  <button
                    onClick={() => handleDelete(i.id)}
                    className="text-xs text-cream-muted hover:text-terracota-400 tracking-wider transition-colors"
                  >
                    Eliminar
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Modal */}
      {showForm && (
        <div className="fixed inset-0 z-50 bg-black bg-opacity-70 flex items-end md:items-center justify-center p-4">
          <div className="bg-noir-800 border border-gold border-opacity-30 w-full max-w-md p-6 space-y-5">
            <div className="flex items-center justify-between">
              <h2 className="font-display text-2xl text-gold">
                {editing ? "Editar insumo" : "Nuevo insumo"}
              </h2>
              <button onClick={() => setShowForm(false)} className="text-cream-muted hover:text-cream text-xl">×</button>
            </div>

            {[
              { label: "Nombre", key: "name", type: "text" },
              { label: "Precio por unidad", key: "price_per_unit", type: "number" },
              { label: "Stock actual", key: "stock_quantity", type: "number" },
              { label: "Umbral de alerta", key: "alert_threshold", type: "number" },
              { label: "Proveedor", key: "supplier", type: "text" },
            ].map(({ label, key, type }) => (
              <div key={key}>
                <label className="block text-xs tracking-widest uppercase text-cream-muted mb-1.5 font-light">
                  {label}
                </label>
                <input
                  type={type}
                  value={(form as any)[key]}
                  onChange={(e) => setForm({ ...form, [key]: type === "number" ? parseFloat(e.target.value) || 0 : e.target.value })}
                  className="w-full bg-noir-700 border border-gold border-opacity-20 text-cream
                             px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold
                             focus:border-opacity-60 transition-all"
                />
              </div>
            ))}

            <div>
              <label className="block text-xs tracking-widest uppercase text-cream-muted mb-1.5 font-light">
                Unidad base
              </label>
              <select
                value={form.default_unit}
                onChange={(e) => setForm({ ...form, default_unit: e.target.value })}
                className="w-full bg-noir-700 border border-gold border-opacity-20 text-cream
                           px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold
                           focus:border-opacity-60 transition-all"
              >
                {UNITS.map((u) => <option key={u} value={u}>{u}</option>)}
              </select>
            </div>

            {error && <p className="text-red-400 text-xs tracking-wide">{error}</p>}

            <button
              onClick={handleSave}
              disabled={saving}
              className="w-full bg-gold text-noir text-xs tracking-widest uppercase font-medium
                         py-3 hover:bg-gold-light transition-all disabled:opacity-50"
            >
              {saving ? "Guardando..." : "Guardar"}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}