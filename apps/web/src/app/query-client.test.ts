import { ApiError, NetworkError, toProblem } from "@opsgrid/api-client";
import { describe, expect, it } from "vitest";
import { retryDelay, shouldRetry } from "./query-client";

const apiError = (status: number, retryAfter?: number) =>
  new ApiError(toProblem(status, undefined), retryAfter);

describe("query retry policy", () => {
  it("retries transient failures only", () => {
    expect(shouldRetry(0, new NetworkError(new TypeError("offline")))).toBe(true);
    expect(shouldRetry(0, apiError(503))).toBe(true);
    expect(shouldRetry(0, apiError(429))).toBe(true);
    for (const status of [400, 401, 403, 404, 409, 412, 422]) {
      expect(shouldRetry(0, apiError(status)), `status ${status}`).toBe(false);
    }
    expect(shouldRetry(0, new Error("bug"))).toBe(false);
  });

  it("stops after three attempts", () => {
    expect(shouldRetry(3, apiError(503))).toBe(false);
  });

  it("honors Retry-After and otherwise caps jittered backoff", () => {
    expect(retryDelay(1, apiError(429, 7))).toBe(7000);
    for (let attempt = 0; attempt < 10; attempt++) {
      const d = retryDelay(attempt, apiError(503));
      expect(d).toBeGreaterThanOrEqual(0);
      expect(d).toBeLessThanOrEqual(10_000);
    }
  });
});
