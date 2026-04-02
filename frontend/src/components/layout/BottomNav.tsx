import { NavLink } from "react-router-dom";

const links = [
  { to: "/dashboard",      icon: "◉", label: "Dashboard"   },
  { to: "/ingredients",    icon: "⚗", label: "Insumos"     },
  { to: "/recipes",        icon: "✦", label: "Recetas"     },
  { to: "/costs",          icon: "◈", label: "Costos"      },
  { to: "/fixed-costs",    icon: "⊟", label: "Fijos"       },
  { to: "/sales",          icon: "⊕", label: "Ventas"      },
  { to: "/expenses",       icon: "◷", label: "Gastos"      },
  { to: "/profitability",  icon: "◈", label: "Rentabilidad"},
];

export default function BottomNav() {
  return (
    <nav className="fixed bottom-0 left-0 right-0 z-50 bg-white border-t border-stone-200 md:hidden shadow-sm">
      <div className="flex justify-around items-center h-16">
        {links.map(({ to, icon, label }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              `flex flex-col items-center gap-0.5 px-4 py-2 transition-all duration-200 ${
                isActive ? "text-gold" : "text-stone-400"
              }`
            }
          >
            <span className="text-lg leading-none">{icon}</span>
            <span className="text-[10px] font-light tracking-widest uppercase">{label}</span>
          </NavLink>
        ))}
      </div>
    </nav>
  );
}