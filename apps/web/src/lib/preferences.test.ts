import { afterEach, describe, expect, it } from "vitest";
import { reloadPreferences, setPreferences } from "./preferences";

afterEach(() => {
  localStorage.clear();
  reloadPreferences();
});

describe("preferences", () => {
  it("applies and persists an explicit theme, and system removes the override", () => {
    setPreferences({ theme: "dark" });
    expect(document.documentElement.dataset.theme).toBe("dark");
    expect(localStorage.getItem("opsgrid.theme")).toBe("dark");

    setPreferences({ theme: "system" });
    expect(document.documentElement.dataset.theme).toBeUndefined();
    expect(localStorage.getItem("opsgrid.theme")).toBeNull();
  });

  it("applies compact density and restores it on reload", () => {
    setPreferences({ density: "compact" });
    expect(document.documentElement.dataset.density).toBe("compact");
    delete document.documentElement.dataset.density;
    reloadPreferences();
    expect(document.documentElement.dataset.density).toBe("compact");
  });

  it("compact density shrinks row height tokens", () => {
    const row = () =>
      getComputedStyle(document.documentElement).getPropertyValue("--density-row").trim();
    setPreferences({ density: "comfortable" });
    const comfortable = row();
    setPreferences({ density: "compact" });
    expect(Number.parseInt(row(), 10)).toBeLessThan(Number.parseInt(comfortable, 10));
  });
});
