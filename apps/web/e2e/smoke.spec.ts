import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";

test.describe("foundation smoke", () => {
  test("landing page loads with strict security headers", async ({ page }) => {
    const response = await page.goto("/");
    expect(response?.status()).toBe(200);
    const headers = response?.headers() ?? {};
    expect(headers["content-security-policy"]).toContain("script-src 'self'");
    expect(headers["content-security-policy"]).toContain("frame-ancestors 'none'");
    expect(headers["x-content-type-options"]).toBe("nosniff");
    expect(headers["x-request-id"]).toMatch(/^req_/);
    await expect(page.getByRole("heading", { level: 1, name: "OpsGrid" })).toBeVisible();
  });

  test("deep links are served by the SPA fallback", async ({ page }) => {
    const response = await page.goto("/o/cascade-facilities/work-orders");
    expect(response?.status()).toBe(200);
    await expect(page.getByRole("heading", { level: 1, name: "Work orders" })).toBeVisible();
  });

  test("unknown API routes return problem+json, not the SPA", async ({ request }) => {
    const res = await request.get("/v1/does-not-exist");
    expect(res.status()).toBe(404);
    expect(res.headers()["content-type"]).toContain("application/problem+json");
    const body = await res.json();
    expect(body.code).toBe("RESOURCE_NOT_FOUND");
    expect(body.requestId).toMatch(/^req_/);
  });

  test("API meta endpoint reports the build", async ({ request }) => {
    const res = await request.get("/v1/meta");
    expect(res.ok()).toBe(true);
    expect(await res.json()).toMatchObject({ service: "opsgrid" });
  });

  test("cross-site POST is rejected (CSRF)", async ({ request }) => {
    const res = await request.post("/v1/meta", {
      headers: { "Sec-Fetch-Site": "cross-site", "X-OpsGrid-CSRF": "1" },
    });
    expect(res.status()).toBe(403);
  });

  for (const theme of ["light", "dark"] as const) {
    test(`console shell has no axe violations (${theme})`, async ({ page }) => {
      await page.emulateMedia({ colorScheme: theme });
      await page.goto("/o/cascade-facilities/dispatch");
      await expect(page.getByRole("heading", { level: 1, name: "Dispatch" })).toBeVisible();
      const results = await new AxeBuilder({ page })
        .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
        .analyze();
      expect(results.violations.map((v) => `${v.id} (${v.nodes.length})`)).toEqual([]);
    });
  }

  test("keyboard: skip link moves focus to main content", async ({ page, isMobile }) => {
    test.skip(isMobile, "physical keyboard flow");
    await page.goto("/o/cascade-facilities");
    await page.keyboard.press("Tab");
    const skip = page.getByRole("link", { name: "Skip to main content" });
    await expect(skip).toBeFocused();
    await page.keyboard.press("Enter");
    await expect(page.locator("#main-content")).toBeFocused();
  });
});
