// The queue of notices: pure logic, so it can be tested without a screen.
export type ToastKind = "success" | "error" | "info";

export interface Toast {
  id: number;
  kind: ToastKind;
  title: string;
  detail?: string;
}

/** How many are shown at once; the oldest is dropped when one more arrives. */
export const MAX_VISIBLE = 5;

/** How long a notice stays: long enough to read it; an error stays longer (it usually says what to do). */
export function durationFor(kind: ToastKind): number {
  return kind === "error" ? 10000 : 6000;
}

/** The newest goes first (on top). */
export function pushToast(list: Toast[], toast: Toast): Toast[] {
  return [toast, ...list].slice(0, MAX_VISIBLE);
}

export function dismissToast(list: Toast[], id: number): Toast[] {
  return list.filter((t) => t.id !== id);
}
