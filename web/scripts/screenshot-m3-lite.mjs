#!/usr/bin/env node
/**
 * M3-lite core UX screenshot diagnostics.
 * Captures:
 * - screenshots/login/{ts}.png
 * - screenshots/hub-portal/{ts}.png
 * - screenshots/watch/{ts}.png
 * - screenshots/monitoring-streams/{ts}.png
 * - screenshots/incidents/{ts}.png
 * - screenshots/incidents-detail/{ts}.png
 * - screenshots/telegram-delivery-settings/{ts}.png
 *
 * Requires API and frontend to be up (docker compose up --build -d).
 * Uses test_screenshot_admin / TestScreenshot1 by default.
 */
import { mkdir, writeFile } from "fs/promises";
import path from "path";
import { fileURLToPath } from "url";
import { chromium } from "playwright";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const WEB_DIR = path.resolve(__dirname, "..");
const ROOT_DIR = path.resolve(WEB_DIR, "..");

const LOGIN_DIR = path.join(ROOT_DIR, "screenshots", "login");
const HUB_DIR = path.join(ROOT_DIR, "screenshots", "hub-portal");
const HUB_ARTIFACT_DIR = path.join(ROOT_DIR, "screenshots", "hub");
const WATCH_DIR = path.join(ROOT_DIR, "screenshots", "watch");
const MONITORING_STREAMS_DIR = path.join(ROOT_DIR, "screenshots", "monitoring-streams");
const INCIDENTS_DIR = path.join(ROOT_DIR, "screenshots", "incidents");
const INCIDENT_DETAIL_DIR = path.join(ROOT_DIR, "screenshots", "incidents-detail");
const TELEGRAM_DIR = path.join(ROOT_DIR, "screenshots", "telegram-delivery-settings");
const M3_REPORT = path.join(ROOT_DIR, "screenshots", "milestone3-lite", "REPORT.txt");

const LOGIN = process.env.SCREENSHOT_LOGIN || "test_screenshot_admin";
const PASSWORD = process.env.SCREENSHOT_PASSWORD || "TestScreenshot1";
const BASE_URL = process.env.SCREENSHOT_BASE_URL || "http://localhost:3000";
const API_URL = process.env.SCREENSHOT_API_URL || (() => {
  const u = new URL(BASE_URL);
  if (u.hostname === "frontend" || u.hostname.includes("frontend")) return "http://api:8080";
  return u.port === "3000" ? "http://localhost:8080" : `${u.protocol}//${u.hostname}:8080`;
})();

function log(msg) {
  const ts = new Date().toISOString().slice(11, 23);
  // eslint-disable-next-line no-console
  console.log(`[${ts}] ${msg}`);
}

