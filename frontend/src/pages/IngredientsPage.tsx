import { useEffect, useRef, useState } from "react";
import IngredientImportModal from "../components/IngredientImportModal";
import StockAdjustmentModal from "../components/StockAdjustmentModal";
import { ingredientService } from "../api/ingredientService";
import type { IIngredient, IngredientFormData } from "../types/ingredient";
import { emptyIngredientForm } from "../types/ingredient";
import type { IngredientKind } from "../types/ingredient";
import { todayInSantiago } from "../utils/businessDate";

const UNITS = ["gr", "kg", "ml", "lt", "und"];

const clp = (n: number) =>
  `$${Math.round(n).toLocaleString("es-CL")}`;

function PriceDisplay({ price, unit }: { price: number; unit: string }) {
  const conversions: Record<string, { label: string; factor: number }[]> = {
    gr:  [{ label: "por gr", factor: 1 }, { label: "por kg", factor: 1000 }],
    kg:  [{ label: "por kg", factor: 1 }, { label: "por gr", factor: 0.001 }],
    ml:  [{ label: "por ml", factor: 1 }, { label: "por lt", factor: 1000 }],
    lt:  [{ label: "por lt", factor: 1 }, { label: "por ml", factor: 0.001 }],
    und: [{ label: "por unidad", factor: 1 }],
  };
  const variants = conversions[unit] || [{ label: `por ${unit}`, factor: 1 }];
  return (
    <div className="flex gap-3 flex-wrap mt-1">
      {variants.map(({ label, factor }) => (
        <span key={label} className="text-negro/70 text-xs font-light">
          {clp(price * factor)} <span className="opacity-60">{label}</span>
        </span>
      ))}
    </div>
  );
}

