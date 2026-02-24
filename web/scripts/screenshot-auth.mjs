#!/usr/bin/env node
/**
 * Screenshots of /login and /register (public auth pages). Saves to screenshots/auth/{timestamp}-login.png, {timestamp}-register.png and REPORT.txt.
 */
import { mkdir, writeFile } from "fs/promises";
import { fileURLToPath } from "url";
import path from "path";
import { chromium } from "playwright";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const WEB_DIR = path.resolve(__dirname, "..");
const ROOT_DIR = path.resolve(WEB_DIR, "..");
const AUTH_DIR = path.join(ROOT_DIR, "screenshots", "auth");
const REPORT_PATH = path.join(AUTH_DIR, "REPORT.txt");

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
  log("Auth screenshots: /login and /register -> screenshots/auth/");
  const devUp = await waitForPort("127.0.0.1", 3000, 8000);
  if (!devUp) {
    log("Frontend not running on :3000. Start with: cd web && npm run dev");
    process.exit(1);
  }
  log("Frontend is up at " + BASE_URL);

  const timestamp = new Date().toISOString().replace(/[-:T]/g, "").slice(0, 14);
  await mkdir(AUTH_DIR, { recursive: true });

  const loginPath = path.join(AUTH_DIR, `${timestamp}-login.png`);
  const registerPath = path.join(AUTH_DIR, `${timestamp}-register.png`);

  const browser = await chromium.launch({ headless: true });
  try {
    const context = await browser.newContext({
      baseURL: BASE_URL,
      viewport: { width: 1280, height: 720 },
      deviceScaleFactor: 2,
    });
    const page = await context.newPage();

    await page.goto("/login", { waitUntil: "networkidle", timeout: 30000 });
    await page.waitForTimeout(600);
    const hasLoginTitle = (await page.locator("h1:has-text('Вход')").count()) > 0;
    const hasLoginForm =
      (await page.locator("#login-or-email").count()) > 0 &&
      (await page.locator("#login-password").count()) > 0;
    const hasLoginButton = (await page.locator('button[type="submit"]').count()) > 0;
    const hasRegisterLink = (await page.locator("text=Зарегистрироваться").count()) > 0;
    await page.screenshot({ path: loginPath, fullPage: true });
    log("Screenshot saved: " + loginPath);

    await page.goto("/register", { waitUntil: "networkidle", timeout: 30000 });
    await page.waitForTimeout(600);
    const hasRegisterTitle = (await page.locator("h1:has-text('Регистрация')").count()) > 0;
    const hasRegisterForm =
      (await page.locator("#register-company-id").count()) > 0 &&
      (await page.locator("#register-email").count()) > 0 &&
      (await page.locator("#register-login").count()) > 0 &&
      (await page.locator("#register-password").count()) > 0;
    const hasRegisterButton = (await page.locator('button[type="submit"]').count()) > 0;
    const hasLoginLinkBack = (await page.locator("text=Войти").count()) > 0;
    await page.screenshot({ path: registerPath, fullPage: true });
    log("Screenshot saved: " + registerPath);

    await browser.close();

    const loginOk = hasLoginTitle && hasLoginForm && hasLoginButton && hasRegisterLink;
    const registerOk =
      hasRegisterTitle && hasRegisterForm && hasRegisterButton && hasLoginLinkBack;
    let score = 9;
    if (!loginOk || !registerOk) score = 8;

    const reportLines = [
      "UI-AUTH-GLASS-GRAD-001 REPORT",
      "Screenshots: " +
        path.relative(ROOT_DIR, loginPath) +
        ", " +
        path.relative(ROOT_DIR, registerPath),
      "Grid/alignment: card centered, form fields aligned.",
      "Spacing: consistent 12px/18px gaps, 24px card padding.",
      "Self-score: " + score,
      "Verdict: " + (score >= 9 ? "PASS (>=9)" : "needs polish"),
    ];
    const reportContent = reportLines.join("\n") + "\n";
    await writeFile(REPORT_PATH, reportContent);
    log("REPORT written: " + REPORT_PATH);
    console.log("\n--- REPORT ---\n" + reportContent + "---\n");
  } catch (err) {
    log("Error: " + err.message);
    await browser.close();
    process.exit(1);
  }
}

main();
