import { Outlet } from "react-router-dom";
import Sidebar from "./Sidebar";
import BottomNav from "./BottomNav";

export default function AppLayout() {
  return (
    <div className="min-h-screen">
      <Sidebar />
      <main
        className="pb-20 md:pb-0 overflow-y-auto min-h-screen"
        style={{ marginLeft: "var(--sidebar-width, 0)" }}
      >
        <Outlet />
      </main>
      <BottomNav />
    </div>
  );
}