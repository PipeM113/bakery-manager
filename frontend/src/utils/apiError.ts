/** The message the backend sent with a failed request ({"error": "..."}), or the fallback. */
export function apiErrorMessage(e: unknown, fallback: string): string {
  const message = (e as { response?: { data?: { error?: unknown } } })?.response?.data?.error;
  return typeof message === "string" && message ? message : fallback;
}
