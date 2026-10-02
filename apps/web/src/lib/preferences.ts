import { useSyncExternalStore } from "react";

/**
 * Per-device UI preferences. Persisted to localStorage and mirrored onto
 * <html data-theme / data-density>, which the design tokens read. The same
 * keys are applied before first paint by public/theme-init.js.
 */
export type ThemePreference = "system" | "light" | "dark";
export type Density = "comfortable" | "compact";

export interface Preferences {
  theme: ThemePreference;
  density: Density;
  sidebarCollapsed: boolean;
}

const THEME_KEY = "opsgrid.theme";
const DENSITY_KEY = "opsgrid.density";
const SIDEBAR_KEY = "opsgrid.sidebar";

function read(): Preferences {
  let theme: ThemePreference = "system";
  let density: Density = "comfortable";
  let sidebarCollapsed = false;
  try {
    const t = localStorage.getItem(THEME_KEY);
    if (t === "light" || t === "dark") theme = t;
    if (localStorage.getItem(DENSITY_KEY) === "compact") density = "compact";
    sidebarCollapsed = localStorage.getItem(SIDEBAR_KEY) === "collapsed";
  } catch {
    // Storage blocked (private mode, policy): defaults apply.
  }
  return { theme, density, sidebarCollapsed };
}

let current = read();
const listeners = new Set<() => void>();

function apply(p: Preferences) {
  const root = document.documentElement;
  if (p.theme === "system") delete root.dataset.theme;
  else root.dataset.theme = p.theme;
  if (p.density === "compact") root.dataset.density = "compact";
  else delete root.dataset.density;
}

export function setPreferences(patch: Partial<Preferences>): void {
  current = { ...current, ...patch };
  try {
    if (current.theme === "system") localStorage.removeItem(THEME_KEY);
    else localStorage.setItem(THEME_KEY, current.theme);
    localStorage.setItem(DENSITY_KEY, current.density);
    localStorage.setItem(SIDEBAR_KEY, current.sidebarCollapsed ? "collapsed" : "expanded");
  } catch {
    // Non-fatal: the preference still applies for this session.
  }
  apply(current);
  for (const l of listeners) l();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function usePreferences(): Preferences {
  return useSyncExternalStore(subscribe, () => current);
}

/** Test helper: reset module state to stored values. */
export function reloadPreferences(): void {
  current = read();
  apply(current);
  for (const l of listeners) l();
}
