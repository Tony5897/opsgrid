import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

export type ConnectionState = "connected" | "reconnecting" | "offline";

const META: Record<ConnectionState, { label: string; detail: string; color: string }> = {
  connected: {
    label: "Online",
    detail: "Changes from your team appear automatically.",
    color: "bg-live-connected",
  },
  reconnecting: {
    label: "Reconnecting…",
    detail: "Live updates paused; data will refresh when the connection returns.",
    color: "bg-live-reconnecting",
  },
  offline: {
    label: "Offline",
    detail: "You're offline. What you see may be out of date.",
    color: "bg-live-offline",
  },
};

/**
 * Connection indicator: dot + text label (never color alone). The live region
 * announces changes politely so screen-reader users learn about degradation.
 */
export function ConnectionStatus({ state }: { state: ConnectionState }) {
  const meta = META[state];
  return (
    <Tooltip content={meta.detail}>
      <output
        aria-live="polite"
        className="inline-flex h-8 items-center gap-2 rounded-md px-2 text-sm text-text-secondary"
      >
        <span className="relative flex size-2.5" aria-hidden>
          {state === "connected" && (
            <span
              className={cn(
                "absolute inline-flex size-full animate-ping rounded-full opacity-40 motion-reduce:hidden",
                meta.color,
              )}
            />
          )}
          <span className={cn("relative inline-flex size-2.5 rounded-full", meta.color)} />
        </span>
        <span className="hidden sm:inline">{meta.label}</span>
        <span className="sr-only sm:hidden">{meta.label}</span>
      </output>
    </Tooltip>
  );
}
