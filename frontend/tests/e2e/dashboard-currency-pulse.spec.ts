import { expect, test } from "@playwright/test";

test("a single currency pulse fills the panel without an empty grid cell", async ({ page }) => {
  await page.route("**/api/v1/me", (route) => route.fulfill({
    contentType: "application/json",
    body: JSON.stringify({ data: { id: "user-pulse", email: "reviewer@example.com", first_name: "Ada", last_name: "Reviewer" }, requestId: "req-pulse" }),
  }));
  await page.route("**/api/v1/documents?*", (route) => route.fulfill({
    contentType: "application/json",
    body: JSON.stringify({ data: [], page: { number: 1, size: 5, totalItems: 2, totalPages: 1, hasMore: false } }),
  }));
  await page.route("**/api/v1/reports/summary?*", (route) => route.fulfill({
    contentType: "application/json",
    body: JSON.stringify({ data: {
      from: "2026-01-01",
      to: "2026-12-31",
      documentCount: 1,
      currencies: [
        { currency: "USD", documentCount: 1, subtotal: "450.00", totalDiscount: "40.00", totalTax: "11.50", grandTotal: "421.50", taxBreakdown: [] },
      ],
    }, requestId: "req-pulse" }),
  }));

  await page.goto("/dashboard");
  const panel = page.locator(".dashboard-panel");
  const list = page.locator(".currency-pulse-list");
  await expect(page.locator(".currency-pulse")).toHaveCount(1);
  await expect(panel.getByText("USD 421.50", { exact: false })).toBeVisible();
  const desktop = await page.evaluate(() => {
    const grid = document.querySelector<HTMLElement>(".dashboard-grid")!;
    const panel = document.querySelector<HTMLElement>(".dashboard-panel")!;
    const list = document.querySelector<HTMLElement>(".currency-pulse-list")!;
    const pulse = document.querySelector<HTMLElement>(".currency-pulse")!;
    return {
      alignItems: getComputedStyle(grid).alignItems,
      unusedSpace: Math.round(panel.getBoundingClientRect().bottom - list.getBoundingClientRect().bottom),
      widthDifference: Math.round(list.getBoundingClientRect().width - pulse.getBoundingClientRect().width),
      renderedChildren: list.children.length,
    };
  });
  expect(desktop).toEqual({ alignItems: "start", unusedSpace: 1, widthDifference: 0, renderedChildren: 1 });

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(list).toHaveCSS("grid-template-columns", `${await list.evaluate((element) => element.clientWidth)}px`);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
});
