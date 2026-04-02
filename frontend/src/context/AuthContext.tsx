import { createContext, useContext, useState, type ReactNode } from "react";

interface AuthContextType {
  token: string | null;
  userName: string;
  login: (token: string) => void;
  logout: () => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | null>(null);

function decodeUserName(token: string): string {
  try {
    const base64 = token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
    const json = decodeURIComponent(
      atob(base64).split("").map((c) => "%" + c.charCodeAt(0).toString(16).padStart(2, "0")).join("")
    );
    const payload = JSON.parse(json);
    return payload.name ?? "";
  } catch {
    return "";
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const stored = localStorage.getItem("token");
  const [token, setToken] = useState<string | null>(stored);
  const [userName, setUserName] = useState<string>(stored ? decodeUserName(stored) : "");

  const login = (t: string) => {
    localStorage.setItem("token", t);
    setToken(t);
    setUserName(decodeUserName(t));
  };

  const logout = () => {
    localStorage.removeItem("token");
    setToken(null);
    setUserName("");
  };

  return (
    <AuthContext.Provider value={{ token, userName, login, logout, isAuthenticated: !!token }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
