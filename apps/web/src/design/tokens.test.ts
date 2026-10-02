import { describe, expect, it } from "vitest";
import { WORK_ORDER_STATUSES } from "@/features/work-orders/status";

/**
 * WCAG 2.2 contrast proof for the design tokens, in both themes.
 *
 * Each token is resolved by the browser (light-dark() under a forced
 * color-scheme), painted to a 1×1 sRGB canvas, and read back as the actual
 * rendered pixel, so OKLCH gamut mapping is measured, not assumed.
 */

type Scheme = "light" | "dark";
type RGB = [number, number, number];

const canvas = document.createElement("canvas");
canvas.width = 1;
canvas.height = 1;
function context2d(): CanvasRenderingContext2D {
  const c = canvas.getContext("2d", { colorSpace: "srgb", willReadFrequently: true });
  if (!c) throw new Error("2d canvas unavailable");
  return c;
}
const ctx = context2d();

function rendered(token: string, scheme: Scheme): RGB {
  const el = document.createElement("div");
  el.style.colorScheme = scheme;
  el.style.color = `var(${token})`;
  document.body.append(el);
  const css = getComputedStyle(el).color;
  el.remove();
  if (!css || css === "rgba(0, 0, 0, 0)") throw new Error(`${token} did not resolve`);
  ctx.clearRect(0, 0, 1, 1);
  ctx.fillStyle = css;
  ctx.fillRect(0, 0, 1, 1);
  const [r = 0, g = 0, b = 0] = ctx.getImageData(0, 0, 1, 1).data;
  return [r, g, b];
}

function luminance([r, g, b]: RGB): number {
  const lin = (c: number) => {
    const s = c / 255;
    return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
}

function contrast(a: RGB, b: RGB): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x) as [number, number];
  return (hi + 0.05) / (lo + 0.05);
}

const TEXT = 4.5; // WCAG 1.4.3 normal text
const UI = 3; // WCAG 1.4.11 non-text UI components / graphical objects

const statusPairs = WORK_ORDER_STATUSES.map((s) => {
  const t = s.replace("_", "-");
  return [`--status-${t}-fg`, `--status-${t}-bg`, TEXT] as const;
});

const PAIRS: ReadonlyArray<readonly [fg: string, bg: string, min: number]> = [
  ["--text-primary", "--surface-canvas", TEXT],
  ["--text-primary", "--surface-1", TEXT],
  ["--text-primary", "--surface-2", TEXT],
  ["--text-primary", "--surface-3", TEXT],
  ["--text-secondary", "--surface-canvas", TEXT],
  ["--text-secondary", "--surface-1", TEXT],
  ["--text-secondary", "--surface-2", TEXT],
  ["--text-secondary", "--surface-3", TEXT],
  ["--text-subtle", "--surface-1", TEXT],
  ["--text-link", "--surface-1", TEXT],
  ["--text-inverse", "--surface-inverse", TEXT],
  ["--accent-fg", "--accent", TEXT],
  ["--accent-subtle-fg", "--accent-subtle", TEXT],
  ["--danger-fg", "--danger", TEXT],
  ["--priority-low", "--surface-1", TEXT],
  ["--priority-normal", "--surface-1", TEXT],
  ["--priority-high", "--surface-1", TEXT],
  ["--priority-urgent", "--surface-1", TEXT],
  ["--warning", "--surface-1", UI],
  ["--success", "--surface-1", UI],
  ["--border-strong", "--surface-1", UI],
  ["--focus-ring", "--surface-1", UI],
  ["--focus-ring", "--surface-canvas", UI],
  ["--live-connected", "--surface-1", UI],
  ["--live-reconnecting", "--surface-1", UI],
  ["--live-offline", "--surface-1", UI],
  ...statusPairs,
];

describe.each<Scheme>(["light", "dark"])("design tokens (%s)", (scheme) => {
  it.each(PAIRS)("%s on %s meets %s:1", (fg, bg, min) => {
    const ratio = contrast(rendered(fg, scheme), rendered(bg, scheme));
    expect(ratio, `${fg} on ${bg} = ${ratio.toFixed(2)}:1`).toBeGreaterThanOrEqual(min);
  });

  it("themes actually differ (light-dark() resolves per scheme)", () => {
    const other: Scheme = scheme === "light" ? "dark" : "light";
    expect(rendered("--surface-1", scheme)).not.toEqual(rendered("--surface-1", other));
  });
});
