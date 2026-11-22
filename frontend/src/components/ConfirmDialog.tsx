import { useCallback, useEffect, useId, useRef, useState, type ReactNode } from "react";
import { ConfirmContext, type Ask, type ConfirmOptions } from "../hooks/useConfirm";

/** Replaces the question box of the browser: the same question with the colors of the app, named after what it affects. */
export function ConfirmProvider({ children }: { children: ReactNode }) {
  const [pending, setPending] = useState<(ConfirmOptions & { resolve: (ok: boolean) => void }) | null>(null);

  const ask = useCallback<Ask>((options) => new Promise<boolean>((resolve) => setPending({ ...options, resolve })), []);
  const answer = (ok: boolean) => {
    pending?.resolve(ok);
    setPending(null);
  };

  return (
    <ConfirmContext.Provider value={ask}>
      {children}
      {pending && <Dialog options={pending} onAnswer={answer} />}
    </ConfirmContext.Provider>
  );
}

function Dialog({ options, onAnswer }: { options: ConfirmOptions; onAnswer: (ok: boolean) => void }) {
  const titleId = useId();
  const messageId = useId();
  const cancelRef = useRef<HTMLButtonElement>(null);

  // Esc cancels; the focus starts on "Cancelar", so a stray Enter never deletes anything.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === "Escape") onAnswer(false); };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onAnswer]);

  return (
    <div className="fixed inset-0 z-[65] bg-negro bg-opacity-40 flex items-end md:items-center justify-center p-4" onClick={() => onAnswer(false)}>
      <div
        role="alertdialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={messageId}
        onClick={(e) => e.stopPropagation()}
        className="bg-white border border-linea w-full max-w-md shadow-lg"
      >
        <div className="px-5 pt-5 pb-4">
          <h2 id={titleId} className="font-display text-lg text-negro">{options.title}</h2>
          <p id={messageId} className="text-sm font-light text-negro/70 mt-2 break-words">{options.message}</p>
        </div>
        <div className="flex flex-col-reverse sm:flex-row sm:justify-end gap-2 border-t border-linea p-4">
          <button
            ref={cancelRef}
            autoFocus
            type="button"
            onClick={() => onAnswer(false)}
            className="text-xs tracking-widest uppercase font-medium px-5 py-3 border border-linea text-negro hover:bg-lino transition-colors"
          >
            Cancelar
          </button>
          <button
            type="button"
            data-primary-action
            data-testid="confirm-yes"
            onClick={() => onAnswer(true)}
            className={`text-xs tracking-widest uppercase font-medium px-5 py-3 text-crema transition-colors ${options.danger ? "bg-peligro hover:bg-black" : "bg-negro hover:bg-black"}`}
          >
            {options.confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
