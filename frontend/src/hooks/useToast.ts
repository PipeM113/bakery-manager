import { createContext, useContext } from "react";

export interface ToastApi {
  success: (title: string, detail?: string) => void;
  error: (title: string, detail?: string) => void;
  info: (title: string, detail?: string) => void;
}

export const ToastContext = createContext<ToastApi | null>(null);

/** Shows a notice at the top right: useToast().success("Receta creada", "Manjarcito"). */
export function useToast(): ToastApi {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error("useToast needs ToastProvider");
  return ctx;
}
