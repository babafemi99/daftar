import { expect, test } from "@playwright/test";

test.use({ viewport: { width: 390, height: 844 } });

test("authenticated user can sign out from the mobile header", async ({ page }) => {
  let signedOut = false;
  await page.route("**/api/v1/me", (route) => route.fulfill(signedOut ? {
    status: 401,
    contentType: "application/json",
    body: JSON.stringify({ error: { code: "UNAUTHORIZED", message: "Authentication is required.", requestId: "req-mobile" } }),
  } : {
    contentType: "application/json",
    body: JSON.stringify({ data: { id: "user-mobile", email: "reviewer@example.com", first_name: "Ada", last_name: "Reviewer" }, requestId: "req-mobile" }),
  }));
  await page.route("**/api/v1/documents?*", (route) => route.fulfill({
    contentType: "application/json",
    body: JSON.stringify({ data: [], page: { number: 1, size: 10, totalItems: 0, totalPages: 0, hasMore: false } }),
  }));
  const logout = page.waitForRequest((request) => request.url().endsWith("/api/v1/auth/logout") && request.method() === "POST");
  await page.route("**/api/v1/auth/logout", async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 150));
    signedOut = true;
    await route.fulfill({ status: 204 });
  });

  await page.goto("/documents");
  const signOut = page.locator(".mobile-bar").getByRole("button", { name: "Sign out" });
  await expect(signOut).toBeVisible();
  await expect(signOut).toHaveCSS("width", "40px");
  await expect(signOut).toHaveCSS("height", "40px");
  await signOut.click();
  await page.evaluate(() => window.dispatchEvent(new Event("daftar:session-expired")));

  await logout;
  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByText("Signed out", { exact: true })).toBeVisible();
  await expect(page.getByText("Session expired", { exact: true })).toHaveCount(0);
});
