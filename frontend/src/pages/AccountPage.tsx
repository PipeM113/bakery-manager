import { useToast } from "../hooks/useToast";
import { useState } from "react";
import { authService } from "../api/authService";
import { apiErrorMessage } from "../utils/apiError";
import { useAuth } from "../context/useAuth";

const MIN_CHARS = 10;

const fieldClass = "w-full border border-linea bg-white px-3 py-2 text-sm font-light text-negro focus:outline-none focus:border-oro";
const labelClass = "block text-xs tracking-widest uppercase text-negro/70 mb-1.5 font-light";

export default function AccountPage() {
  const { userName } = useAuth();
  const [current, setCurrent] = useState("");
  const [next, setNext]       = useState("");
  const [repeat, setRepeat]   = useState("");
  const [error, setError]     = useState("");
  const toast = useToast();
  const [saving, setSaving]   = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!current) { setError("Escribe tu contraseña actual"); return; }
    if ([...next].length < MIN_CHARS) { setError(`La contraseña nueva debe tener al menos ${MIN_CHARS} caracteres`); return; }
    if (next !== repeat) { setError("Las contraseñas nuevas no coinciden"); return; }
    setError("");
    setSaving(true);
    try {
      await authService.changePassword(current, next);
      setCurrent(""); setNext(""); setRepeat("");
      toast.success("Contraseña cambiada", "Úsala la próxima vez que ingreses.");
    } catch (err) {
      setError(apiErrorMessage(err, "No se pudo cambiar la contraseña"));
      toast.error("No se pudo cambiar la contraseña", apiErrorMessage(err, ""));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="px-6 py-10 md:px-10">
    <div className="max-w-xl">
      <div className="mb-8">
        <h1 className="font-display text-3xl text-negro">Mi cuenta</h1>
        <p className="text-negro/70 text-sm font-light mt-2">
          {userName ? `${userName}, aquí` : "Aquí"} cambias tu contraseña. Las sesiones que ya están abiertas siguen valiendo hasta
          que vencen (24 horas).
        </p>
      </div>

      <form data-testid="account-form" onSubmit={handleSubmit} className="bg-white border border-linea p-6 space-y-4">
        <div>
          <label htmlFor="pw-current" className={labelClass}>Contraseña actual</label>
          <input id="pw-current" type="password" autoComplete="current-password" className={fieldClass}
            value={current} onChange={(e) => { setCurrent(e.target.value); }} />
        </div>
        <div>
          <label htmlFor="pw-new" className={labelClass}>Nueva contraseña</label>
          <input id="pw-new" type="password" autoComplete="new-password" className={fieldClass}
            value={next} onChange={(e) => { setNext(e.target.value); }} />
          <p className="text-negro/70 text-xs font-light mt-1">Al menos 10 caracteres y distinta de la actual.</p>
        </div>
        <div>
          <label htmlFor="pw-repeat" className={labelClass}>Repite la nueva contraseña</label>
          <input id="pw-repeat" type="password" autoComplete="new-password" className={fieldClass}
            value={repeat} onChange={(e) => { setRepeat(e.target.value); }} />
        </div>

        {error && <p role="alert" className="text-peligro text-xs">{error}</p>}

        <div className="pt-2">
          <button type="submit" disabled={saving}
            className="bg-negro text-crema text-xs tracking-widest uppercase font-medium px-6 py-3 hover:bg-black transition-[color,background-color,border-color,opacity] disabled:opacity-50">
            {saving ? "Guardando…" : "Cambiar contraseña"}
          </button>
        </div>
      </form>
    </div>
    </div>
  );
}
