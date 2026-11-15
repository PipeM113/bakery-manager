import { NavLink } from "react-router-dom";
import { useAuth } from "../../context/useAuth";
import { Icon, type IconName } from "./icons";
import logo from "../../assets/brand/logo-color.svg";

const links: { to: string; icon: IconName; label: string }[] = [
  { to: "/dashboard",   icon: "dashboard", label: "Dashboard"    },
  { to: "/ingredients", icon: "ingredients", label: "Ingredientes"  },
  { to: "/recipes",     icon: "recipes", label: "Recetas"      },
  { to: "/costs",       icon: "costs", label: "Costos"       },
  { to: "/fixed-costs", icon: "fixed", label: "Costos Fijos" },
  { to: "/sales",       icon: "sales", label: "Ventas"        },
  { to: "/expenses",       icon: "expenses", label: "Gastos"        },
  { to: "/profitability",  icon: "profit", label: "Rentabilidad"  },
  { to: "/business",       icon: "business", label: "Datos del negocio" },
  { to: "/account",        icon: "account", label: "Mi cuenta" },
];

export default function Sidebar() {
  const { logout } = useAuth();

  return (
    <aside
      data-testid="sidebar"
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
      <div className="flex-shrink-0 px-8 py-10 [@media(max-height:760px)]:py-5 border-b border-linea">
        <img src={logo} alt="Angeles Bakery" className="h-14 [@media(max-height:760px)]:h-11 w-auto" />
        <p className="text-negro/70 text-xs tracking-[0.25em] uppercase mt-3 [@media(max-height:760px)]:mt-2 font-light">
          Gestión
        </p>
      </div>

      <nav className="flex-1 overflow-y-auto px-4 py-8 [@media(max-height:760px)]:py-3 space-y-1 [@media(max-height:760px)]:space-y-0.5">
        {links.map(({ to, icon, label }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              `flex items-center gap-3 px-4 py-3 [@media(max-height:760px)]:py-2 focus-visible:outline-offset-[-2px] transition-all duration-200 text-sm tracking-wider ${
                isActive
                  ? "bg-crema text-oro border-l-2 border-oro"
                  : "text-negro/70 hover:text-negro hover:bg-lino"
              }`
            }
          >
            <Icon name={icon} />
            <span className="font-light">{label}</span>
          </NavLink>
        ))}
      </nav>

      <div className="flex-shrink-0 px-8 py-6 [@media(max-height:760px)]:py-4 border-t border-linea">
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