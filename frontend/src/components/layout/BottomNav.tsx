import { useState } from "react";
import { NavLink, useLocation } from "react-router-dom";
import { useAuth } from "../../context/useAuth";

// Four sections fit in the bar at 360 px; the rest are one tap away in "Más".
const main = [
  { to: "/dashboard",   icon: "◉", label: "Dashboard"    },
  { to: "/ingredients", icon: "⚗", label: "Ingredientes" },
  { to: "/recipes",     icon: "✦", label: "Recetas"      },
  { to: "/sales",       icon: "⊕", label: "Ventas"       },
];
const more = [
  { to: "/costs",         icon: "◈", label: "Costos"       },
  { to: "/fixed-costs",   icon: "⊟", label: "Costos fijos" },
  { to: "/expenses",      icon: "◷", label: "Gastos"       },
  { to: "/profitability", icon: "◈", label: "Rentabilidad" },
  { to: "/business",      icon: "⌂", label: "Datos del negocio" },
  { to: "/account",       icon: "☺", label: "Mi cuenta" },
];

const item = (active: boolean) =>
  `flex-1 min-w-0 flex flex-col items-center justify-center gap-0.5 min-h-[56px] px-1 transition-all duration-200 ${
    active ? "text-oro" : "text-negro/70"
  }`;

export default function BottomNav() {
  const [open, setOpen] = useState(false);
  const { pathname } = useLocation();
  const { logout } = useAuth();
  const moreActive = more.some((l) => pathname.startsWith(l.to));

  return (
    <nav data-testid="bottom-nav" className="fixed bottom-0 left-0 right-0 z-50 bg-white border-t border-linea md:hidden shadow-sm">
      {open && (
        <div data-testid="nav-more-menu" className="border-b border-linea bg-white">
          {more.map(({ to, icon, label }) => (
            <NavLink
              key={to}
              to={to}
              onClick={() => setOpen(false)}
              className={({ isActive }) =>
                `flex items-center gap-3 px-6 min-h-[48px] text-sm tracking-wider ${isActive ? "text-oro" : "text-negro/70"}`
              }
            >
              <span className="text-lg leading-none">{icon}</span>
              <span className="font-light">{label}</span>
            </NavLink>
          ))}
          <button
            type="button"
            data-testid="nav-logout"
            onClick={logout}
            className="flex items-center gap-3 px-6 min-h-[48px] w-full text-left text-sm tracking-wider text-negro/70 border-t border-linea"
          >
            <span className="text-lg leading-none">⎋</span>
            <span className="font-light">Cerrar sesión</span>
          </button>
        </div>
      )}
      <div className="flex items-stretch h-16">
        {main.map(({ to, icon, label }) => (
          <NavLink key={to} to={to} onClick={() => setOpen(false)} className={({ isActive }) => item(isActive)}>
            <span className="text-lg leading-none">{icon}</span>
            <span className="text-[10px] font-light tracking-wider uppercase truncate max-w-full">{label}</span>
          </NavLink>
        ))}
        <button
          type="button"
          data-testid="nav-more"
          aria-expanded={open}
          onClick={() => setOpen((o) => !o)}
          className={item(open || moreActive)}
        >
          <span className="text-lg leading-none">⋯</span>
          <span className="text-[10px] font-light tracking-wider uppercase">Más</span>
        </button>
      </div>
    </nav>
  );
}
