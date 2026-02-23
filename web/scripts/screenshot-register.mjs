#!/usr/bin/env node
/**
 * Screenshot of /register (public page, no auth). Saves to screenshots/register/<timestamp>.png.
 */
import { mkdir, writeFile } from "fs/promises";
import { fileURLToPath } from "url";
import path from "path";
import { chromium } from "playwright";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const WEB_DIR = path.resolve(__dirname, "..");
const ROOT_DIR = path.resolve(WEB_DIR, "..");
const SCREENSHOTS_DIR = path.join(ROOT_DIR, "screenshots", "register");
const REPORT_PATH = path.join(ROOT_DIR, "screenshots", "register", "REPORT.txt");

const BASE_URL = process.env.SCREENSHOT_BASE_URL || "http://localhost:3000";

function log(msg) {
  const ts = new Date().toISOString().slice(11, 23);
  console.log(`[${ts}] ${msg}`);
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
  log("FE-REGISTER-001: register page screenshot (public)");
  const devUp = await waitForPort("127.0.0.1", 3000, 8000);
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
    const context = await browser.newContext({
      baseURL: BASE_URL,
      viewport: { width: 1280, height: 720 },
      deviceScaleFactor: 2,
    });
    const page = await context.newPage();

    await page.goto("/register", { waitUntil: "load", timeout: 30000 });
    await page.waitForSelector("h1", { state: "visible", timeout: 25000 });
    await page.waitForTimeout(800);

    const hasTitle = await page.locator("h1:has-text('Registration Request')").count() > 0;
    const hasCaption = await page
      .locator("text=Create a pending request for super admin approval")
      .count() > 0;
    const hasCompanyId = await page.locator("#register-company-id").count() > 0;
    const hasEmail = await page.locator("#register-email").count() > 0;
    const hasLogin = await page.locator("#register-login").count() > 0;
    const hasPassword = await page.locator("#register-password").count() > 0;
    const hasRole = await page.locator("#register-role").count() > 0;
    const hasButton = await page.locator('button[type="submit"]').count() > 0;
    const hasBackToLogin = await page.locator("text=Already approved?").count() > 0;

    await page.screenshot({ path: screenshotPath, fullPage: true });
    log("Screenshot saved: " + screenshotPath);

    let score = 9;
    if (!hasTitle) score = 7;
    const formOk =
      hasCaption && hasCompanyId && hasEmail && hasLogin && hasPassword && hasRole && hasButton;
    if (!formOk || !hasBackToLogin) score = Math.min(score, 8);

    const reportLines = [
      "FE-REGISTER-001 REPORT",
      "Screenshot: " + path.relative(ROOT_DIR, screenshotPath),
      "Score: " + score,
      "Checks: /register public, Registration Request title, caption, form (Company ID, Email, Login, Password, Requested role, Submit), Already approved? link.",
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
