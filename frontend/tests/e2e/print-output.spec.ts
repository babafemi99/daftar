import { expect, test } from "@playwright/test";

const pageMeta = { number: 1, size: 25, totalItems: 0, totalPages: 0, hasMore: false };

test("finalized CrossVal document prints as one clean A4 page", async ({ page }) => {
  await page.route("**/api/v1/me", (route) => route.fulfill({
    contentType: "application/json",
    body: JSON.stringify({ data: { id: "user-print", email: "reviewer@example.com", first_name: "Ada", last_name: "Reviewer" }, requestId: "req-print" }),
  }));
  await page.route("**/api/v1/documents/document-print/audit-events?limit=25", (route) => route.fulfill({
    contentType: "application/json",
    body: JSON.stringify({ data: [], page: pageMeta }),
  }));
  await page.route("**/api/v1/documents/document-print", (route) => route.fulfill({
    contentType: "application/json",
    body: JSON.stringify({ data: finalizedDocument(), requestId: "req-print" }),
  }));

  await page.goto("/documents/document-print");
  await expect(page.getByRole("button", { name: "Print / Save PDF" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Document activity" })).toBeVisible();

  await page.emulateMedia({ media: "print" });
  await expect(page.locator(".print-document-header")).toBeVisible();
  await expect(page.locator(".view-metadata")).toBeVisible();
  await expect(page.locator(".view-line")).toHaveCount(4);
  await expect(page.locator(".view-summary")).toBeVisible();
  await expect(page.locator(".tax-summary")).toBeVisible();
  await expect(page.locator(".print-document-footer")).toBeVisible();
  await expect(page.locator(".print-document-header strong")).toHaveText("DOC-2026-000421");

  for (const selector of [
    ".skip-link",
    ".app-sidebar",
    ".mobile-bar",
    ".mobile-nav",
    ".detail-page-heading",
    ".document-actions",
    ".audit-card",
    ".dialog-backdrop",
    ".save-state",
    ".save-hint",
    "[data-sonner-toaster]",
  ]) await expect(page.locator(selector)).toBeHidden();

  const styles = await page.evaluate(() => {
    const style = (selector: string) => getComputedStyle(document.querySelector<HTMLElement>(selector)!);
    const pageRule = [...document.styleSheets].flatMap((sheet) => {
      try { return [...sheet.cssRules]; } catch { return []; }
    }).find((rule) => rule.cssText.startsWith("@page"));
    return {
      totalBackground: style(".grand-total").backgroundColor,
      labelColor: style(".grand-total dt").color,
      amountColor: style(".grand-total .money-amount").color,
      rowBreakInside: style(".view-line:not(.view-line--head)").breakInside,
      summaryBreakInside: style(".view-summary").breakInside,
      pageRule: pageRule?.cssText ?? "",
    };
  });
  expect(styles.totalBackground).toBe("rgb(23, 63, 52)");
  expect(styles.labelColor).toBe("rgb(255, 255, 255)");
  expect(styles.amountColor).toBe("rgb(255, 255, 255)");
  expect(styles.rowBreakInside).toBe("avoid");
  expect(styles.summaryBreakInside).toBe("avoid");
  expect(styles.pageRule.toLowerCase()).toContain("size: a4");
  expect(styles.pageRule).toContain("margin: 14mm");

  const pdf = await page.pdf({ printBackground: true, preferCSSPageSize: true });
  expect(pdf.toString("latin1").match(/\/Type\s*\/Page\b/g) ?? []).toHaveLength(1);
});

function finalizedDocument() {
  return {
    id: "document-print",
    reference: "DOC-2026-000421",
    title: "CrossVal multi-rate sample",
    customer: "CrossVal Reviewer",
    issueDate: "2026-08-10",
    currency: "USD",
    status: "finalized",
    version: 2,
    lineItems: [
      line("line-one", "Consulting", 2, "100.00", { type: "percentage", value: "10" }, "5", "199.50"),
      line("line-two", "Implementation", 1, "200.00", { type: "fixed", value: "20.00" }, "7.5", "193.50"),
      line("line-three", "Support", 1, "50.00", undefined, "0", "50.00"),
    ],
    totals: { subtotal: "450.00", discount: "40.00", tax: "33.00", grandTotal: "443.00" },
    taxBreakdown: [
      { rate: "0", taxableAmount: "50.00", taxAmount: "0.00" },
      { rate: "5", taxableAmount: "180.00", taxAmount: "9.00" },
      { rate: "7.5", taxableAmount: "180.00", taxAmount: "13.50" },
    ],
    calculationPolicyVersion: "crossval-v1",
    finalizedAt: "2026-08-10T12:00:00Z",
    archivedAt: null,
    createdAt: "2026-08-10T11:00:00Z",
    updatedAt: "2026-08-10T12:00:00Z",
  };
}

function line(id: string, description: string, quantity: number, unitPrice: string, discount: { type: string; value: string } | undefined, taxRate: string, lineTotal: string) {
  return { id, description, quantity, unitPrice, discount, taxRate, calculated: { subtotal: unitPrice, discountAmount: "0.00", discountedAmount: unitPrice, taxAmount: "0.00", lineTotal } };
}
