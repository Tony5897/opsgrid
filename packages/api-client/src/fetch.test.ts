import { describe, expect, it, vi } from "vitest";
import { apiFetch, CSRF_HEADER } from "./fetch.ts";
import { ApiError, NetworkError } from "./problem.ts";

function respond(status: number, body: unknown, headers: Record<string, string> = {}) {
  const isJSON = body !== undefined;
  return vi.fn<typeof fetch>(
    async () =>
      new Response(isJSON ? JSON.stringify(body) : null, {
        status,
        headers: {
          ...(isJSON && {
            "Content-Type": status >= 400 ? "application/problem+json" : "application/json",
          }),
          ...headers,
        },
      }),
  );
}

function sentHeaders(f: ReturnType<typeof respond>): Headers {
  const init = f.mock.calls[0]?.[1];
  return new Headers(init?.headers);
}

describe("apiFetch", () => {
  it("sends the CSRF header only on unsafe methods", async () => {
    const get = respond(200, { ok: true });
    await apiFetch("/v1/meta", {}, get);
    expect(sentHeaders(get).has(CSRF_HEADER)).toBe(false);

    const post = respond(201, { id: "1" });
    await apiFetch("/v1/x", { json: { a: 1 } }, post);
    expect(sentHeaders(post).get(CSRF_HEADER)).toBe("1");
    expect(post.mock.calls[0]?.[1]?.method).toBe("POST");
  });

  it("sends idempotency and precondition headers and same-origin credentials", async () => {
    const f = respond(200, { id: "wo" }, { ETag: '"v12"', "Idempotent-Replayed": "true" });
    const res = await apiFetch(
      "/v1/x",
      { method: "PATCH", json: {}, idempotencyKey: "k1", ifMatch: '"v11"' },
      f,
    );
    const h = sentHeaders(f);
    expect(h.get("Idempotency-Key")).toBe("k1");
    expect(h.get("If-Match")).toBe('"v11"');
    expect(f.mock.calls[0]?.[1]?.credentials).toBe("same-origin");
    expect(res.etag).toBe('"v12"');
    expect(res.replayed).toBe(true);
  });

  it("maps problem+json to a typed ApiError", async () => {
    const f = respond(
      412,
      {
        type: "https://opsgrid.dev/problems/VERSION_CONFLICT",
        title: "Version conflict",
        status: 412,
        code: "VERSION_CONFLICT",
        detail: "The resource changed since you loaded it.",
        requestId: "req_1",
        details: { currentVersion: 12 },
      },
      { "X-Request-Id": "req_1" },
    );
    const err = await apiFetch("/v1/x", { method: "PATCH", json: {} }, f).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    const apiErr = err as ApiError;
    expect(apiErr.code).toBe("VERSION_CONFLICT");
    expect(apiErr.status).toBe(412);
    expect(apiErr.problem.details).toEqual({ currentVersion: 12 });
    expect(apiErr.retryable).toBe(false);
  });

  it("normalizes non-problem gateway errors and reads Retry-After", async () => {
    const f = vi.fn<typeof fetch>(
      async () =>
        new Response("<html>Bad Gateway</html>", { status: 503, headers: { "Retry-After": "7" } }),
    );
    const err = (await apiFetch("/v1/x", {}, f).catch((e: unknown) => e)) as ApiError;
    expect(err.code).toBe("DEPENDENCY_UNAVAILABLE");
    expect(err.retryAfter).toBe(7);
    expect(err.retryable).toBe(true);
  });

  it("unknown codes from a newer server degrade to INTERNAL_ERROR", async () => {
    const f = respond(500, { code: "SOMETHING_NEW", status: 500, title: "x", type: "x" });
    const err = (await apiFetch("/v1/x", {}, f).catch((e: unknown) => e)) as ApiError;
    expect(err.code).toBe("INTERNAL_ERROR");
  });

  it("wraps transport failures as NetworkError but preserves aborts", async () => {
    const offline = vi.fn<typeof fetch>(async () => {
      throw new TypeError("Failed to fetch");
    });
    await expect(apiFetch("/v1/x", {}, offline)).rejects.toBeInstanceOf(NetworkError);

    const aborted = vi.fn<typeof fetch>(async () => {
      throw new DOMException("aborted", "AbortError");
    });
    await expect(apiFetch("/v1/x", {}, aborted)).rejects.toMatchObject({ name: "AbortError" });
  });
});
