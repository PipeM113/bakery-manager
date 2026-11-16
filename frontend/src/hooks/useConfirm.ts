import { createContext, useContext } from "react";

export interface ConfirmOptions {
  title: string;
  /** What will happen, naming the thing. */
  message: string;
  /** What the main button says, as an action: "Eliminar receta". */
  confirmLabel: string;
  /** A destructive action is red; the rest use the dark button. */
  danger?: boolean;
}

export type Ask = (options: ConfirmOptions) => Promise<boolean>;

export const ConfirmContext = createContext<Ask | null>(null);

/** Asks before something that cannot be undone: if (!(await ask({...}))) return; */
export function useConfirm(): Ask {
  const ctx = useContext(ConfirmContext);
  if (!ctx) throw new Error("useConfirm needs ConfirmProvider");
  return ctx;
}
