import { useRouterState } from "@tanstack/react-router";
import { useEffect, useRef } from "react";

/**
 * After client-side navigation, move focus to the new page's <h1> (or <main>)
 * so keyboard and screen-reader users land at the top of the new content,
 * mirroring what a full page load would do. Skipped on first render.
 *
 * Keyed on the *resolved* location: routes are lazy chunks, so focusing on the
 * pathname change would target the outgoing page's heading, which React then
 * unmounts, dropping focus to <body>.
 */
export function useRouteFocus(): void {
  const resolvedPath = useRouterState({ select: (s) => s.resolvedLocation?.pathname });
  const previous = useRef(resolvedPath);
  useEffect(() => {
    if (resolvedPath === previous.current) return;
    previous.current = resolvedPath;
    const frame = requestAnimationFrame(() => {
      const target =
        document.querySelector<HTMLElement>("#main-content h1") ??
        document.getElementById("main-content");
      target?.focus();
    });
    return () => cancelAnimationFrame(frame);
  }, [resolvedPath]);
}
