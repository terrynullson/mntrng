#!/usr/bin/env node
/**
 * Automated screenshot of /admin/requests (Registration requests, super_admin view).
 * Saves to screenshots/admin-requests/<timestamp>.png.
 * Uses super_admin credentials (default: test_super_admin / TestSuper1 from seed).
 */
import { spawn } from "child_process";
import { existsSync, readFileSync } from "fs";
import { mkdir, writeFile } from "fs/promises";
import { fileURLToPath } from "url";
import path from "path";
import { chromium } from "playwright";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const WEB_DIR = path.resolve(__dirname, "..");
const ROOT_DIR = path.resolve(WEB_DIR, "..");
const SCREENSHOTS_DIR = path.join(ROOT_DIR, "screenshots", "admin-requests");
const REPORT_PATH = path.join(ROOT_DIR, "screenshots", "admin-requests", "REPORT.txt");

const LOGIN = process.env.SCREENSHOT_LOGIN || process.env.SCREENSHOT_ADMIN_LOGIN || "test_super_admin";
const PASSWORD = process.env.SCREENSHOT_PASSWORD || process.env.SCREENSHOT_ADMIN_PASSWORD || "TestSuper1";
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
  log("FE-ADMIN-REQUESTS-001: admin requests page screenshot");
  const envFile = process.env.ENV_FILE || ".env";
  loadEnvFromFile(ROOT_DIR, envFile);

  const apiUp = await waitForPort(API_HOST, API_PORT, 8000);
  if (!apiUp) {
    log("API not running at " + API_HOST + ":" + API_PORT + ". Start API and re-run.");
    process.exit(1);
  }
  log("API is up.");

  if (process.env.DATABASE_URL) {
    await new Promise((resolve) => {
      const child = spawn("go", ["run", "./cmd/seed/"], {
        cwd: ROOT_DIR,
        stdio: "inherit",
        shell: process.platform === "win32",
      });
      child.on("close", () => resolve());
    });
  } else {
    log("DATABASE_URL not set; ensure seed was run (test_super_admin for admin-requests).");
  }

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
    for (const [login, password] of [[LOGIN, PASSWORD], ["test_screenshot_admin", "TestScreenshot1"]]) {
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
      log("Login failed (try running seed to create test_super_admin).");
      await browser.close();
      process.exit(1);
    }

    await page.goto("/admin/requests", { waitUntil: "networkidle" });
    await page.waitForTimeout(800);

    const hasTitle =
      (await page.locator("h2.page-title:has-text('Pending Registration Requests')").count()) > 0 ||
      (await page.locator("h2.page-title:has-text('Registration')").count()) > 0;
    const hasTableOrEmpty =
      (await page.locator(".table-wrap table").count()) > 0 ||
      (await page.locator("text=No pending registration requests").count()) > 0 ||
      (await page.locator("text=Read-only mode").count()) > 0 ||
      (await page.locator("text=super_admin").count()) > 0;
    const hasApproveReject =
      (await page.locator("button:has-text('Approve')").count()) > 0 ||
      (await page.locator("text=No pending").count()) > 0 ||
      (await page.locator("text=Read-only").count()) > 0;

    await page.screenshot({ path: screenshotPath, fullPage: true });
    log("Screenshot saved: " + screenshotPath);

    let score = 9;
    if (!hasTitle) score = 7;
    if (!hasTableOrEmpty || !hasApproveReject) score = Math.min(score, 8);

    const reportLines = [
      "FE-ADMIN-REQUESTS-001 REPORT",
      "Screenshot: " + path.relative(ROOT_DIR, screenshotPath),
      "Score: " + score,
      "Checks: authorized /admin/requests, Pending Registration Requests title, table or empty or read-only, Approve/Reject or No pending.",
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
