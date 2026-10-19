import { useState, type ReactNode } from "react";
import { AuthContext } from "./useAuth";

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
