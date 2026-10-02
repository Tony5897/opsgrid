import { QueryClientProvider } from "@tanstack/react-query";
import { createMemoryHistory, createRouter, RouterProvider } from "@tanstack/react-router";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { page } from "vitest/browser";
import { render } from "vitest-browser-react";
import { TooltipProvider } from "@/components/ui/tooltip";
import { routeTree } from "@/routeTree.gen";
import { createQueryClient } from "./query-client";

async function renderAt(path: string) {
  const queryClient = createQueryClient();
  const router = createRouter({
    routeTree,
    context: { queryClient },
    history: createMemoryHistory({ initialEntries: [path] }),
  });
  const screen = await render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <RouterProvider router={router} />
      </TooltipProvider>
    </QueryClientProvider>,
  );
  return { screen, router };
}

afterEach(async () => {
  await page.viewport(1280, 800);
});

describe("application shell", () => {
  it("renders the page heading and marks the active nav item", async () => {
    const { screen } = await renderAt("/o/cascade-facilities/dispatch");
    await expect.element(screen.getByRole("heading", { level: 1, name: "Dispatch" })).toBeVisible();
    const active = screen
      .getByRole("navigation", { name: "Primary" })
      .getByRole("link", { name: "Dispatch" });
    await expect.element(active).toHaveAttribute("aria-current", "page");
    expect(document.title).toBe("Dispatch · OpsGrid");
  });

  it("moves focus to the new page heading after navigation", async () => {
    const { screen } = await renderAt("/o/cascade-facilities/dispatch");
    await screen
      .getByRole("navigation", { name: "Primary" })
      .getByRole("link", { name: "Assets" })
      .click();
    const h1 = screen.getByRole("heading", { level: 1, name: "Assets" });
    await expect.element(h1).toBeVisible();
    await expect.poll(() => document.activeElement?.textContent).toBe("Assets");
  });

  it("provides a skip link to the main content", async () => {
    const { screen } = await renderAt("/o/cascade-facilities");
    const skip = screen.getByRole("link", { name: "Skip to main content" });
    await expect.element(skip).toHaveAttribute("href", "#main-content");
  });

  it("rejects malformed organization slugs with the not-found page", async () => {
    const { screen } = await renderAt("/o/Not_A_Slug!/dispatch");
    await expect.element(screen.getByRole("heading", { name: "Page not found" })).toBeVisible();
  });

  it("has no detectable accessibility violations (axe, WCAG 2.2 AA)", async () => {
    const { screen } = await renderAt("/o/cascade-facilities/work-orders");
    await expect
      .element(screen.getByRole("heading", { level: 1, name: "Work orders" }))
      .toBeVisible();
    const results = await axe.run(document.body, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"] },
    });
    const summary = results.violations.map((v) => `${v.id}: ${v.nodes.length} node(s) — ${v.help}`);
    expect(summary).toEqual([]);
  });

  it("on narrow screens, navigation lives in a focus-trapped sheet", async () => {
    await page.viewport(390, 844);
    const { screen } = await renderAt("/o/cascade-facilities/dispatch");
    await screen.getByRole("button", { name: "Open navigation" }).click();
    const dialog = screen.getByRole("dialog", { name: "Navigation" });
    await expect.element(dialog).toBeVisible();
    await dialog.getByRole("link", { name: "Schedule" }).click();
    await expect.element(screen.getByRole("heading", { level: 1, name: "Schedule" })).toBeVisible();
    await expect.element(screen.getByRole("dialog")).not.toBeInTheDocument();
  });
});
