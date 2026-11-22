import { Outlet } from "react-router-dom";
import Sidebar from "./Sidebar";
import BottomNav from "./BottomNav";

export default function AppLayout() {
  return (
    <div className="min-h-screen">
      <a
        href="#contenido"
        className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-[80] focus:bg-negro focus:text-crema focus:px-4 focus:py-2 focus:text-xs focus:tracking-widest focus:uppercase"
      >
        Saltar al contenido
      </a>
      <Sidebar />
      <main
        id="contenido"
        tabIndex={-1}
        className="pb-20 md:pb-0 overflow-y-auto min-h-screen"
        style={{ marginLeft: "var(--sidebar-width, 0)" }}
      >
        <Outlet />
      </main>
      <BottomNav />
    </div>
  );
}