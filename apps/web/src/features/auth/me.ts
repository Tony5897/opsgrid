import { apiFetch, type Me } from "@opsgrid/api-client";
import { queryOptions } from "@tanstack/react-query";

export const meKeys = {
  all: ["me"] as const,
  detail: () => [...meKeys.all, "detail"] as const,
};

export function meQueryOptions(opts?: { redirectOnAuth?: boolean }) {
  return queryOptions({
    queryKey: meKeys.detail(),
    queryFn: async (): Promise<Me> => {
      const init =
        opts?.redirectOnAuth === undefined ? {} : { redirectOnAuth: opts.redirectOnAuth };
      const res = await apiFetch<Me>("/v1/me", init);
      return res.data;
    },
    staleTime: 30_000,
    retry: (count, err) => {
      if (err && typeof err === "object" && "status" in err && err.status === 401) return false;
      return count < 2;
    },
  });
}
