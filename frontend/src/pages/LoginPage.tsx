import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import api from "../services/api";

export default function LoginPage() {
  const [email, setEmail]       = useState("");
  const [password, setPassword] = useState("");
  const [error, setError]       = useState("");
  const [loading, setLoading]   = useState(false);
  const { login }               = useAuth();
  const navigate                = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const { data } = await api.post("/auth/login", { email, password });
      login(data.token);
      navigate("/dashboard");
    } catch {
      setError("Correo o contraseña incorrectos");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-noir flex flex-col items-center justify-center px-6">

      {/* Ornamento superior */}
      <div className="flex items-center gap-3 mb-10">
        <div className="h-px w-16 bg-gold opacity-40" />
        <span className="text-gold text-xs tracking-[0.3em] uppercase font-light">
          Bienvenida
        </span>
        <div className="h-px w-16 bg-gold opacity-40" />
      </div>

      {/* Logo / título */}
      <h1 className="font-display text-5xl text-gold text-center leading-tight mb-2">
        Angeles'S
      </h1>
      <p className="text-cream-muted text-xs tracking-[0.25em] uppercase font-light mb-12">
        Coffee & Bakery
      </p>

      {/* Formulario */}
      <form onSubmit={handleSubmit} className="w-full max-w-sm space-y-5">
        <div>
          <label className="block text-xs tracking-widest uppercase text-cream-muted mb-2 font-light">
            Correo
          </label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            className="w-full bg-noir-700 border border-gold border-opacity-30 text-cream
                       px-4 py-3 text-sm font-light tracking-wide
                       focus:outline-none focus:border-gold focus:border-opacity-80
                       transition-all duration-200 placeholder-stone-600"
            placeholder="tu@correo.com"
          />
        </div>

        <div>
          <label className="block text-xs tracking-widest uppercase text-cream-muted mb-2 font-light">
            Contraseña
          </label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            className="w-full bg-noir-700 border border-gold border-opacity-30 text-cream
                       px-4 py-3 text-sm font-light tracking-wide
                       focus:outline-none focus:border-gold focus:border-opacity-80
                       transition-all duration-200 placeholder-stone-600"
            placeholder="••••••••"
          />
        </div>

        {error && (
          <p className="text-red-400 text-xs tracking-wide text-center font-light">
            {error}
          </p>
        )}

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-gold text-noir font-medium text-sm tracking-[0.2em] uppercase
                     py-3.5 mt-2 transition-all duration-200
                     hover:bg-gold-light disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? "Ingresando..." : "Ingresar"}
        </button>
      </form>

      {/* Ornamento inferior */}
      <div className="flex items-center gap-3 mt-12">
        <div className="h-px w-12 bg-gold opacity-20" />
        <span className="text-gold opacity-40 text-xs">◆</span>
        <div className="h-px w-12 bg-gold opacity-20" />
      </div>
    </div>
  );
}