import { useEffect, useState, useMemo } from "react";
import {
  analyticsService,
  type MonthlyMetrics,
  type RecipeMetrics,
  type TrendPoint,
} from "../api/analyticsService";

// ── helpers ──────────────────────────────────────────────────────────────────

function clp(value: number) {
  return new Intl.NumberFormat("es-CL", {
    style: "currency",
    currency: "CLP",
    maximumFractionDigits: 0,
  }).format(value);
}

function pct(value: number) {
  return `${value.toFixed(1)}%`;
}

const MONTH_NAMES = [
  "Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio",
  "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre",
];

// ── SVG trend chart ───────────────────────────────────────────────────────────

interface TrendChartProps {
  data: TrendPoint[];
}

function TrendChart({ data }: TrendChartProps) {
  const W = 600;
  const H = 160;
  const PAD = { top: 16, right: 16, bottom: 28, left: 8 };
  const chartW = W - PAD.left - PAD.right;
  const chartH = H - PAD.top - PAD.bottom;

  const allValues = data.flatMap((d) => [d.revenue, d.net_profit]);
  const minV = Math.min(...allValues, 0);
  const maxV = Math.max(...allValues, 1);
  const range = maxV - minV || 1;

  const xStep = data.length > 1 ? chartW / (data.length - 1) : chartW;

  function x(i: number) {
    return PAD.left + (data.length > 1 ? i * xStep : chartW / 2);
  }
  function y(v: number) {
    return PAD.top + chartH - ((v - minV) / range) * chartH;
  }

  function polyline(key: "revenue" | "net_profit") {
    return data.map((d, i) => `${x(i)},${y(d[key])}`).join(" ");
  }

  // zero baseline (if visible)
  const zeroY = y(0);
  const showZero = zeroY >= PAD.top && zeroY <= PAD.top + chartH;

  return (
    <svg
      viewBox={`0 0 ${W} ${H}`}
      className="w-full"
      style={{ height: H }}
      aria-label="Tendencia ingresos y ganancia neta"
    >
      {/* zero line */}
      {showZero && (
        <line
          x1={PAD.left} y1={zeroY}
          x2={W - PAD.right} y2={zeroY}
          stroke="#e7e5e4" strokeWidth={1} strokeDasharray="4 3"
        />
      )}

      {/* revenue line */}
      <polyline
        points={polyline("revenue")}
        fill="none"
        stroke="#b5954a"
        strokeWidth={2}
        strokeLinejoin="round"
        strokeLinecap="round"
      />

      {/* net profit line */}
      <polyline
        points={polyline("net_profit")}
        fill="none"
        stroke="#4a7c59"
        strokeWidth={2}
        strokeLinejoin="round"
        strokeLinecap="round"
      />

      {/* dots — revenue */}
      {data.map((d, i) => (
        <circle key={`rv-${i}`} cx={x(i)} cy={y(d.revenue)} r={3} fill="#b5954a" />
      ))}

      {/* dots — net profit */}
      {data.map((d, i) => (
        <circle key={`np-${i}`} cx={x(i)} cy={y(d.net_profit)} r={3} fill="#4a7c59" />
      ))}

      {/* x-axis labels */}
      {data.map((d, i) => {
        const label = d.date.slice(5); // "MM"
        const monthIdx = parseInt(label, 10) - 1;
        const short = MONTH_NAMES[monthIdx]?.slice(0, 3) ?? label;
        return (
          <text
            key={`lbl-${i}`}
            x={x(i)}
            y={H - 4}
            textAnchor="middle"
            fontSize={10}
            fill="#a8a29e"
          >
            {short}
          </text>
        );
      })}
    </svg>
  );
}

// ── KPI card ──────────────────────────────────────────────────────────────────

interface KpiCardProps {
  label: string;
  value: string;
  accent?: "green" | "red" | "yellow" | "dark-green" | "neutral";
  large?: boolean;
}

function KpiCard({ label, value, accent = "neutral", large = false }: KpiCardProps) {
  const colorMap: Record<string, string> = {
    green:      "text-emerald-600",
    "dark-green": "text-emerald-800",
    red:        "text-red-500",
    yellow:     "text-amber-500",
    neutral:    "text-gold",
  };
  return (
    <div className={`bg-white border border-stone-200 p-5 hover:border-gold transition-all ${large ? "col-span-2 md:col-span-1" : ""}`}>
      <p className={`font-display ${large ? "text-4xl" : "text-2xl"} ${colorMap[accent]} mb-1 leading-tight truncate`}>
        {value}
      </p>
      <p className="text-xs tracking-widest uppercase text-stone-400 font-light">{label}</p>
    </div>
  );
}

