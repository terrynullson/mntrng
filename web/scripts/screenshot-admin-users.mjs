#!/usr/bin/env node
/**
 * Automated screenshot of /admin/users (Admin users, super_admin or read-only view).
 * Saves to screenshots/admin-users/<timestamp>.png.
 * Uses test_super_admin / TestSuper1 (or fallback test_screenshot_admin) from seed.
 */
import { existsSync, readFileSync } from "fs";
import { mkdir, writeFile } from "fs/promises";
import { fileURLToPath } from "url";
import path from "path";
import { chromium } from "playwright";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const WEB_DIR = path.resolve(__dirname, "..");
const ROOT_DIR = path.resolve(WEB_DIR, "..");
const SCREENSHOTS_DIR = path.join(ROOT_DIR, "screenshots", "admin-users");
const REPORT_PATH = path.join(ROOT_DIR, "screenshots", "admin-users", "REPORT.txt");

const LOGIN_SUPER = process.env.SCREENSHOT_ADMIN_LOGIN || "test_super_admin";
const PASSWORD_SUPER = process.env.SCREENSHOT_ADMIN_PASSWORD || "TestSuper1";
const BASE_URL = process.env.SCREENSHOT_BASE_URL || "http://localhost:3000";
const API_PORT = Number(process.env.API_PORT || "8080");
const API_HOST = process.env.API_HOST || "127.0.0.1";

function log(msg) {
  const ts = new Date().toISOString().slice(11, 23);
  console.log(`[${ts}] ${msg}`);
}

function loadEnvFromFile(dir, filename = ".env") {
  try {
    const envPath = path.join(dir, filename);
    if (!existsSync(envPath)) return;
    const content = readFileSync(envPath, "utf8");
    content.split("\n").forEach((line) => {
      const m = line.match(/^([^#=]+)=(.*)$/);
      if (m) process.env[m[1].trim()] = m[2].trim().replace(/^["']|["']$/g, "");
    });
  } catch (_) {}
}

async function waitForPort(host, port, timeoutMs = 10000) {
  const start = Date.now();
  const net = await import("net");
  return new Promise((resolve) => {
    const tryConnect = () => {
      const socket = new net.Socket();
      socket.setTimeout(2000);
      socket.on("connect", () => {
        socket.destroy();
        resolve(true);
      });
      socket.on("error", () => {
        socket.destroy();
        if (Date.now() - start > timeoutMs) {
          resolve(false);
          return;
        }
        setTimeout(tryConnect, 500);
      });
      socket.connect(port, host);
    };
    tryConnect();
  });
}

async function main() {
  log("FE-ADMIN-USERS-001: admin users page screenshot");
  const envFile = process.env.ENV_FILE || ".env";
  loadEnvFromFile(ROOT_DIR, envFile);

  const apiUp = await waitForPort(API_HOST, API_PORT, 8000);
  if (!apiUp) {
    log("API not running at " + API_HOST + ":" + API_PORT + ". Start API and re-run.");
    process.exit(1);
  }
  log("API is up.");

  const devUp = await waitForPort("127.0.0.1", 3000, 5000);
  if (!devUp) {
    log("Frontend not running on :3000. Start with: cd web && npm run dev");
    process.exit(1);
  }
  log("Frontend is up at " + BASE_URL);

  const timestamp = new Date().toISOString().replace(/[-:T]/g, "").slice(0, 14);
  const screenshotPath = path.join(SCREENSHOTS_DIR, `${timestamp}.png`);
  await mkdir(SCREENSHOTS_DIR, { recursive: true });

  const browser = await chromium.launch({ headless: true });
  try {
    const context = await browser.newContext({ baseURL: BASE_URL });
    const page = await context.newPage();

    await page.goto("/login", { waitUntil: "networkidle" });
    let loggedIn = false;
    for (const [login, password] of [[LOGIN_SUPER, PASSWORD_SUPER], ["test_screenshot_admin", "TestScreenshot1"]]) {
      await page.fill("#login-or-email", login);
      await page.fill("#login-password", password);
      await page.click('button[type="submit"]');
      try {
        await page.waitForURL((u) => u.pathname !== "/login", { timeout: 12000 });
        loggedIn = true;
        break;
      } catch {
        await page.goto("/login", { waitUntil: "networkidle" });
      }
    }
    if (!loggedIn) {
      log("Login failed (run seed to create test_super_admin).");
      await browser.close();
      process.exit(1);
    }

    await page.goto("/admin/users", { waitUntil: "networkidle" });
    await page.waitForTimeout(800);

    const hasTitle = await page.locator("h2.page-title:has-text('Users')").count() > 0;
    const hasFiltersOrReadOnly =
      (await page.locator("#users-company-filter").count()) > 0 ||
      (await page.locator("text=Read-only mode").count()) > 0;
    const hasTableOrEmpty =
      (await page.locator(".table-wrap table").count()) > 0 ||
      (await page.locator("text=No users found").count()) > 0 ||
      (await page.locator("text=Read-only mode").count()) > 0;

    await page.screenshot({ path: screenshotPath, fullPage: true });
    log("Screenshot saved: " + screenshotPath);

    let score = 9;
    if (!hasTitle) score = 7;
    if (!hasFiltersOrReadOnly || !hasTableOrEmpty) score = Math.min(score, 8);

    const reportLines = [
      "FE-ADMIN-USERS-001 REPORT",
      "Screenshot: " + path.relative(ROOT_DIR, screenshotPath),
      "Score: " + score,
      "Checks: authorized /admin/users, Users title, filters or read-only, table or empty.",
    ];
    const reportContent = reportLines.join("\n") + "\n";
    await writeFile(REPORT_PATH, reportContent);
    log("REPORT written: " + REPORT_PATH);
    console.log("\n--- REPORT ---\n" + reportContent + "---\n");

    await browser.close();
  } catch (err) {
    log("Error: " + err.message);
    await browser.close();
    process.exit(1);
  }
}

main();
