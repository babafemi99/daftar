import { expect, test } from "@playwright/test";

test("authenticated CrossVal workflow remains owner-scoped", async ({ browser, page }) => {
  const run = `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  const password = "strong-password";

  await page.goto("/register");
  await page.getByLabel("First name").fill("Ada");
  await page.getByLabel("Last name").fill("Lovelace");
  await page.getByLabel("Email address").fill(`owner-${run}@example.com`);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(/\/dashboard$/);

  await page.getByRole("button", { name: "Create the 421.50 sample" }).click();
  await expect(page).toHaveURL(/\/documents\/[^/]+$/);
  const documentURL = page.url();
  await expect(page.getByText("USD 421.50", { exact: false })).toBeVisible();
  await expect(page.getByText("USD 450.00", { exact: false })).toBeVisible();

  await page.getByLabel("Title").fill("CrossVal autosaved sample");
  await expect(page.getByText("All changes saved")).toBeVisible({ timeout: 10_000 });
  await expect(page.getByRole("heading", { name: "CrossVal autosaved sample" })).toBeVisible();

  await page.getByRole("button", { name: "Finalize", exact: true }).click();
  await expect(page.getByRole("alertdialog")).toBeVisible();
  await page.getByRole("button", { name: "Finalize document" }).click();
  await expect(page.getByText("finalized", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Print / Save PDF" })).toBeVisible();
  await expect(page.getByLabel("Title")).toHaveCount(0);

  const activity = page.getByLabel("Document activity");
  await expect(activity.getByRole("heading", { name: "Document activity" })).toBeVisible();
  await expect(activity.getByText("Document finalized", { exact: true })).toBeVisible();
  await expect(activity.getByText("Draft created", { exact: true })).toBeVisible();
  await expect(activity.getByText(/Version 3/)).toBeVisible();

  await page.emulateMedia({ media: "print" });
  await expect(page.locator(".print-document-header")).toBeVisible();
  await expect(page.locator(".view-metadata")).toBeVisible();
  await expect(page.locator(".view-lines")).toBeVisible();
  await expect(page.locator(".view-line")).toHaveCount(4);
  await expect(page.locator(".view-summary")).toBeVisible();
  await expect(page.locator(".tax-summary")).toBeVisible();
  await expect(page.locator(".print-document-footer")).toBeVisible();
  await expect(page.locator(".print-document-shell").getByText(/DOC-/).first()).toBeVisible();
  await expect(page.locator(".skip-link")).toBeHidden();
  await expect(page.locator(".app-sidebar")).toBeHidden();
  await expect(page.locator(".mobile-bar")).toBeHidden();
  await expect(page.locator(".mobile-nav")).toBeHidden();
  await expect(page.locator(".detail-page-heading")).toBeHidden();
  await expect(page.locator("[data-sonner-toaster]")).toBeHidden();
  await expect(activity).toBeHidden();
  const printStyles = await page.evaluate(() => {
    const total = document.querySelector<HTMLElement>(".grand-total");
    const label = document.querySelector<HTMLElement>(".grand-total dt");
    const amount = document.querySelector<HTMLElement>(".grand-total .money-amount");
    const row = document.querySelector<HTMLElement>(".view-line:not(.view-line--head)");
    const summary = document.querySelector<HTMLElement>(".view-summary");
    if (!total || !label || !amount || !row || !summary) throw new Error("print document is incomplete");
    return {
      totalBackground: getComputedStyle(total).backgroundColor,
      labelColor: getComputedStyle(label).color,
      amountColor: getComputedStyle(amount).color,
      rowBreakInside: getComputedStyle(row).breakInside,
      summaryBreakInside: getComputedStyle(summary).breakInside,
    };
  });
  expect(printStyles).toEqual({
    totalBackground: "rgb(23, 63, 52)",
    labelColor: "rgb(255, 255, 255)",
    amountColor: "rgb(255, 255, 255)",
    rowBreakInside: "avoid",
    summaryBreakInside: "avoid",
  });
  await page.emulateMedia({ media: "screen" });

  for (let index = 0; index < 10; index += 1) {
    const created = await page.request.post("/api/v1/documents", {
      headers: { Origin: "http://localhost:3000" },
      data: {
        title: `Pagination record ${String(index + 1).padStart(2, "0")}`,
        customer: "Ledger reviewer",
        issueDate: "2026-08-08",
        currency: "USD",
        lineItems: [{ description: "Review line", quantity: 1, unitPrice: "1.00", taxRate: "0" }],
      },
    });
    expect(created.status()).toBe(201);
  }

  await page.getByLabel("Main navigation").getByRole("link", { name: "Documents" }).click();
  await expect(page).toHaveURL(/\/documents$/);
  await expect(page.getByText("Page 1 of 2")).toBeVisible();
  await page.getByRole("button", { name: "Next" }).click();
  await expect(page.getByText("Page 2 of 2")).toBeVisible();

  await page.getByLabel("Search documents").fill("CrossVal autosaved");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await expect(page.getByRole("link", { name: "CrossVal autosaved sample", exact: true })).toBeVisible();
  await expect(page.getByText("Page 2 of 2")).toHaveCount(0);

  const refreshed = await page.request.post("/api/v1/auth/refresh", { headers: { Origin: "http://localhost:3000" } });
  expect(refreshed.status()).toBe(200);
  const currentUser = await page.request.get("/api/v1/me");
  expect(currentUser.status()).toBe(200);

  await page.getByLabel("Main navigation").getByRole("link", { name: "Reports" }).click();
  await expect(page).toHaveURL(/\/reports$/);
  await expect(page.locator(".currency-report-card .money-amount").filter({ hasText: "USD 421.50" })).toBeVisible();
  await expect(page.locator(".currency-report-card .document-count")).toContainText("1 document");

  const reportScreenshotPath = process.env.DAFTAR_REPORT_SCREENSHOT_PATH;
  if (reportScreenshotPath) {
    await page.setViewportSize({ width: 1440, height: 900 });
    await expect(page.locator("[data-sonner-toaster] [data-sonner-toast]")).toHaveCount(0);
    await page.screenshot({ path: reportScreenshotPath, fullPage: false });
  }

  const otherContext = await browser.newContext();
  const other = await otherContext.newPage();
  await other.goto("/register");
  await other.getByLabel("First name").fill("Grace");
  await other.getByLabel("Last name").fill("Hopper");
  await other.getByLabel("Email address").fill(`other-${run}@example.com`);
  await other.getByLabel("Password").fill(password);
  await other.getByRole("button", { name: "Create account" }).click();
  await expect(other).toHaveURL(/\/dashboard$/);

  const inaccessible = other.waitForResponse((response) => response.url() === documentURL.replace("localhost:3000", "localhost:3000").replace(/\/documents\//, "/api/v1/documents/") && response.request().method() === "GET");
  await other.goto(documentURL);
  const response = await inaccessible;
  expect(response.status()).toBe(404);
  const body = await response.json();
  expect(body.error.code).toBe("RESOURCE_NOT_FOUND");
  await otherContext.close();
});