// ── main page ─────────────────────────────────────────────────────────────────

export default function ProfitabilityPage() {
  const now = new Date();
  const [month, setMonth] = useState(now.getMonth() + 1);
  const [year, setYear]   = useState(now.getFullYear());

  const [metrics,  setMetrics]  = useState<MonthlyMetrics | null>(null);
  const [recipes,  setRecipes]  = useState<RecipeMetrics[]>([]);
  const [trends,   setTrends]   = useState<TrendPoint[]>([]);
  const [loading,  setLoading]  = useState(true);

  // year options: current year and 2 years back
  const yearOptions = useMemo(() => {
    const y = now.getFullYear();
    return [y - 2, y - 1, y];
  }, []);

  useEffect(() => {
    setLoading(true);
    Promise.all([
      analyticsService.getMonthlyMetrics(month, year),
      analyticsService.getRecipeMetrics(month, year),
      analyticsService.getTrends(6),
    ])
      .then(([m, r, t]) => {
        console.log('Monthly metrics:', m);
        setMetrics(m);
        setRecipes(r ?? []);
        setTrends(t ?? []);
      })
      .finally(() => setLoading(false));
  }, [month, year]);

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">

      {/* header */}
      <div className="mb-8 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div>
          <div className="flex items-center gap-3 mb-3">
            <div className="h-px w-8 bg-gold opacity-40" />
            <span className="text-gold text-xs tracking-[0.3em] uppercase font-light">Rentabilidad</span>
          </div>
          <h1 className="font-display text-4xl text-stone-800">Dashboard de ganancias</h1>
        </div>

        {/* month / year selector */}
        <div className="flex items-center gap-3">
          <select
            value={month}
            onChange={(e) => setMonth(Number(e.target.value))}
            className="border border-stone-200 bg-white text-stone-700 text-sm px-3 py-2 focus:outline-none focus:border-gold"
          >
            {MONTH_NAMES.map((name, i) => (
              <option key={i + 1} value={i + 1}>{name}</option>
            ))}
          </select>
          <select
            value={year}
            onChange={(e) => setYear(Number(e.target.value))}
            className="border border-stone-200 bg-white text-stone-700 text-sm px-3 py-2 focus:outline-none focus:border-gold"
          >
            {yearOptions.map((y) => (
              <option key={y} value={y}>{y}</option>
            ))}
          </select>
        </div>
      </div>

      {loading ? (
        <div className="min-h-64 flex items-center justify-center">
          <span className="text-gold text-sm tracking-widest uppercase font-light animate-pulse">Calculando...</span>
        </div>
      ) : (
        <>
          {/* KPI grid */}
          {metrics && (
            <div className="grid grid-cols-2 gap-4 mb-10 md:grid-cols-3 lg:grid-cols-6">
              <KpiCard label="Ingresos Totales"      value={clp(metrics.total_revenue)}             accent="green" />
              <KpiCard label="Costo Insumos"         value={clp(metrics.total_ingredients_cost)}    accent="red" />
              <KpiCard label="Costos Fijos"          value={clp(metrics.total_fixed_costs)}         accent="yellow" />
              <KpiCard label="Gastos Operacionales"  value={clp(metrics.total_operational_expenses)} accent="red" />
              <KpiCard label="Ganancia Neta"         value={clp(metrics.net_profit)}                accent="dark-green" large />
              <KpiCard label="Margen de Ganancia"    value={pct(metrics.profit_margin)}             accent={metrics.profit_margin >= 0 ? "green" : "red"} />
            </div>
          )}

          {/* Trend chart */}
          <div className="bg-white border border-stone-200 p-6 mb-10">
            <div className="flex items-center gap-4 mb-5">
              <h2 className="font-display text-xl text-stone-800">Tendencia últimos 6 meses</h2>
              <div className="flex-1 h-px bg-stone-200" />
              <div className="flex items-center gap-4 text-xs text-stone-400 font-light">
                <span className="flex items-center gap-1.5">
                  <span className="inline-block w-4 h-0.5 bg-gold" /> Ingresos
                </span>
                <span className="flex items-center gap-1.5">
                  <span className="inline-block w-4 h-0.5 bg-emerald-700" /> Ganancia Neta
                </span>
              </div>
            </div>
            {trends.length === 0 ? (
              <p className="text-stone-400 text-sm font-light text-center py-8">Sin datos de tendencia</p>
            ) : (
              <TrendChart data={trends} />
            )}
          </div>

          {/* Recipe ranking table */}
          <div className="bg-white border border-stone-200 mb-10">
            <div className="flex items-center gap-4 px-6 py-5 border-b border-stone-100">
              <h2 className="font-display text-xl text-stone-800">Rendimiento por receta</h2>
              <div className="flex-1 h-px bg-stone-200" />
              <span className="text-xs text-stone-400 font-light tracking-widest uppercase">
                {MONTH_NAMES[month - 1]} {year}
              </span>
            </div>

            {recipes.length === 0 ? (
              <p className="text-stone-400 text-sm font-light text-center py-10">
                Sin ventas registradas para este período
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-stone-100">
                      {["Receta", "Unidades", "Ingreso", "Costo", "Ganancia", "Margen"].map((h) => (
                        <th
                          key={h}
                          className="px-5 py-3 text-left text-xs tracking-widest uppercase text-stone-400 font-light whitespace-nowrap"
                        >
                          {h}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {recipes.map((r) => {
                      const profit = r.total_revenue - r.total_cost;
                      const positiveMargin = r.gross_margin >= 0;
                      return (
                        <tr
                          key={r.recipe_id}
                          className="border-b border-stone-50 hover:bg-vanilla-50 transition-colors"
                        >
                          <td className="px-5 py-3 text-stone-800 font-light">{r.recipe_name}</td>
                          <td className="px-5 py-3 text-stone-600">{r.units_sold}</td>
                          <td className="px-5 py-3 text-stone-600">{clp(r.total_revenue)}</td>
                          <td className="px-5 py-3 text-stone-600">{clp(r.total_cost)}</td>
                          <td className={`px-5 py-3 font-light ${profit >= 0 ? "text-emerald-600" : "text-red-500"}`}>
                            {clp(profit)}
                          </td>
                          <td className={`px-5 py-3 font-light ${positiveMargin ? "text-emerald-600" : "text-red-500"}`}>
                            {pct(r.gross_margin)}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Cost breakdown summary */}
          {metrics && (
            <div className="bg-white border border-stone-200 p-6">
              <div className="flex items-center gap-4 mb-5">
                <h2 className="font-display text-xl text-stone-800">Desglose mensual</h2>
                <div className="flex-1 h-px bg-stone-200" />
              </div>
              <div className="space-y-3">
                {[
                  { label: "Ingresos totales",       value: metrics.total_revenue,             color: "text-emerald-600" },
                  { label: "− Costo de insumos",     value: -metrics.total_ingredients_cost,   color: "text-red-500" },
                  { label: "= Ganancia bruta",        value: metrics.gross_profit,              color: metrics.gross_profit >= 0 ? "text-stone-800" : "text-red-500", bold: true },
                  { label: "− Costos fijos",          value: -metrics.total_fixed_costs,        color: "text-amber-500" },
                  { label: "− Gastos operacionales",  value: -metrics.total_operational_expenses, color: "text-red-400" },
                  { label: "= Ganancia neta",         value: metrics.net_profit,                color: metrics.net_profit >= 0 ? "text-emerald-700" : "text-red-600", bold: true },
                ].map(({ label, value, color, bold }) => (
                  <div
                    key={label}
                    className={`flex justify-between items-center py-2 border-b border-stone-50 ${bold ? "border-stone-200 pt-3" : ""}`}
                  >
                    <span className={`text-sm font-light ${bold ? "text-stone-700 font-normal" : "text-stone-500"}`}>
                      {label}
                    </span>
                    <span className={`text-sm ${color} ${bold ? "font-medium" : "font-light"}`}>
                      {clp(Math.abs(value))}
                    </span>
                  </div>
                ))}
                {/* Audit row */}
                {(() => {
                  const allCosts = metrics.total_ingredients_cost + metrics.total_fixed_costs + metrics.total_operational_expenses;
                  const computed = metrics.total_revenue - allCosts;
                  const diff = Math.abs(computed - metrics.net_profit);
                  const ok = diff < 0.01;
                  return (
                    <div className="mt-4 pt-3 border-t border-stone-100">
                      <p className="text-xs text-stone-400 font-light mb-1 tracking-wide uppercase">Auditoría</p>
                      <p className="text-xs text-stone-400 font-mono">
                        Ganancia neta = {clp(metrics.total_revenue)} − ({clp(metrics.total_ingredients_cost)} + {clp(metrics.total_fixed_costs)} + {clp(metrics.total_operational_expenses)}) = {clp(computed)}
                        {" "}<span className={ok ? "text-emerald-500" : "text-red-500"}>{ok ? "✓" : `⚠ diff ${clp(diff)}`}</span>
                      </p>
                    </div>
                  );
                })()}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
