// Applies the saved theme and density before first paint to avoid a flash.
// A separate file (not inline) keeps the CSP strict: script-src 'self'.
(() => {
  try {
    const theme = localStorage.getItem("opsgrid.theme");
    const density = localStorage.getItem("opsgrid.density");
    const root = document.documentElement;
    if (theme === "light" || theme === "dark") root.dataset.theme = theme;
    if (density === "compact") root.dataset.density = "compact";
  } catch {
    /* storage unavailable: fall back to system preference */
  }
})();
