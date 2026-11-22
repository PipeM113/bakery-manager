type Failed = {
  response?: { status?: number; data?: { error?: unknown }; headers?: Record<string, unknown> };
};

/**
 * The message the backend sent with a failed request ({"error": "..."}), or the fallback. When the server itself
 * failed (5xx) the code of the request is added, so a person can read it out and it can be found in the log of the
 * backend. Only a hexadecimal code is shown: the header is not trusted blindly.
 */
export function apiErrorMessage(e: unknown, fallback: string): string {
  const response = (e as Failed)?.response;
  const sent = response?.data?.error;
  const message = typeof sent === "string" && sent ? sent : fallback;
  const status = response?.status ?? 0;
  const code = response?.headers?.["x-request-id"];
  if (status >= 500 && typeof code === "string" && /^[0-9a-f]{8,32}$/.test(code)) return `${message} (código ${code})`;
  return message;
}
