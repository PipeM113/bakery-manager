import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { ToastContext, type ToastApi } from "../hooks/useToast";
import { dismissToast, durationFor, pushToast, type Toast, type ToastKind } from "../utils/toasts";


/** The notices of the whole app: success, error and info, at the top right, stacked. */
export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const counter = useRef(0);

  const show = useCallback((kind: ToastKind, title: string, detail?: string) => {
    counter.current += 1;
    const id = counter.current;
    setToasts((list) => pushToast(list, { id, kind, title, detail }));
  }, []);
  const close = useCallback((id: number) => setToasts((list) => dismissToast(list, id)), []);

  const api = useMemo<ToastApi>(() => ({
    success: (t, d) => show("success", t, d),
    error: (t, d) => show("error", t, d),
    info: (t, d) => show("info", t, d),
  }), [show]);

  return (
    <ToastContext.Provider value={api}>
      {children}
      <div
        data-testid="toasts"
        className="fixed top-4 right-4 left-4 sm:left-auto z-[70] flex flex-col gap-2 items-end pointer-events-none"
      >
        {toasts.map((t) => <ToastItem key={t.id} toast={t} onClose={close} />)}
      </div>
    </ToastContext.Provider>
  );
}

function Icon({ kind }: { kind: ToastKind }) {
  const path = kind === "success" ? "M5 12.5l4.5 4.5L19 7.5" : kind === "error" ? "M12 7v6m0 3.5v.01" : "M12 11v5m0-8.5v.01";
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="w-5 h-5 shrink-0 mt-0.5">
      {kind !== "success" && <circle cx="12" cy="12" r="9" />}
      <path d={path} />
    </svg>
  );
}

const style: Record<ToastKind, string> = {
  success: "bg-salvia border-oro text-negro",
  error: "bg-rosa border-peligro text-peligro",
  info: "bg-white border-linea text-negro",
};

function ToastItem({ toast: t, onClose }: { toast: Toast; onClose: (id: number) => void }) {
  const [paused, setPaused] = useState(false);

  // The time counts only while the mouse or the keyboard is not on the notice.
  useEffect(() => {
    if (paused) return;
    const timer = setTimeout(() => onClose(t.id), durationFor(t.kind));
    return () => clearTimeout(timer);
  }, [paused, t.id, t.kind, onClose]);

  return (
    <div
      role={t.kind === "error" ? "alert" : "status"}
      data-toast={t.kind}
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onFocus={() => setPaused(true)}
      onBlur={() => setPaused(false)}
      className={`pointer-events-auto w-full sm:w-[22rem] border-l-4 border shadow-lg px-4 py-3 flex gap-3 items-start ${style[t.kind]}`}
    >
      <Icon kind={t.kind} />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium break-words">{t.title}</p>
        {t.detail && <p className="text-xs font-light mt-0.5 break-words text-negro">{t.detail}</p>}
      </div>
      <button type="button" aria-label="Cerrar aviso" onClick={() => onClose(t.id)} className="text-negro/70 hover:text-negro text-lg leading-none -mt-0.5 shrink-0">×</button>
    </div>
  );
}
