import type { ReactNode } from "react";

interface Props {
  title: string;
  onClose: () => void;
  children: ReactNode;
  /** The actions; always visible at the bottom while the body scrolls. Mark the main one with data-primary-action. */
  footer?: ReactNode;
  maxWidth?: string;
}

/** A window that always fits the screen: the body scrolls and the actions stay within reach. */
export default function Modal({ title, onClose, children, footer, maxWidth = "max-w-md" }: Props) {
  return (
    <div className="fixed inset-0 z-[60] bg-negro bg-opacity-40 flex items-end md:items-center justify-center p-4">
      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className={`bg-white border border-linea w-full ${maxWidth} max-h-full flex flex-col shadow-lg`}
      >
        <div className="flex items-center justify-between gap-3 px-4 pt-4 pb-3 shrink-0">
          <h2 className="font-display text-lg text-negro">{title}</h2>
          <button onClick={onClose} aria-label="Cerrar" className="text-negro/70 hover:text-negro text-xl leading-none">×</button>
        </div>
        <div className="flex-1 min-h-0 overflow-y-auto px-4 pb-3 space-y-3">{children}</div>
        {footer && <div className="shrink-0 border-t border-linea p-4">{footer}</div>}
      </div>
    </div>
  );
}
