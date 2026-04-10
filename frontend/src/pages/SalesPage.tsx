import { useEffect, useRef, useState } from "react";
import { saleService } from "../api/saleService";
import { recipeService } from "../api/recipeService";
import { ingredientService } from "../api/ingredientService";
import { quotationService, type IQuotation } from "../api/quotationService";
import type { ISale } from "../types/sale";
import type { IRecipe } from "../types/recipe";
import type { IIngredient } from "../types/ingredient";
import api from "../services/api";

const clp = (n: number) => `$${Math.round(n).toLocaleString("es-CL")}`;

const today = () => new Date().toISOString().slice(0, 10);

const STATUS_LABEL: Record<IQuotation["status"], string> = {
  pending:   "Pendiente",
  confirmed: "Confirmada",
  cancelled: "Cancelada",
};

const STATUS_STYLE: Record<IQuotation["status"], string> = {
  pending:   "bg-amber-50 text-amber-700 border border-amber-200",
  confirmed: "bg-green-50 text-green-700 border border-green-200",
  cancelled: "bg-red-50 text-red-500 border border-red-200 line-through",
};

interface IngredientPreview {
  name: string;
  quantity_used: number;
  unit: string;
  available: number;
  insufficient: boolean;
}

type QuotationStatusFilter = IQuotation["status"] | "";

