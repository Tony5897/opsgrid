/** Same value as CSRF_HEADER in fetch.ts (avoid circular import). */
const CSRF = "X-OpsGrid-CSRF";

/** Build the BFF login URL with a same-origin return path. */
export function loginUrl(returnTo: string = "/"): string {
  const path = returnTo.startsWith("/") ? returnTo : "/";
  return `/auth/login?returnTo=${encodeURIComponent(path)}`;
}

/** Top-level navigation to the BFF login (full page, not XHR). */
export function startLogin(returnTo?: string): void {
  const here =
    returnTo ??
    (typeof window !== "undefined" ? `${window.location.pathname}${window.location.search}` : "/");
  window.location.assign(loginUrl(here));
}

/**
 * Sign out with a same-origin POST + CSRF header, then top-level navigate to
 * the IdP end-session URL (or home). Do not use apiFetch — logout returns 302
 * to Keycloak and must not be treated as a JSON API call.
 */
export async function startLogout(): Promise<void> {
  try {
    const res = await fetch("/auth/logout", {
      method: "POST",
      credentials: "same-origin",
      headers: {
        [CSRF]: "1",
        Accept: "application/json, application/problem+json",
      },
      redirect: "manual",
    });
    const loc = res.headers.get("Location");
    if (loc) {
      window.location.assign(loc);
      return;
    }
    // Browsers hide cross-origin Location on opaque redirects.
    if (
      res.type === "opaqueredirect" ||
      res.status === 0 ||
      (res.status >= 300 && res.status < 400)
    ) {
      window.location.assign("/");
      return;
    }
  } catch {
    // Fall through to home.
  }
  window.location.assign("/");
}
