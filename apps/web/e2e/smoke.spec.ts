import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";

test.describe("foundation smoke", () => {
  test("landing page loads with strict security headers and Sign in", async ({ page }) => {
    const response = await page.goto("/");
    expect(response?.status()).toBe(200);
    const headers = response?.headers() ?? {};
    expect(headers["content-security-policy"]).toContain("script-src 'self'");
    expect(headers["content-security-policy"]).toContain("frame-ancestors 'none'");
    expect(headers["x-content-type-options"]).toBe("nosniff");
    expect(headers["x-request-id"]).toMatch(/^req_/);
    await expect(page.getByRole("heading", { level: 1, name: "OpsGrid" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();
  });

  test("unauthenticated org deep link redirects to login", async ({ page }) => {
    await page.goto("/o/cascade-facilities/work-orders");
    // beforeLoad → /v1/me 401 → assign /auth/login → Keycloak or BFF login.
    await expect(page).toHaveURL(/\/auth\/login|openid-connect|realms\/opsgrid/, {
      timeout: 30_000,
    });
  });

  test("unknown API routes require auth when anonymous", async ({ request }) => {
    const res = await request.get("/v1/does-not-exist");
    expect(res.status()).toBe(401);
    expect(res.headers()["content-type"]).toContain("application/problem+json");
    const body = await res.json();
    expect(body.code).toBe("AUTHENTICATION_REQUIRED");
    expect(body.requestId).toMatch(/^req_/);
  });

  test("API meta endpoint reports the build", async ({ request }) => {
    const res = await request.get("/v1/meta");
    expect(res.ok()).toBe(true);
    expect(await res.json()).toMatchObject({ service: "opsgrid" });
  });

  test("GET /v1/me without session is 401", async ({ request }) => {
    const res = await request.get("/v1/me");
    expect(res.status()).toBe(401);
    expect(await res.json()).toMatchObject({ code: "AUTHENTICATION_REQUIRED" });
  });

  test("cross-site POST is rejected (CSRF)", async ({ request }) => {
    const res = await request.post("/v1/meta", {
      headers: { "Sec-Fetch-Site": "cross-site", "X-OpsGrid-CSRF": "1" },
    });
    expect(res.status()).toBe(403);
  });

  test("keyboard: landing Sign in is reachable", async ({ page, isMobile }) => {
    test.skip(isMobile, "physical keyboard flow");
    await page.goto("/");
    await page.bringToFront();
    await page.locator("body").evaluate((el) => el.focus());
    await page.keyboard.press("Tab");
    // Skip link or Sign in depending on focus order.
    const skip = page.getByRole("link", { name: "Skip to main content" });
    if (await skip.count()) {
      await expect(skip).toBeFocused();
    } else {
      await expect(page.getByRole("button", { name: "Sign in" })).toBeFocused();
    }
  });
});

test.describe("authenticated console (Keycloak)", () => {
  test("dana can sign in and open Cascade", async ({ page }) => {
    test.setTimeout(90_000);
    await page.goto("/auth/login?returnTo=/");
    // Keycloak login form.
    await page.locator("#username, input[name=username]").first().fill("dana");
    await page.locator("#password, input[name=password]").first().fill("opsgrid-dev-password");
    await page.locator("#kc-login, button[type=submit]").first().click();
    await expect(page.getByRole("heading", { level: 1, name: "OpsGrid" })).toBeVisible({
      timeout: 30_000,
    });
    await expect(page.getByRole("link", { name: /Cascade Facilities/ })).toBeVisible();
    await page.getByRole("link", { name: /Cascade Facilities/ }).click();
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
    const results = await new AxeBuilder({ page })
      .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
      .analyze();
    expect(results.violations.map((v) => `${v.id} (${v.nodes.length})`)).toEqual([]);
  });
});