export default function IngredientsPage() {
  const [ingredients, setIngredients] = useState<IIngredient[]>([]);
  const [search, setSearch]           = useState("");
  const [tab, setTab]                 = useState<IngredientKind>("ingredient");
  const [showForm, setShowForm]       = useState(false);
  const [editing, setEditing]         = useState<IIngredient | null>(null);
  const [form, setForm]               = useState<IngredientFormData>(emptyIngredientForm);
  const [loading, setLoading]         = useState(true);
  const [saving, setSaving]           = useState(false);
  const [error, setError]             = useState("");
  const [showImport, setShowImport]   = useState(false);
  const [adjusting, setAdjusting]     = useState<IIngredient | null>(null);
  const [exporting, setExporting]     = useState(false);
  const [successMsg, setSuccessMsg]   = useState("");
  const successTimer                  = useRef<ReturnType<typeof setTimeout> | null>(null);

  const load = () => {
    ingredientService.getAll()
      .then(setIngredients)
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);


  const showSuccess = (msg: string) => {
    setSuccessMsg(msg);
    if (successTimer.current) clearTimeout(successTimer.current);
    successTimer.current = setTimeout(() => setSuccessMsg(""), 3000);
  };

  const filtered = ingredients.filter((i) =>
    (i.kind ?? "ingredient") === tab && i.name.toLowerCase().includes(search.toLowerCase())
  );
  const singular = tab === "supply" ? "insumo" : "ingrediente";

  const openCreate = () => {
    setEditing(null);
    setForm({ ...emptyIngredientForm, kind: tab });
    setError("");
    setShowForm(true);
  };

  const openEdit = (i: IIngredient) => {
    setEditing(i);
    setForm({
      kind:            i.kind ?? "ingredient",
      name:            i.name,
      default_unit:    i.default_unit,
      package_size:    i.package_size,
      package_price:   i.package_price,
      stock_quantity:  i.stock_quantity,
      alert_threshold: i.alert_threshold,
      brand:        i.brand,
    });
    setError("");
    setShowForm(true);
  };

  const handleSave = async () => {
    setError("");
    if (!form.name.trim()) { setError("El nombre es requerido"); return; }
    if (!form.package_price || Number(form.package_price) <= 0) {
      setError("El precio del paquete es requerido"); return;
    }
    setSaving(true);
    try {
      if (editing) {
        await ingredientService.update(editing.id, form);
        showSuccess(`✅ ${form.kind === "supply" ? "Insumo" : "Ingrediente"} actualizado`);
      } else {
        await ingredientService.create(form);
        showSuccess(`✅ ${form.kind === "supply" ? "Insumo" : "Ingrediente"} creado`);
      }
      setShowForm(false);
      load();
    } catch (e: any) {
      setError(e.response?.data?.error || "Error al guardar");
    } finally {
      setSaving(false);
    }
  };

  const handleExport = async () => {
    setExporting(true);
    try {
      const blob = await ingredientService.exportExcel();
      const date = todayInSantiago().replace(/-/g, "");
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `ingredientes_${date}.xlsx`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } finally {
      setExporting(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("¿Eliminar este elemento?")) return;
    await ingredientService.delete(id);
    load();
  };

  if (loading) return (
    <div className="min-h-screen flex items-center justify-center">
      <span className="text-oro text-sm tracking-widest uppercase animate-pulse">Cargando...</span>
    </div>
  );

  const numericFields = [
    { label: "Tamaño del paquete",  key: "package_size",    placeholder: "ej. 1000 (gr/ml) · 1 (kg/lt)" },
    { label: "Precio del paquete",  key: "package_price",   placeholder: "ej. 2990"                     },
    { label: "Stock actual",        key: "stock_quantity",  placeholder: "ej. 500"                      },
    { label: "Umbral de alerta",    key: "alert_threshold", placeholder: "ej. 100"                      },
  ];

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">

      {/* Success toast */}
      {successMsg && (
        <div className="fixed top-4 right-4 z-50 bg-white border border-salvia text-negro px-4 py-3 text-sm font-light shadow-md">
          {successMsg}
        </div>
      )}

      {/* Header */}
      <div className="flex items-end justify-between mb-8">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <div className="h-px w-8 bg-oro/40" />
            <span className="text-oro text-xs tracking-[0.3em] uppercase font-light">Catálogo</span>
          </div>
          <h1 className="font-display text-4xl text-negro">Ingredientes e insumos</h1>
        </div>
        <div className="flex gap-2">
          <button onClick={handleExport} disabled={exporting}
            className="border border-linea text-negro/70 text-xs tracking-widest uppercase font-medium px-4 py-3
                       hover:border-oro hover:text-oro transition-all duration-200 disabled:opacity-50">
            {exporting ? "Exportando..." : "Exportar Excel"}
          </button>
          <button onClick={() => setShowImport(true)}
            className="border border-linea text-negro/70 text-xs tracking-widest uppercase font-medium px-4 py-3
                       hover:border-oro hover:text-oro transition-all duration-200">
            Importar Excel
          </button>
          <button onClick={openCreate}
            className="bg-negro text-crema text-xs tracking-widest uppercase font-medium px-5 py-3
                       hover:bg-black transition-all duration-200">
            + Nuevo
          </button>
        </div>
      </div>

      {/* Tipo: ingredientes o insumos (cajas, cartones) */}
      <div className="flex gap-2 mb-4">
        {([["ingredient", "Ingredientes"], ["supply", "Insumos (cajas, cartones)"]] as const).map(([k, label]) => (
          <button key={k} onClick={() => setTab(k)}
            className={`text-xs tracking-widest uppercase font-medium px-4 py-2 border transition-all duration-200
              ${tab === k ? "bg-negro text-crema border-oro" : "border-linea text-negro/70 hover:border-oro hover:text-oro"}`}>
            {label}
          </button>
        ))}
      </div>

      {/* Búsqueda */}
      <input
        type="text"
        placeholder={`Buscar ${singular}...`}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="w-full bg-white border border-linea text-negro text-sm
                   px-4 py-3 mb-6 font-light tracking-wide placeholder-negro/70
                   focus:outline-none focus:border-oro transition-all"
      />

      {/* Lista */}
      {filtered.length === 0 ? (
        <p className="text-center text-negro/70 text-sm font-light py-16 tracking-wide">
          {`No hay ${tab === "supply" ? "insumos" : "ingredientes"} registrados`}
        </p>
      ) : (
        <div className="space-y-2">
          {filtered.map((i) => {
            const lowStock = i.alert_threshold > 0 && i.stock_quantity < i.alert_threshold;
            return (
              <div key={i.id}
                className={`flex items-center justify-between p-4 bg-white border transition-all duration-200
                  ${lowStock
                    ? "border-peligro"
                    : "border-linea hover:border-oro"}`}
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-negro text-sm font-light truncate">{i.name}</p>
                    {lowStock && <span className="text-peligro text-xs shrink-0">◆ Stock bajo</span>}
                  </div>
                  <PriceDisplay price={i.price_per_unit} unit={i.default_unit} />
                  <p className="text-negro/70 text-xs font-light mt-0.5">
                    Envase: {i.package_size} {i.default_unit} · {clp(i.package_price)}
                    {i.brand && <span className="ml-2">· {i.brand}</span>}
                  </p>
                </div>
                <div className="flex items-center gap-3 ml-4">
                  <button onClick={() => setAdjusting(i)}
                    className="text-xs text-negro/70 hover:text-oro tracking-wider transition-colors">
                    Ajustar stock
                  </button>
                  <button onClick={() => openEdit(i)}
                    className="text-xs text-negro/70 hover:text-oro tracking-wider transition-colors">
                    Editar
                  </button>
                  <button onClick={() => handleDelete(i.id)}
                    className="text-xs text-negro/70 hover:text-peligro tracking-wider transition-colors">
                    Eliminar
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Stock adjustment (merma, corrección, otro) */}
      {adjusting && (
        <StockAdjustmentModal
          ingredient={ingredients.find((x) => x.id === adjusting.id) ?? adjusting}
          onClose={() => setAdjusting(null)}
          onSaved={load}
        />
      )}

      {/* Import Modal */}
      {showImport && (
        <IngredientImportModal
          onClose={() => setShowImport(false)}
          onSuccess={() => { setShowImport(false); load(); }}
        />
      )}

      {/* Modal */}
      {showForm && (
        <div
          className="fixed inset-0 z-50 bg-negro bg-opacity-40 flex items-end md:items-center justify-center p-4"
        >
          <div className="bg-white border border-linea w-full max-w-xs p-4 space-y-3 shadow-lg">
            <div className="flex items-center justify-between">
              <h2 className="font-display text-lg text-negro">
                {editing ? `Editar ${singular}` : `Nuevo ${singular}`}
              </h2>
              <button onClick={() => setShowForm(false)}
                className="text-negro/70 hover:text-negro text-lg leading-none">×</button>
            </div>

            <div>
              <label className="block text-xs tracking-widest uppercase text-negro/70 mb-1 font-light">Tipo</label>
              <select value={form.kind}
                onChange={(e) => setForm({ ...form, kind: e.target.value as IngredientKind })}
                className="w-full bg-crema border border-linea text-negro
                           px-3 py-1.5 text-sm font-light focus:outline-none focus:border-oro transition-all">
                <option value="ingredient">Ingrediente</option>
                <option value="supply">Insumo (caja, cartón)</option>
              </select>
            </div>

            <div>
              <label className="block text-xs tracking-widest uppercase text-negro/70 mb-1 font-light">Nombre</label>
              <input type="text" value={form.name} placeholder="ej. Manjar Colun"
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                className="w-full bg-crema border border-linea text-negro
                           px-3 py-1.5 text-sm font-light focus:outline-none focus:border-oro transition-all" />
            </div>

            <div>
              <label className="block text-xs tracking-widest uppercase text-negro/70 mb-1 font-light">Unidad base</label>
              <select value={form.default_unit}
                onChange={(e) => setForm({ ...form, default_unit: e.target.value })}
                className="w-full bg-crema border border-linea text-negro
                           px-3 py-1.5 text-sm font-light focus:outline-none focus:border-oro transition-all">
                {UNITS.map((u) => <option key={u} value={u}>{u}</option>)}
              </select>
            </div>

            {numericFields.map(({ label, key, placeholder }) => (
              <div key={key}>
                <label className="block text-xs tracking-widest uppercase text-negro/70 mb-1 font-light">{label}</label>
                <input type="number" value={(form as any)[key]} placeholder={placeholder}
                  onChange={(e) => setForm({ ...form, [key]: e.target.value === "" ? "" : parseFloat(e.target.value) })}
                  className="w-full bg-crema border border-linea text-negro
                             px-3 py-1.5 text-sm font-light focus:outline-none focus:border-oro transition-all" />
              </div>
            ))}

            <div>
              <label className="block text-xs tracking-widest uppercase text-negro/70 mb-1 font-light">Marca</label>
              <input type="text" value={form.brand} placeholder="ej. Colun, Lider, Santa Isabel"
                onChange={(e) => setForm({ ...form, brand: e.target.value })}
                className="w-full bg-crema border border-linea text-negro
                           px-3 py-1.5 text-sm font-light focus:outline-none focus:border-oro transition-all" />
            </div>

            {error && <p className="text-peligro text-xs tracking-wide">{error}</p>}

            <button onClick={handleSave} disabled={saving}
              className="w-full bg-negro text-crema text-xs tracking-widest uppercase font-medium
                         py-2 hover:bg-black transition-all disabled:opacity-50">
              {saving ? "Guardando..." : "Guardar"}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}