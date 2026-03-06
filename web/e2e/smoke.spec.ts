import { test, expect } from "@playwright/test";

test.describe("smoke", () => {
  test("login page loads and shows heading", async ({ page }) => {
    await page.goto("/login");
    await expect(page.getByRole("heading", { name: "Вход" })).toBeVisible();
  });

  test("root redirects unauthenticated user to login", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveURL(/login/);
    await expect(page.getByRole("heading", { name: "Вход" })).toBeVisible();
  });

  test("register page loads and shows heading", async ({ page }) => {
    await page.goto("/register");
    await expect(page.getByRole("heading", { name: "Регистрация" })).toBeVisible();
  });
});
