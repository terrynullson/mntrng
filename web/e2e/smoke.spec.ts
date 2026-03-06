import { test, expect } from "@playwright/test";

test.describe("smoke", () => {
  test("login page loads and shows heading", async ({ page }) => {
    await page.goto("/login");
    await expect(page.getByRole("heading", { name: "Вход" })).toBeVisible();
  });
});
