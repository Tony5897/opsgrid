import { ApiError, NetworkError } from "@opsgrid/api-client";
import { QueryClient } from "@tanstack/react-query";

const MAX_RETRIES = 3;

/**
 * Server-state cache. Realtime events (Gate 5) invalidate queries; until then
 * focus/reconnect refetching keeps views fresh.
 *
 * Retries: only transient failures (network, 429, 502-504). Client errors
 * (4xx) are deterministic and retrying them only delays feedback.
 */
export function shouldRetry(failureCount: number, error: unknown): boolean {
  if (failureCount >= MAX_RETRIES) return false;
  if (error instanceof NetworkError) return true;
  if (error instanceof ApiError) return error.retryable;
  return false;
}

export function retryDelay(attempt: number, error: unknown): number {
  if (error instanceof ApiError && error.retryAfter) return error.retryAfter * 1000;
  // Exponential backoff with full jitter, capped at 10 s.
  return Math.random() * Math.min(10_000, 500 * 2 ** attempt);
}

export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        gcTime: 5 * 60_000,
        retry: shouldRetry,
        retryDelay,
        refetchOnWindowFocus: true,
        refetchOnReconnect: true,
      },
      mutations: {
        // Mutations retry only with the same Idempotency-Key (set per call).
        retry: false,
      },
    },
  });
}