async function waitForPortFromBaseUrl(timeoutMs = 20000) {
  const url = new URL(BASE_URL);
  const host = url.hostname || "127.0.0.1";
  const port = Number(url.port || (url.protocol === "https:" ? 443 : 80));
  const net = await import("net");

  const start = Date.now();
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

async function ensureFrontend() {
  log(`Checking frontend at ${BASE_URL} ...`);
  const ok = await waitForPortFromBaseUrl(25000);
  if (!ok) {
    log("Frontend is not reachable on BASE_URL; start stack via docker compose or npm dev.");
    process.exit(1);
  }
  log("Frontend is reachable.");
}

async function ensureDirs() {
  await Promise.all([
    mkdir(LOGIN_DIR, { recursive: true }),
    mkdir(HUB_DIR, { recursive: true }),
    mkdir(HUB_ARTIFACT_DIR, { recursive: true }),
    mkdir(WATCH_DIR, { recursive: true }),
    mkdir(MONITORING_STREAMS_DIR, { recursive: true }),
    mkdir(INCIDENTS_DIR, { recursive: true }),
    mkdir(INCIDENT_DETAIL_DIR, { recursive: true }),
    mkdir(TELEGRAM_DIR, { recursive: true }),
    mkdir(path.dirname(M3_REPORT), { recursive: true })
  ]);
}

async function captureLogin(page, timestamp) {
  log("Capturing /login ...");
  await page.goto("/login", { waitUntil: "networkidle", timeout: 30000 });
  await page.evaluate(() => {
    document.documentElement.setAttribute("data-theme", "dark");
  });
  await page.waitForTimeout(500);
  const screenshotPath = path.join(LOGIN_DIR, `${timestamp}.png`);
  await page.screenshot({ path: screenshotPath, fullPage: true });
  log("Login screenshot: " + path.relative(ROOT_DIR, screenshotPath));
}

async function loginAsAdmin(page) {
  log("Logging in as screenshot admin (API via Playwright) ...");
  const ctx = page.context();
  const loginRes = await ctx.request.post(API_URL + "/api/v1/auth/login", {
    data: { login_or_email: LOGIN, password: PASSWORD },
    headers: { "Content-Type": "application/json" },
    failOnStatusCode: false
  });
  if (loginRes.ok()) {
    log("API login OK, cookies stored in context.");
    await page.goto("/hub", { waitUntil: "networkidle", timeout: 30000 });
    await page.evaluate(() => {
      document.documentElement.setAttribute("data-theme", "dark");
    });
    return;
  }
  log("API login failed: status=" + loginRes.status() + ", falling back to UI login ...");
  await page.goto("/login", { waitUntil: "networkidle", timeout: 30000 });
  await page.fill("#login-or-email", LOGIN);
  await page.fill("#login-password", PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL((url) => url.pathname !== "/login", { timeout: 25000 });
  await page.evaluate(() => {
    document.documentElement.setAttribute("data-theme", "dark");
  });
}

async function captureHub(page, timestamp) {
  log("Capturing /hub ...");
  await page.goto("/hub", { waitUntil: "networkidle", timeout: 30000 });
  await page.waitForSelector(".hub-hero-title", { timeout: 10000 }).catch(() => {});
  await page.waitForSelector(".hub-hero-row", { timeout: 10000 }).catch(() => {});
  const buf = await page.screenshot({ fullPage: true });
  const hubPortalPath = path.join(HUB_DIR, `${timestamp}.png`);
  const hubArtifactPath = path.join(HUB_ARTIFACT_DIR, `${timestamp}.png`);
  await writeFile(hubPortalPath, buf);
  await writeFile(hubArtifactPath, buf);
  log("Hub screenshot: " + path.relative(ROOT_DIR, hubPortalPath) + ", " + path.relative(ROOT_DIR, hubArtifactPath));
}

async function captureWatch(page, timestamp) {
  log("Capturing /watch ...");
  await page.goto("/watch", { waitUntil: "networkidle", timeout: 30000 });
  await page.waitForSelector(".watch-layout", { timeout: 15000 }).catch(() => {});
  const firstStream = page.locator(".watch-stream-item").first();
  const hasStream = await firstStream.count().then((c) => c > 0);
  if (hasStream) {
    await firstStream.click();
    await page.waitForTimeout(800);
  }
  const screenshotPath = path.join(WATCH_DIR, `${timestamp}.png`);
  await page.screenshot({ path: screenshotPath, fullPage: true });
  log("Watch screenshot: " + path.relative(ROOT_DIR, screenshotPath));
}

async function captureMonitoringStreams(page, timestamp) {
  log("Capturing /monitoring/streams ...");
  await page.goto("/monitoring/streams", { waitUntil: "networkidle", timeout: 30000 });
  await page.waitForSelector(".filters-grid.streams-v2-filters", { timeout: 15000 }).catch(() => {});
  const screenshotPath = path.join(MONITORING_STREAMS_DIR, `${timestamp}.png`);
  await page.screenshot({ path: screenshotPath, fullPage: true });
  log("Monitoring Streams screenshot: " + path.relative(ROOT_DIR, screenshotPath));
}

async function captureIncidents(page, timestamp) {
  log("Capturing /incidents ...");
  await page.goto("/incidents", { waitUntil: "networkidle", timeout: 30000 });
  await page.waitForSelector("h2.page-title:has-text('Инциденты')", { timeout: 15000 }).catch(() => {});
  const screenshotPath = path.join(INCIDENTS_DIR, `${timestamp}.png`);
  await page.screenshot({ path: screenshotPath, fullPage: true });
  log("Incidents list screenshot: " + path.relative(ROOT_DIR, screenshotPath));
}

async function captureIncidentDetail(page, timestamp) {
  log("Capturing incident detail ...");
  // assume we are already on /incidents; if not, navigate
  const url = new URL(page.url());
  if (!url.pathname.startsWith("/incidents")) {
    await page.goto("/incidents", { waitUntil: "networkidle", timeout: 30000 });
  }

  const firstLink = page.locator('a[href^="/incidents/"]').first();
  const hasLink = await firstLink.count().then((c) => c > 0);
  if (!hasLink) {
    log("No incident detail link found; taking screenshot of list state instead.");
    const fallback = path.join(INCIDENT_DETAIL_DIR, `${timestamp}.png`);
    await page.screenshot({ path: fallback, fullPage: true });
    log("Incident detail (fallback) screenshot: " + path.relative(ROOT_DIR, fallback));
    return;
  }

  await firstLink.click();
  await page.waitForURL((u) => u.pathname.startsWith("/incidents/"), { timeout: 15000 }).catch(() => {});
  await page.waitForSelector(".stream-details-grid", { timeout: 15000 }).catch(() => {});

  const screenshotPath = path.join(INCIDENT_DETAIL_DIR, `${timestamp}.png`);
  await page.screenshot({ path: screenshotPath, fullPage: true });
  log("Incident detail screenshot: " + path.relative(ROOT_DIR, screenshotPath));
}

async function captureSettingsTelegram(page, timestamp) {
  log("Capturing /settings (Telegram section) ...");
  await page.goto("/settings", { waitUntil: "networkidle", timeout: 30000 });
  await page.waitForTimeout(800);
  const screenshotPath = path.join(TELEGRAM_DIR, `${timestamp}.png`);
  await page.screenshot({ path: screenshotPath, fullPage: true });
  log("Settings/Telegram screenshot: " + path.relative(ROOT_DIR, screenshotPath));
}

async function main() {
  log("MS3-lite: core UX screenshot diagnostics starting.");
  await ensureFrontend();
  await ensureDirs();

  const timestamp = new Date().toISOString().replace(/[-:T]/g, "").slice(0, 14);

  const browser = await chromium.launch({ headless: true });
  let context;
  try {
    context = await browser.newContext({
      baseURL: BASE_URL,
      viewport: { width: 1440, height: 900 },
      deviceScaleFactor: 2
    });
    const page = await context.newPage();

    await captureLogin(page, timestamp);
    await loginAsAdmin(page);
    await captureHub(page, timestamp);
    await captureWatch(page, timestamp);
    await captureMonitoringStreams(page, timestamp);
    await captureIncidents(page, timestamp);
    await captureIncidentDetail(page, timestamp);
    await captureSettingsTelegram(page, timestamp);

    const reportLines = [
      "MS3L-SCREENSHOT-DIAG-002",
      "STATUS: READY (runtime-tested: " + (process.env.DOCKER_DESKTOP || "unknown-env") + ")",
      "DATE: " + new Date().toISOString(),
      "Screens:",
      "- login: " + path.relative(ROOT_DIR, path.join(LOGIN_DIR, `${timestamp}.png`)),
      "- hub-portal: " + path.relative(ROOT_DIR, path.join(HUB_DIR, `${timestamp}.png`)),
      "- watch: " + path.relative(ROOT_DIR, path.join(WATCH_DIR, `${timestamp}.png`)),
      "- monitoring-streams: " + path.relative(ROOT_DIR, path.join(MONITORING_STREAMS_DIR, `${timestamp}.png`)),
      "- incidents: " + path.relative(ROOT_DIR, path.join(INCIDENTS_DIR, `${timestamp}.png`)),
      "- incidents-detail: " + path.relative(ROOT_DIR, path.join(INCIDENT_DETAIL_DIR, `${timestamp}.png`)),
      "- settings-telegram: " + path.relative(ROOT_DIR, path.join(TELEGRAM_DIR, `${timestamp}.png`)),
      "Note: pipeline assumes seeded test data (see docs/screenshot_automation.md)."
    ];
    const reportContent = reportLines.join("\n") + "\n";
    await writeFile(M3_REPORT, reportContent);
    log("M3-lite REPORT written: " + path.relative(ROOT_DIR, M3_REPORT));
    // eslint-disable-next-line no-console
    console.log("\n--- M3-LITE REPORT ---\n" + reportContent + "---\n");

    await browser.close();
  } catch (err) {
    log("Error in M3-lite screenshot pipeline: " + (err.message || String(err)));
    if (context) await context.close().catch(() => {});
    await browser.close();
    process.exit(1);
  }
}

main();

