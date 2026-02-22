#!/usr/bin/env node
/**
 * Automated screenshot of /streams/[streamId] (Stream Player, authorized).
 * Saves to screenshots/stream-detail/<timestamp>.png.
 * Flow: login → /streams → navigate to first stream link or /streams/1.
 */
import { existsSync, readFileSync } from "fs";
import { mkdir, writeFile } from "fs/promises";
import { fileURLToPath } from "url";
import path from "path";
import { chromium } from "playwright";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const WEB_DIR = path.resolve(__dirname, "..");
const ROOT_DIR = path.resolve(WEB_DIR, "..");
const SCREENSHOTS_DIR = path.join(ROOT_DIR, "screenshots", "stream-detail");
const REPORT_PATH = path.join(ROOT_DIR, "screenshots", "stream-detail", "REPORT.txt");

const LOGIN = process.env.SCREENSHOT_LOGIN || "test_screenshot_admin";
const PASSWORD = process.env.SCREENSHOT_PASSWORD || "TestScreenshot1";
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
  log("FE-STREAM-DETAIL-001: stream detail (player) page screenshot");
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
    await page.fill("#login-or-email", LOGIN);
    await page.fill("#login-password", PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL((u) => u.pathname !== "/login", { timeout: 15000 });

    await page.goto("/streams", { waitUntil: "networkidle" });
    await page.waitForTimeout(1200);

    const streamLinks = page.locator('a.stream-link[href^="/streams/"]');
    const count = await streamLinks.count();
    let detailPath = "/streams/1";
    if (count > 0) {
      const href = await streamLinks.first().getAttribute("href");
      if (href && href !== "/streams") detailPath = href;
    }
    await page.goto(detailPath, { waitUntil: "networkidle" });
    await page.waitForTimeout(600);

    const hasTitle = await page.locator("h2.page-title:has-text('Stream Player')").count() > 0;
    const hasBackLink = await page.locator('a.stream-link:has-text("Back to streams")').count() > 0;
    const hasPlayerOrEmpty =
      (await page.locator(".stream-details-grid .player-card").count()) > 0 ||
      (await page.locator("text=Stream not found").count()) > 0 ||
      (await page.locator("text=Select company scope").count()) > 0;
    const hasStatusBlock =
      (await page.locator(".status-card").count()) > 0 ||
      (await page.locator("text=Latest Status").count()) > 0 ||
      (await page.locator("text=No check results").count()) > 0 ||
      (await page.locator("text=Stream not found").count()) > 0 ||
      (await page.locator("text=Select company scope").count()) > 0;

    await page.screenshot({ path: screenshotPath, fullPage: true });
    log("Screenshot saved: " + screenshotPath);

    let score = 9;
    if (!hasTitle) score = 7;
    if (!hasBackLink) score = Math.min(score, 8);
    if (!hasPlayerOrEmpty || !hasStatusBlock) score = Math.min(score, 8);

    const reportLines = [
      "FE-STREAM-DETAIL-001 REPORT",
      "Screenshot: " + path.relative(ROOT_DIR, screenshotPath),
      "Score: " + score,
      "Checks: authorized Stream Player, Back to streams, stream card or empty, Latest Status or No check results.",
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
