import { Routes, Route, Navigate } from "react-router-dom";
import { useAuth } from "./context/AuthContext";
import AppLayout from "./components/layout/AppLayout";
import LoginPage from "./pages/LoginPage";
import IngredientsPage from "./pages/IngredientsPage";
import RecipesPage from "./pages/RecipesPage";
import CostsPage from "./pages/CostsPage";
import DashboardPage from "./pages/DashboardPage";
import FixedCostsPage from "./pages/FixedCostsPage";
import SalesPage from "./pages/SalesPage";
import ExpensesPage from "./pages/ExpensesPage";
import ProfitabilityPage from "./pages/ProfitabilityPage";

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth();
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <AppLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard"   element={<DashboardPage />} />
        <Route path="ingredients" element={<IngredientsPage />} />
        <Route path="recipes"     element={<RecipesPage />} />
        <Route path="costs"        element={<CostsPage />} />
        <Route path="fixed-costs"  element={<FixedCostsPage />} />
        <Route path="sales"        element={<SalesPage />} />
        <Route path="expenses"        element={<ExpensesPage />} />
        <Route path="profitability"   element={<ProfitabilityPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  );
}