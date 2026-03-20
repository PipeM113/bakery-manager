import { NavLink } from "react-router-dom";
import { useAuth } from "../../context/AuthContext";

const links = [
  { to: "/dashboard",   icon: "◉", label: "Dashboard" },
  { to: "/ingredients", icon: "⚗", label: "Insumos" },
  { to: "/recipes",     icon: "✦", label: "Recetas" },
  { to: "/costs",       icon: "◈", label: "Costos" },
];

export default function Sidebar() {
  const { logout } = useAuth();

  return (
    <aside className="hidden md:flex flex-col w-64 min-h-screen bg-noir-800 border-r border-gold border-opacity-20">
      <div className="px-8 py-10 border-b border-gold border-opacity-20">
        <h1 className="font-display text-2xl text-gold leading-tight">Angeles'S</h1>
        <p className="text-cream-muted text-xs tracking-widest uppercase mt-1 font-light">
          Coffee & Bakery
        </p>
      </div>

      <nav className="flex-1 px-4 py-8 space-y-1">
        {links.map(({ to, icon, label }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              `flex items-center gap-3 px-4 py-3 rounded transition-all duration-200 text-sm tracking-wider ${
                isActive
                  ? "bg-gold bg-opacity-10 text-gold border-l-2 border-gold"
                  : "text-cream-muted hover:text-cream hover:bg-white hover:bg-opacity-5"
              }`
            }
          >
            <span className="text-base">{icon}</span>
            <span className="font-light">{label}</span>
          </NavLink>
        ))}
      </nav>

      <div className="px-8 py-6 border-t border-gold border-opacity-20">
        <button
          onClick={logout}
          className="text-xs text-cream-muted hover:text-gold tracking-widest uppercase transition-colors"
        >
          Cerrar sesión
        </button>
      </div>
    </aside>
  );
}