import { NavLink } from "react-router-dom";
import { useAuth } from "../../context/AuthContext";
import logo from "../../assets/brand/logo-color.svg";

const links = [
  { to: "/dashboard",   icon: "◉", label: "Dashboard"    },
  { to: "/ingredients", icon: "⚗", label: "Ingredientes"  },
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
      className="hidden md:flex flex-col bg-white border-r border-linea"
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
      <div className="flex-shrink-0 px-8 py-10 border-b border-linea">
        <img src={logo} alt="Angeles Bakery" className="h-14 w-auto" />
        <p className="text-negro/70 text-xs tracking-[0.25em] uppercase mt-3 font-light">
          Gestión
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
                  ? "bg-crema text-oro border-l-2 border-oro"
                  : "text-negro/70 hover:text-negro hover:bg-lino"
              }`
            }
          >
            <span className="text-base">{icon}</span>
            <span className="font-light">{label}</span>
          </NavLink>
        ))}
      </nav>

      <div className="flex-shrink-0 px-8 py-6 border-t border-linea">
        <button
          onClick={logout}
          className="text-xs text-negro/70 hover:text-oro tracking-widest uppercase transition-colors"
        >
          Cerrar sesión
        </button>
      </div>
    </aside>
  );
}