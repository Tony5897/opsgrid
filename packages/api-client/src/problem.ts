import type { FieldError, Problem, ProblemCode } from "./gen/types.gen.ts";
import { zProblemCode } from "./gen/zod.gen.ts";

/**
 * RFC 9457 problem details as returned by the OpsGrid API (ADR-012). The
 * shapes are generated from api/openapi.yaml; `code` is the stable,
 * machine-readable identifier the UI maps to copy.
 */
export type { FieldError, Problem, ProblemCode };

/** Error thrown for every non-2xx API response. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: ProblemCode;
  readonly problem: Problem;
  /** Seconds to wait before retrying (429/503), when the server says so. */
  readonly retryAfter: number | undefined;

  constructor(problem: Problem, retryAfter?: number) {
    super(problem.detail ?? problem.title);
    this.name = "ApiError";
    this.status = problem.status;
    this.code = problem.code;
    this.problem = problem;
    this.retryAfter = retryAfter;
  }

  /** Errors worth retrying automatically: never for 4xx client errors. */
  get retryable(): boolean {
    return this.status === 429 || this.status === 502 || this.status === 503 || this.status === 504;
  }
}

/** Thrown when the request never produced an HTTP response. */
export class NetworkError extends Error {
  constructor(cause: unknown) {
    super("Network request failed", { cause });
    this.name = "NetworkError";
  }
}

const KNOWN_CODES: ReadonlySet<string> = new Set(zProblemCode.options);

/**
 * Normalizes any error body into a Problem. Non-problem responses (proxy
 * HTML error pages, empty bodies) become INTERNAL_ERROR / DEPENDENCY_UNAVAILABLE
 * so callers can always switch on `code`.
 */
export function toProblem(status: number, body: unknown, requestId?: string | null): Problem {
  if (body && typeof body === "object" && "code" in body && typeof body.code === "string") {
    const rawCode: string = body.code;
    const b = body as Partial<Problem>;
    return {
      type: b.type ?? "about:blank",
      title: b.title ?? "Request failed",
      status: b.status ?? status,
      code: KNOWN_CODES.has(rawCode) ? (rawCode as ProblemCode) : "INTERNAL_ERROR",
      ...(b.detail !== undefined && { detail: b.detail }),
      ...(b.instance !== undefined && { instance: b.instance }),
      ...((b.requestId ?? requestId) ? { requestId: b.requestId ?? (requestId as string) } : {}),
      ...(b.traceId !== undefined && { traceId: b.traceId }),
      ...(b.details !== undefined && { details: b.details }),
      ...(b.errors !== undefined && { errors: b.errors }),
    };
  }
  const gateway = status === 502 || status === 503 || status === 504;
  return {
    type: "about:blank",
    title: gateway ? "Service temporarily unavailable" : "Request failed",
    status,
    code: gateway
      ? "DEPENDENCY_UNAVAILABLE"
      : status === 401
        ? "AUTHENTICATION_REQUIRED"
        : "INTERNAL_ERROR",
    ...(requestId && { requestId }),
  };
}
