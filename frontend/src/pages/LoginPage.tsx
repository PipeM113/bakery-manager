import logo from "../assets/brand/logo-color.svg";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../context/useAuth";
import api from "../services/api";
import { apiErrorMessage } from "../utils/apiError";

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
    } catch (err) {
      const res = (err as { response?: { status?: number; headers?: Record<string, string> } }).response;
      if (res?.status === 429) {
        // Locked after too many failed attempts: the backend says when to try again (Retry-After, in seconds).
        const seconds = Number(res.headers?.["retry-after"]);
        const minutes = Number.isFinite(seconds) && seconds > 0 ? Math.ceil(seconds / 60) : 0;
        setError(minutes
          ? `Demasiados intentos fallidos. Vuelve a intentar en ${minutes} ${minutes === 1 ? "minuto" : "minutos"}.`
          : apiErrorMessage(err, "Demasiados intentos fallidos. Espera unos minutos."));
      } else {
        setError("Correo o contraseña incorrectos");
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-crema flex flex-col items-center justify-center px-6">

      <div className="flex items-center gap-3 mb-10">
        <div className="h-px w-16 bg-oro/40" />
        <span className="text-oro text-xs tracking-[0.3em] uppercase font-light">Bienvenida</span>
        <div className="h-px w-16 bg-oro/40" />
      </div>

      <h1 className="mb-2">
        <img src={logo} alt="Angeles Bakery" className="h-20 w-auto mx-auto" />
      </h1>
      <p className="text-oro text-xs tracking-[0.25em] uppercase font-light mb-12">
        Gestión
      </p>

      <form onSubmit={handleSubmit} className="w-full max-w-sm space-y-5">
        <div>
          <label className="block text-xs tracking-widest uppercase text-negro/70 mb-2 font-light">
            Correo
          </label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            placeholder="tu@correo.com"
            className="w-full bg-white border border-linea text-negro
                       px-4 py-3 text-sm font-light tracking-wide
                       focus:outline-none focus:border-oro transition-all duration-200"
          />
        </div>

        <div>
          <label className="block text-xs tracking-widest uppercase text-negro/70 mb-2 font-light">
            Contraseña
          </label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            placeholder="••••••••"
            className="w-full bg-white border border-linea text-negro
                       px-4 py-3 text-sm font-light tracking-wide
                       focus:outline-none focus:border-oro transition-all duration-200"
          />
        </div>

        {error && (
          <p className="text-peligro text-xs tracking-wide text-center font-light">{error}</p>
        )}

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-negro text-crema font-medium text-sm tracking-[0.2em] uppercase
                     py-3.5 mt-2 transition-all duration-200
                     hover:bg-black disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? "Ingresando..." : "Ingresar"}
        </button>
      </form>

      <div className="flex items-center gap-3 mt-12">
        <div className="h-px w-12 bg-oro/20" />
        <span className="text-oro opacity-40 text-xs">◆</span>
        <div className="h-px w-12 bg-oro/20" />
      </div>
    </div>
  );
}