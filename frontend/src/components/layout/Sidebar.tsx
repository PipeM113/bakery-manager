import { NavLink } from "react-router-dom";
import { useAuth } from "../../context/AuthContext";

const links = [
  { to: "/dashboard",   icon: "◉", label: "Dashboard"    },
  { to: "/ingredients", icon: "⚗", label: "Insumos"      },
  { to: "/recipes",     icon: "✦", label: "Recetas"      },
  { to: "/costs",       icon: "◈", label: "Costos"       },
  { to: "/fixed-costs", icon: "⊟", label: "Costos Fijos" },
  { to: "/sales",       icon: "⊕", label: "Ventas"        },
  { to: "/expenses",       icon: "◷", label: "Gastos"        },
  { to: "/profitability",  icon: "◈", label: "Rentabilidad"  },
];

export default function Sidebar() {
  const { logout } = useAuth();

  return (
    <aside
      className="hidden md:flex flex-col bg-white border-r border-stone-200"
      style={{
        position: "fixed",
        top: 0,
        left: 0,
        width: 280,
        height: "100vh",
        overflow: "hidden",
        zIndex: 100,
      }}
    >
      <div className="flex-shrink-0 px-8 py-10 border-b border-stone-100">
        <h1 className="font-display text-2xl text-gold leading-tight">Angeles'S</h1>
        <p className="text-stone-400 text-xs tracking-widest uppercase mt-1 font-light">
          Coffee & Bakery
        </p>
      </div>

      <nav className="flex-1 overflow-y-auto px-4 py-8 space-y-1">
        {links.map(({ to, icon, label }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              `flex items-center gap-3 px-4 py-3 transition-all duration-200 text-sm tracking-wider ${
                isActive
                  ? "bg-vanilla-100 text-gold border-l-2 border-gold"
                  : "text-stone-400 hover:text-stone-800 hover:bg-stone-100"
              }`
            }
          >
            <span className="text-base">{icon}</span>
            <span className="font-light">{label}</span>
          </NavLink>
        ))}
      </nav>

      <div className="flex-shrink-0 px-8 py-6 border-t border-stone-100">
        <button
          onClick={logout}
          className="text-xs text-stone-400 hover:text-gold tracking-widest uppercase transition-colors"
        >
          Cerrar sesión
        </button>
      </div>
    </aside>
  );
}