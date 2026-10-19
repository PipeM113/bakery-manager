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
          stroke="#E8DDD3" strokeWidth={1} strokeDasharray="4 3"
        />
      )}

      {/* revenue line */}
      <polyline
        points={polyline("revenue")}
        fill="none"
        stroke="#735523"
        strokeWidth={2}
        strokeLinejoin="round"
        strokeLinecap="round"
      />

      {/* net profit line */}
      <polyline
        points={polyline("net_profit")}
        fill="none"
        stroke="#1C1C1C"
        strokeWidth={2}
        strokeLinejoin="round"
        strokeLinecap="round"
      />

      {/* dots — revenue */}
      {data.map((d, i) => (
        <circle key={`rv-${i}`} cx={x(i)} cy={y(d.revenue)} r={3} fill="#735523" />
      ))}

      {/* dots — net profit */}
      {data.map((d, i) => (
        <circle key={`np-${i}`} cx={x(i)} cy={y(d.net_profit)} r={3} fill="#1C1C1C" />
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
            fill="#1C1C1C" fillOpacity={0.7}
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
    green:      "text-negro",
    "dark-green": "text-negro",
    red:        "text-peligro",
    yellow:     "text-oro",
    neutral:    "text-oro",
  };
  return (
    <div className={`bg-white border border-linea p-5 hover:border-oro transition-all ${large ? "col-span-2 md:col-span-1" : ""}`}>
      <p className={`font-display text-2xl ${colorMap[accent]} mb-1 leading-tight whitespace-nowrap tabular-nums`}>
        {value}
      </p>
      <p className="text-xs tracking-widest uppercase text-negro/70 font-light">{label}</p>
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
            <div className="h-px w-8 bg-oro/40" />
            <span className="text-oro text-xs tracking-[0.3em] uppercase font-light">Rentabilidad</span>
          </div>
          <h1 className="font-display text-4xl text-negro">Dashboard de ganancias</h1>
        </div>

        {/* month / year selector */}
        <div className="flex items-center gap-3">
          <select
            value={month}
            onChange={(e) => setMonth(Number(e.target.value))}
            className="border border-linea bg-white text-negro/70 text-sm px-3 py-2 focus:outline-none focus:border-oro"
          >
            {MONTH_NAMES.map((name, i) => (
              <option key={i + 1} value={i + 1}>{name}</option>
            ))}
          </select>
          <select
            value={year}
            onChange={(e) => setYear(Number(e.target.value))}
            className="border border-linea bg-white text-negro/70 text-sm px-3 py-2 focus:outline-none focus:border-oro"
          >
            {yearOptions.map((y) => (
              <option key={y} value={y}>{y}</option>
            ))}
          </select>
        </div>
      </div>

      {loading ? (
        <div className="min-h-64 flex items-center justify-center">
          <span className="text-oro text-sm tracking-widest uppercase font-light animate-pulse">Calculando...</span>
        </div>
      ) : (
        <>
          {/* KPI grid */}
          {metrics && (
            <div className="grid grid-cols-2 gap-4 mb-10 md:grid-cols-3 lg:grid-cols-6">
              <KpiCard label="Ingresos Totales"      value={clp(metrics.total_revenue)}             accent="green" />
              <KpiCard label="Costo de ingredientes e insumos"       value={clp(metrics.total_ingredients_cost)}    accent="red" />
              <KpiCard label="Costos Fijos"          value={clp(metrics.total_fixed_costs)}         accent="yellow" />
              <KpiCard label="Gastos Operacionales"  value={clp(metrics.total_operational_expenses)} accent="red" />
              <KpiCard label="Ganancia Neta"         value={clp(metrics.net_profit)}                accent="dark-green" large />
              <KpiCard label="Margen de Ganancia"    value={pct(metrics.profit_margin)}             accent={metrics.profit_margin >= 0 ? "green" : "red"} />
            </div>
          )}

          {/* Trend chart */}
          <div className="bg-white border border-linea p-6 mb-10">
            <div className="flex items-center gap-4 mb-5">
              <h2 className="font-display text-xl text-negro">Tendencia últimos 6 meses</h2>
              <div className="flex-1 h-px bg-lino" />
              <div className="flex items-center gap-4 text-xs text-negro/70 font-light">
                <span className="flex items-center gap-1.5">
                  <span className="inline-block w-4 border-t-2 border-oro" /> Ingresos
                </span>
                <span className="flex items-center gap-1.5">
                  <span className="inline-block w-4 border-t-2 border-negro" /> Ganancia Neta
                </span>
              </div>
            </div>
            {trends.length === 0 ? (
              <p className="text-negro/70 text-sm font-light text-center py-8">Sin datos de tendencia</p>
            ) : (
              <TrendChart data={trends} />
            )}
          </div>

          {/* Recipe ranking table */}
          <div className="bg-white border border-linea mb-10">
            <div className="flex items-center gap-4 px-6 py-5 border-b border-linea">
              <h2 className="font-display text-xl text-negro">Rendimiento por receta</h2>
              <div className="flex-1 h-px bg-lino" />
              <span className="text-xs text-negro/70 font-light tracking-widest uppercase">
                {MONTH_NAMES[month - 1]} {year}
              </span>
            </div>

            {recipes.length === 0 ? (
              <p className="text-negro/70 text-sm font-light text-center py-10">
                Sin ventas registradas para este período
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-linea">
                      {["Receta", "Unidades", "Ingreso", "Costo", "Ganancia", "Margen"].map((h) => (
                        <th
                          key={h}
                          className="px-5 py-3 text-left text-xs tracking-widest uppercase text-negro/70 font-light whitespace-nowrap"
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
                          className="border-b border-linea hover:bg-lino transition-colors"
                        >
                          <td className="px-5 py-3 text-negro font-light">{r.recipe_name}</td>
                          <td className="px-5 py-3 text-negro/70">{r.units_sold}</td>
                          <td className="px-5 py-3 text-negro/70">{clp(r.total_revenue)}</td>
                          <td className="px-5 py-3 text-negro/70">{clp(r.total_cost)}</td>
                          <td className={`px-5 py-3 font-light ${profit >= 0 ? "text-negro" : "text-peligro"}`}>
                            {clp(profit)}
                          </td>
                          <td className={`px-5 py-3 font-light ${positiveMargin ? "text-negro" : "text-peligro"}`}>
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
            <div className="bg-white border border-linea p-6">
              <div className="flex items-center gap-4 mb-5">
                <h2 className="font-display text-xl text-negro">Desglose mensual</h2>
                <div className="flex-1 h-px bg-lino" />
              </div>
              <div className="space-y-3">
                {[
                  { label: "Ingresos totales",       value: metrics.total_revenue,             color: "text-negro" },
                  { label: "− Costo de ingredientes e insumos",     value: -metrics.total_ingredients_cost,   color: "text-peligro" },
                  { label: "= Ganancia bruta",        value: metrics.gross_profit,              color: metrics.gross_profit >= 0 ? "text-negro" : "text-peligro", bold: true },
                  { label: "− Costos fijos",          value: -metrics.total_fixed_costs,        color: "text-oro" },
                  { label: "− Gastos operacionales",  value: -metrics.total_operational_expenses, color: "text-peligro" },
                  { label: "= Ganancia neta",         value: metrics.net_profit,                color: metrics.net_profit >= 0 ? "text-negro" : "text-peligro", bold: true },
                ].map(({ label, value, color, bold }) => (
                  <div
                    key={label}
                    className={`flex justify-between items-center py-2 border-b border-linea ${bold ? "border-linea pt-3" : ""}`}
                  >
                    <span className={`text-sm font-light ${bold ? "text-negro/70 font-normal" : "text-negro/70"}`}>
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
                    <div className="mt-4 pt-3 border-t border-linea">
                      <p className="text-xs text-negro/70 font-light mb-1 tracking-wide uppercase">Auditoría</p>
                      <p className="text-xs text-negro/70 font-mono">
                        Ganancia neta = {clp(metrics.total_revenue)} − ({clp(metrics.total_ingredients_cost)} + {clp(metrics.total_fixed_costs)} + {clp(metrics.total_operational_expenses)}) = {clp(computed)}
                        {" "}<span className={ok ? "text-negro" : "text-peligro"}>{ok ? "✓" : `⚠ diff ${clp(diff)}`}</span>
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