export default function SalesPage() {
  const [sales, setSales]           = useState<ISale[]>([]);
  const [recipes, setRecipes]       = useState<IRecipe[]>([]);
  const [ingredients, setIngredients] = useState<IIngredient[]>([]);
  const [loading, setLoading]       = useState(false);
  const [error, setError]           = useState("");

  // Form state
  const [recipeId, setRecipeId]     = useState("");
  const [quantity, setQuantity]     = useState("");
  const [unitPrice, setUnitPrice]   = useState("");
  const [recipeMargin, setRecipeMargin] = useState<number | null>(null);
  const [saleDate, setSaleDate]     = useState(today());
  const [notes, setNotes]           = useState("");
  const [formError, setFormError] = useState("");

  // Live stock check (computed reactively)
  const [stockIssues, setStockIssues] = useState<IngredientPreview[]>([]);

  // Confirmation step
  const [preview, setPreview]     = useState<IngredientPreview[] | null>(null);
  const [confirming, setConfirming] = useState(false);

  // Filters
  const [filterFrom, setFilterFrom] = useState("");
  const [filterTo, setFilterTo]     = useState("");

  const [successMsg, setSuccessMsg] = useState("");
  const successTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Quotations
  const [quotations, setQuotations]       = useState<IQuotation[]>([]);
  const [quotationFilter, setQuotationFilter] = useState<QuotationStatusFilter>("");
  const [quotationTab, setQuotationTab]   = useState<"quotations" | "sales">("quotations");

  const showSuccess = (msg: string) => {
    setSuccessMsg(msg);
    if (successTimer.current) clearTimeout(successTimer.current);
    successTimer.current = setTimeout(() => setSuccessMsg(""), 3000);
  };

  const loadSales = (from = filterFrom, to = filterTo) =>
    saleService.getAll(from || undefined, to || undefined)
      .then(setSales)
      .catch(() => setError("Error cargando ventas"));

  const loadQuotations = (status: QuotationStatusFilter = quotationFilter) =>
    quotationService.getAll(status || undefined)
      .then(setQuotations)
      .catch(() => setError("Error cargando cotizaciones"));

  useEffect(() => {
    recipeService.getAll().then(setRecipes).catch(() => setError("Error cargando recetas"));
    ingredientService.getAll().then(setIngredients).catch(() => {});
    loadSales();
    loadQuotations();
  }, []);

  // Recompute stock issues whenever recipe or quantity changes
  useEffect(() => {
    if (!recipeId || !ingredients.length) { setStockIssues([]); return; }
    const recipe = recipes.find((r) => r.id === recipeId);
    if (!recipe) { setStockIssues([]); return; }
    const qty = parseInt(quantity, 10) || 0;
    if (qty <= 0) { setStockIssues([]); return; }

    const stockMap = new Map(ingredients.map((i) => [i.id, i.stock_quantity]));
    const issues: IngredientPreview[] = recipe.ingredients
      .map((ing) => {
        const needed = ing.quantity * qty;
        const available = stockMap.get(ing.ingredient_id) ?? 0;
        return { name: ing.name, quantity_used: needed, unit: ing.unit, available, insufficient: available < needed };
      })
      .filter((p) => p.insufficient);
    setStockIssues(issues);
  }, [recipeId, quantity, ingredients, recipes]);

  // When recipe changes: fetch cost to pre-fill unit price with suggested price (margin applied, rounded to 500)
  const handleRecipeChange = async (id: string) => {
    setRecipeId(id);
    setStockIssues([]);
    if (!id) { setUnitPrice(""); setRecipeMargin(null); return; }
    try {
      const DEFAULT_MARGIN = 0.30;
      const { data } = await api.get<{ suggested_price: number; margin_pct: number }>(
        `/recipes/${id}/cost?margin_pct=${DEFAULT_MARGIN}`
      );
      setUnitPrice(String(data.suggested_price));
      setRecipeMargin(data.margin_pct);
    } catch {
      setUnitPrice("");
      setRecipeMargin(null);
    }
  };

  const handleFilter = () => loadSales(filterFrom, filterTo);

  const handleQuotationFilterChange = (status: QuotationStatusFilter) => {
    setQuotationFilter(status);
    loadQuotations(status);
  };

  const buildPreview = (): IngredientPreview[] => {
    const recipe = recipes.find((r) => r.id === recipeId);
    if (!recipe) return [];
    const qty = parseInt(quantity, 10) || 1;
    const stockMap = new Map(ingredients.map((i) => [i.id, i.stock_quantity]));
    return recipe.ingredients.map((ing) => {
      const needed = ing.quantity * qty;
      const available = stockMap.get(ing.ingredient_id) ?? 0;
      return { name: ing.name, quantity_used: needed, unit: ing.unit, available, insufficient: available < needed };
    });
  };

  const handleSubmitForm = (e: React.FormEvent) => {
    e.preventDefault();
    setFormError("");
    const qty = parseInt(quantity, 10);
    const price = parseFloat(unitPrice);
    if (!recipeId)               { setFormError("Selecciona una receta"); return; }
    if (!qty || qty <= 0)        { setFormError("Cantidad debe ser mayor a 0"); return; }
    if (isNaN(price) || price < 0) { setFormError("Precio debe ser mayor o igual a 0"); return; }
    if (stockIssues.length > 0)  { setFormError("No hay stock suficiente para algunos insumos"); return; }
    setPreview(buildPreview());
    setConfirming(true);
  };

  const handleConfirm = async () => {
    setLoading(true);
    setFormError("");
    try {
      const sale = await saleService.create({
        recipe_id: recipeId,
        quantity_sold: parseInt(quantity, 10),
        unit_price: parseFloat(unitPrice),
        sale_date: saleDate,
        notes,
      });
      setSales((prev) => [sale, ...prev]);
      ingredientService.getAll().then(setIngredients).catch(() => {});
      showSuccess("✅ Venta registrada");
      setRecipeId("");
      setQuantity("");
      setUnitPrice("");
      setRecipeMargin(null);
      setSaleDate(today());
      setNotes("");
      setPreview(null);
      setConfirming(false);
    } catch (err: unknown) {
      let msg = "Error registrando venta";
      if (err && typeof err === "object" && "response" in err) {
        const resp = (err as { response?: { data?: { error?: string } } }).response;
        if (resp?.data?.error) msg = resp.data.error;
      }
      setFormError(msg);
      setConfirming(false);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("¿Eliminar esta venta y revertir el stock?")) return;
    try {
      await saleService.delete(id);
      setSales((prev) => prev.filter((s) => s.id !== id));
      ingredientService.getAll().then(setIngredients).catch(() => {});
    } catch {
      setError("Error eliminando venta");
    }
  };

  const handleConfirmQuotation = async (id: string) => {
    try {
      await quotationService.confirm(id);
      setQuotations((prev) =>
        prev.map((q) => q.id === id ? { ...q, status: "confirmed" } : q)
      );
      showSuccess("✅ Cotización confirmada");
    } catch {
      setError("Error confirmando cotización");
    }
  };

  const handleCancelQuotation = async (id: string) => {
    try {
      await quotationService.cancel(id);
      setQuotations((prev) =>
        prev.map((q) => q.id === id ? { ...q, status: "cancelled" } : q)
      );
      showSuccess("Cotización cancelada");
    } catch {
      setError("Error cancelando cotización");
    }
  };

  const totalRevenue = sales.reduce((sum, s) => sum + s.total_price, 0);
  const hasStockIssue = stockIssues.length > 0;

  const displayedQuotations = quotationFilter
    ? quotations.filter((q) => q.status === quotationFilter)
    : quotations;

  return (
    <div className="px-6 py-10 max-w-4xl mx-auto md:max-w-none md:px-10">

      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <div className="h-px w-8 bg-gold opacity-40" />
          <span className="text-gold text-xs tracking-[0.3em] uppercase font-light">Registro</span>
        </div>
        <h1 className="font-display text-4xl text-stone-800">Ventas</h1>
      </div>

      {error && <p className="text-red-500 text-xs mb-4">{error}</p>}

      {/* Success toast */}
      {successMsg && (
        <div className="fixed top-4 right-4 z-50 bg-white border border-green-200 text-green-800 px-4 py-3 text-sm font-light shadow-md">
          {successMsg}
        </div>
      )}

      {/* Confirmation Modal */}
      {confirming && preview && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 px-4">
          <div className="bg-white border border-stone-200 p-8 max-w-md w-full">
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-display text-2xl text-stone-800">Confirmar Venta</h2>
              <button
                onClick={() => { setConfirming(false); setPreview(null); }}
                className="text-stone-400 hover:text-stone-800 text-xl leading-none"
              >×</button>
            </div>
            <p className="text-sm text-stone-500 font-light mb-4">
              Se descontará el siguiente stock:
            </p>
            <table className="w-full text-sm mb-6">
              <thead>
                <tr className="border-b border-stone-100">
                  <th className="text-left py-2 text-xs tracking-widest uppercase text-stone-400 font-light">Insumo</th>
                  <th className="text-right py-2 text-xs tracking-widest uppercase text-stone-400 font-light">Requerido</th>
                  <th className="text-right py-2 text-xs tracking-widest uppercase text-stone-400 font-light">Disponible</th>
                </tr>
              </thead>
              <tbody>
                {preview.map((p, i) => (
                  <tr key={i} className={`border-b border-stone-100 last:border-0 ${p.insufficient ? "bg-red-50" : ""}`}>
                    <td className={`py-2 font-light ${p.insufficient ? "text-red-700" : "text-stone-800"}`}>{p.name}</td>
                    <td className={`py-2 text-right font-light ${p.insufficient ? "text-red-700 font-medium" : "text-stone-600"}`}>
                      {p.quantity_used % 1 === 0 ? p.quantity_used : p.quantity_used.toFixed(3)} {p.unit}
                    </td>
                    <td className={`py-2 text-right font-light ${p.insufficient ? "text-red-500" : "text-stone-400"}`}>
                      {p.available % 1 === 0 ? p.available : p.available.toFixed(3)} {p.unit}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {preview.some((p) => p.insufficient) && (
              <p className="text-red-500 text-xs mb-4">
                Algunos insumos no tienen stock suficiente. La venta será rechazada.
              </p>
            )}
            <div className="flex gap-2 justify-end">
              <button
                onClick={() => { setConfirming(false); setPreview(null); }}
                className="text-xs tracking-widest uppercase font-medium px-5 py-2.5
                           border border-stone-200 text-stone-400 hover:border-stone-400 transition-all"
              >
                Cancelar
              </button>
              <button
                onClick={handleConfirm}
                disabled={loading}
                className="bg-gold text-white text-xs tracking-widest uppercase font-medium
                           px-5 py-2.5 hover:bg-gold-light transition-all disabled:opacity-50"
              >
                {loading ? "Registrando..." : "Confirmar"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Form */}
      <form onSubmit={handleSubmitForm} className="bg-white border border-stone-200 p-6 mb-8">
        <div className="flex items-center gap-4 mb-5">
          <h2 className="font-display text-xl text-stone-800">Nueva Venta</h2>
          <div className="flex-1 h-px bg-stone-200" />
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 mb-4">
          {/* Recipe */}
          <div className="sm:col-span-2 lg:col-span-1">
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Receta <span className="text-gold">*</span>
            </label>
            <select
              value={recipeId}
              onChange={(e) => handleRecipeChange(e.target.value)}
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            >
              <option value="">Seleccionar receta...</option>
              {recipes.map((r) => (
                <option key={r.id} value={r.id}>{r.name}</option>
              ))}
            </select>
          </div>

          {/* Quantity */}
          <div>
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Cantidad <span className="text-gold">*</span>
            </label>
            <input
              type="number"
              min={1}
              value={quantity}
              onChange={(e) => setQuantity(e.target.value)}
              placeholder="ej. 10"
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            />
          </div>

          {/* Unit price */}
          <div>
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Precio unitario (CLP) <span className="text-gold">*</span>
            </label>
            <input
              type="number"
              min={0}
              value={unitPrice}
              onChange={(e) => setUnitPrice(e.target.value)}
              placeholder="ej. 2500"
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            />
            {recipeMargin !== null && (
              <p className="text-stone-400 text-xs mt-1 font-light">
                Basado en margen {Math.round(recipeMargin * 100)}% — múltiplo de $500
              </p>
            )}
          </div>

          {/* Date */}
          <div>
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Fecha
            </label>
            <input
              type="date"
              value={saleDate}
              onChange={(e) => setSaleDate(e.target.value)}
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            />
          </div>

          {/* Notes */}
          <div className="sm:col-span-2">
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Notas
            </label>
            <input
              type="text"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Opcional"
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            />
          </div>
        </div>

        {/* Live stock warnings */}
        {hasStockIssue && (
          <div className="mb-3 p-3 bg-red-50 border border-red-200">
            <p className="text-red-600 text-xs font-medium mb-1.5 tracking-wide uppercase">
              Stock insuficiente
            </p>
            {stockIssues.map((issue, i) => (
              <p key={i} className="text-red-500 text-xs font-light">
                {issue.name}: disponible{" "}
                <span className="font-medium">{issue.available % 1 === 0 ? issue.available : issue.available.toFixed(3)} {issue.unit}</span>
                , requerido{" "}
                <span className="font-medium">{issue.quantity_used % 1 === 0 ? issue.quantity_used : issue.quantity_used.toFixed(3)} {issue.unit}</span>
              </p>
            ))}
          </div>
        )}

        {formError && (
          <p className="text-red-500 text-xs mb-3">{formError}</p>
        )}

        {quantity && unitPrice && (
          <p className="text-stone-400 text-xs mb-3 font-light">
            Total:{" "}
            <span className="text-gold font-medium">
              {clp((parseInt(quantity) || 0) * (parseFloat(unitPrice) || 0))}
            </span>
          </p>
        )}

        <button
          type="submit"
          disabled={hasStockIssue}
          className="bg-gold text-white text-xs tracking-widest uppercase font-medium
                     px-6 py-2.5 hover:bg-gold-light transition-all disabled:opacity-40 disabled:cursor-not-allowed"
        >
          Registrar Venta
        </button>
      </form>

      {/* Tabs: Cotizaciones / Historial de ventas */}
      <div className="flex border-b border-stone-200 mb-6">
        <button
          onClick={() => setQuotationTab("quotations")}
          className={`px-5 py-2.5 text-xs tracking-widest uppercase font-medium transition-all
            ${quotationTab === "quotations"
              ? "border-b-2 border-gold text-gold"
              : "text-stone-400 hover:text-stone-600"}`}
        >
          Cotizaciones
        </button>
        <button
          onClick={() => setQuotationTab("sales")}
          className={`px-5 py-2.5 text-xs tracking-widest uppercase font-medium transition-all
            ${quotationTab === "sales"
              ? "border-b-2 border-gold text-gold"
              : "text-stone-400 hover:text-stone-600"}`}
        >
          Historial de ventas
        </button>
      </div>

      {/* Quotations panel */}
      {quotationTab === "quotations" && (
        <div className="bg-white border border-stone-200">
          <div className="px-6 py-4 border-b border-stone-100 flex flex-col sm:flex-row sm:items-center gap-3">
            <h2 className="font-display text-xl text-stone-800 mr-auto">Cotizaciones</h2>
            {/* Status filter */}
            <div className="flex gap-1.5 flex-wrap">
              {(["", "pending", "confirmed", "cancelled"] as QuotationStatusFilter[]).map((s) => (
                <button
                  key={s}
                  onClick={() => handleQuotationFilterChange(s)}
                  className={`text-xs tracking-widest uppercase font-medium px-3 py-1.5 border transition-all
                    ${quotationFilter === s
                      ? "bg-gold text-white border-gold"
                      : "border-stone-200 text-stone-400 hover:border-gold hover:text-gold"}`}
                >
                  {s === "" ? "Todas" : STATUS_LABEL[s as IQuotation["status"]]}
                </button>
              ))}
            </div>
          </div>

          {displayedQuotations.length === 0 ? (
            <p className="text-stone-400 text-sm font-light text-center py-10">
              No hay cotizaciones
            </p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-stone-100">
                    <th className="text-left px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Fecha</th>
                    <th className="text-left px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Cliente</th>
                    <th className="text-left px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Producto</th>
                    <th className="text-right px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Monto</th>
                    <th className="text-center px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Estado</th>
                    <th className="px-6 py-3" />
                  </tr>
                </thead>
                <tbody>
                  {displayedQuotations.map((q) => (
                    <tr key={q.id} className="border-b border-stone-100 last:border-0 hover:bg-vanilla-100 transition-colors">
                      <td className="px-6 py-4 text-stone-600 font-light whitespace-nowrap">
                        {new Date(q.created_at).toLocaleDateString("es-CL")}
                      </td>
                      <td className={`px-6 py-4 font-light ${q.status === "cancelled" ? "line-through text-stone-400" : "text-stone-800"}`}>
                        {q.client_name}
                      </td>
                      <td className={`px-6 py-4 font-light ${q.status === "cancelled" ? "line-through text-stone-400" : "text-stone-600"}`}>
                        {q.recipe_name}
                      </td>
                      <td className={`px-6 py-4 text-right font-medium ${q.status === "cancelled" ? "text-stone-300" : "text-stone-800"}`}>
                        {clp(q.final_price)}
                      </td>
                      <td className="px-6 py-4 text-center">
                        <span className={`inline-block px-2 py-0.5 text-xs font-medium rounded-sm ${STATUS_STYLE[q.status]}`}>
                          {STATUS_LABEL[q.status]}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-right whitespace-nowrap">
                        {q.status === "pending" && (
                          <div className="flex gap-2 justify-end">
                            <button
                              onClick={() => handleConfirmQuotation(q.id)}
                              className="text-xs tracking-widest uppercase font-medium px-3 py-1.5
                                         bg-green-600 text-white hover:bg-green-700 transition-all"
                            >
                              Confirmar venta
                            </button>
                            <button
                              onClick={() => handleCancelQuotation(q.id)}
                              className="text-xs tracking-widest uppercase font-medium px-3 py-1.5
                                         border border-stone-200 text-stone-400 hover:border-red-400 hover:text-red-500 transition-all"
                            >
                              Cancelar
                            </button>
                          </div>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* Sales history panel */}
      {quotationTab === "sales" && (
        <div className="bg-white border border-stone-200">
          <div className="px-6 py-4 border-b border-stone-100 flex flex-col sm:flex-row sm:items-center gap-3">
            <h2 className="font-display text-xl text-stone-800 mr-auto">Historial</h2>

            {/* Date filters */}
            <div className="flex items-center gap-2 flex-wrap">
              <input
                type="date"
                value={filterFrom}
                onChange={(e) => setFilterFrom(e.target.value)}
                className="bg-vanilla-100 border border-stone-200 text-stone-700 px-3 py-1.5
                           text-xs font-light focus:outline-none focus:border-gold transition-all"
              />
              <span className="text-stone-400 text-xs">–</span>
              <input
                type="date"
                value={filterTo}
                onChange={(e) => setFilterTo(e.target.value)}
                className="bg-vanilla-100 border border-stone-200 text-stone-700 px-3 py-1.5
                           text-xs font-light focus:outline-none focus:border-gold transition-all"
              />
              <button
                onClick={handleFilter}
                className="text-xs tracking-widest uppercase font-medium px-4 py-1.5
                           border border-stone-200 text-stone-500 hover:border-gold hover:text-gold transition-all"
              >
                Filtrar
              </button>
            </div>

            {sales.length > 0 && (
              <span className="text-xs text-stone-400 font-light whitespace-nowrap">
                Total: <span className="text-gold font-medium">{clp(totalRevenue)}</span>
              </span>
            )}
          </div>

          {sales.length === 0 ? (
            <p className="text-stone-400 text-sm font-light text-center py-10">
              No hay ventas registradas
            </p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-stone-100">
                    <th className="text-left px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Fecha</th>
                    <th className="text-left px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Receta</th>
                    <th className="text-right px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Cant.</th>
                    <th className="text-right px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">P. Unit.</th>
                    <th className="text-right px-6 py-3 text-xs tracking-widest uppercase text-stone-400 font-light">Total</th>
                    <th className="px-6 py-3" />
                  </tr>
                </thead>
                <tbody>
                  {sales.map((s) => (
                    <tr key={s.id} className="border-b border-stone-100 last:border-0 hover:bg-vanilla-100 transition-colors">
                      <td className="px-6 py-4 text-stone-600 font-light whitespace-nowrap">{s.sale_date}</td>
                      <td className="px-6 py-4 text-stone-800 font-light">{s.recipe_name}</td>
                      <td className="px-6 py-4 text-right text-stone-600 font-light">{s.quantity_sold}</td>
                      <td className="px-6 py-4 text-right text-stone-600 font-light">{clp(s.unit_price)}</td>
                      <td className="px-6 py-4 text-right font-medium text-stone-800">{clp(s.total_price)}</td>
                      <td className="px-6 py-4 text-right">
                        <button
                          onClick={() => handleDelete(s.id)}
                          className="text-xs text-stone-400 hover:text-red-500 tracking-widest uppercase transition-colors"
                        >
                          Eliminar
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
