import { useEffect, useState } from "react";
import { businessService, ACCOUNT_TYPES, type BusinessForm } from "../api/businessService";

const empty: BusinessForm = { phone: "", bank: "", account_type: "", holder: "", email: "" };

const fieldClass = "w-full border border-linea bg-white px-3 py-2 text-sm font-light text-negro focus:outline-none focus:border-oro";
const labelClass = "block text-xs tracking-widest uppercase text-negro/70 mb-1.5 font-light";

export default function BusinessPage() {
  const [form, setForm]       = useState<BusinessForm>(empty);
  const [envMissing, setEnvMissing] = useState<string[]>([]);
  const [error, setError]     = useState("");
  const [saved, setSaved]     = useState(false);
  const [saving, setSaving]   = useState(false);

  useEffect(() => {
    businessService.get()
      .then(({ missing, ...rest }) => {
        setForm(rest);
        setEnvMissing(missing.filter((m) => m.startsWith("BIZ_")));
      })
      .catch(() => setError("Error cargando los datos del negocio"));
  }, []);

  const set = (key: keyof BusinessForm) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    setForm((f) => ({ ...f, [key]: e.target.value }));
    setSaved(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setSaving(true);
    try {
      const { missing, ...rest } = await businessService.save(form);
      setForm(rest);
      setEnvMissing(missing.filter((m) => m.startsWith("BIZ_")));
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    } catch (err) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setError(msg ?? "Error guardando los datos del negocio");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="max-w-2xl">
      <div className="mb-8">
        <h1 className="font-display text-3xl text-negro">Datos del negocio</h1>
        <p className="text-negro/70 text-sm font-light mt-2">
          Lo que sale en el PDF de las cotizaciones y puede cambiar: el teléfono y los datos para que el cliente transfiera. El nombre y el Instagram de la marca ya están fijos.
        </p>
      </div>

      <form data-testid="business-form" onSubmit={handleSubmit} className="bg-white border border-linea p-6 space-y-4">
        <div>
          <label htmlFor="biz-phone" className={labelClass}>Teléfono</label>
          <input id="biz-phone" type="tel" className={fieldClass} value={form.phone} onChange={set("phone")} maxLength={30} placeholder="+56 9 1234 5678" />
        </div>

        <div className="flex items-center gap-4 pt-2">
          <h2 className="font-display text-xl text-negro">Datos de transferencia</h2>
          <div className="flex-1 h-px bg-lino" />
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label htmlFor="biz-bank" className={labelClass}>Banco</label>
            <input id="biz-bank" className={fieldClass} value={form.bank} onChange={set("bank")} maxLength={60} />
          </div>
          <div>
            <label htmlFor="biz-type" className={labelClass}>Tipo de cuenta</label>
            <select id="biz-type" className={fieldClass} value={form.account_type} onChange={set("account_type")}>
              <option value="">Elige…</option>
              {ACCOUNT_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
            </select>
          </div>
        </div>
        <div>
          <label htmlFor="biz-holder" className={labelClass}>Titular de la cuenta</label>
          <input id="biz-holder" className={fieldClass} value={form.holder} onChange={set("holder")} maxLength={100} />
        </div>
        <div>
          <label htmlFor="biz-email" className={labelClass}>Correo electrónico</label>
          <input id="biz-email" type="email" className={fieldClass} value={form.email} onChange={set("email")} maxLength={120} />
        </div>

        <p className="text-negro/70 text-xs font-light">
          El RUT y el número de cuenta no se editan aquí: se configuran en Vercel (variables <code>BIZ_RUT</code> y{" "}
          <code>BIZ_ACCOUNT</code>) para que no queden guardados en la app.
        </p>
        {envMissing.length > 0 && (
          <p className="text-peligro text-xs">
            Falta configurar en Vercel: {envMissing.join(", ")}. Sin eso no se puede generar el PDF.
          </p>
        )}
        {error && <p role="alert" className="text-peligro text-xs">{error}</p>}
        {saved && <p role="status" className="text-negro text-sm bg-salvia/40 border border-salvia px-3 py-2">Datos guardados</p>}

        <div className="pt-2">
          <button
            type="submit"
            disabled={saving}
            className="bg-negro text-crema text-xs tracking-widest uppercase font-medium px-6 py-3 hover:bg-black transition-all disabled:opacity-50"
          >
            {saving ? "Guardando…" : "Guardar"}
          </button>
        </div>
      </form>
    </div>
  );
}
