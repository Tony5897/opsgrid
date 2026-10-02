import { ApiError, NetworkError, toProblem } from "./problem.ts";

/** Header the API requires on every unsafe request (ADR-019). */
export const CSRF_HEADER = "X-OpsGrid-CSRF";

const SAFE_METHODS = new Set(["GET", "HEAD", "OPTIONS"]);

export interface ApiRequestInit extends Omit<RequestInit, "body"> {
  /** JSON-serializable body. */
  json?: unknown;
  /** Required on POST commands (ADR / IMPLEMENTATION_PLAN §4.6). */
  idempotencyKey?: string;
  /** Expected entity version for optimistic concurrency (If-Match). */
  ifMatch?: string;
}

export interface ApiResponse<T> {
  data: T;
  status: number;
  etag: string | null;
  requestId: string | null;
  replayed: boolean;
}

/**
 * The single fetch path for the OpsGrid API: same-origin cookies, CSRF
 * header on unsafe methods, JSON in/out, RFC 9457 problems as ApiError.
 */
export async function apiFetch<T>(
  path: string,
  init: ApiRequestInit = {},
  fetchImpl: typeof fetch = fetch,
): Promise<ApiResponse<T>> {
  const { json, idempotencyKey, ifMatch, headers: initHeaders, ...rest } = init;
  const method = (rest.method ?? (json === undefined ? "GET" : "POST")).toUpperCase();
  const headers = new Headers(initHeaders);
  headers.set("Accept", "application/json, application/problem+json");
  if (!SAFE_METHODS.has(method)) headers.set(CSRF_HEADER, "1");
  if (json !== undefined) headers.set("Content-Type", "application/json");
  if (idempotencyKey) headers.set("Idempotency-Key", idempotencyKey);
  if (ifMatch) headers.set("If-Match", ifMatch);

  let res: Response;
  try {
    res = await fetchImpl(path, {
      ...rest,
      method,
      headers,
      credentials: "same-origin",
      ...(json !== undefined && { body: JSON.stringify(json) }),
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") throw err;
    throw new NetworkError(err);
  }

  const requestId = res.headers.get("X-Request-Id");
  const isJSON = /json/i.test(res.headers.get("Content-Type") ?? "");
  const body: unknown =
    res.status === 204 || !isJSON ? undefined : await res.json().catch(() => undefined);

  if (!res.ok) {
    const retryAfter = Number(res.headers.get("Retry-After"));
    throw new ApiError(
      toProblem(res.status, body, requestId),
      Number.isFinite(retryAfter) && retryAfter > 0 ? retryAfter : undefined,
    );
  }
  return {
    data: body as T,
    status: res.status,
    etag: res.headers.get("ETag"),
    requestId,
    replayed: res.headers.get("Idempotent-Replayed") === "true",
  };
}

/** A fresh idempotency key; create one per user intent and reuse it on retries. */
export function newIdempotencyKey(): string {
  return crypto.randomUUID();
}
