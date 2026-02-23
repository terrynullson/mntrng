#!/usr/bin/env node
/**
 * Run all screenshot scripts in order. Continues on failure so maximum screenshots are captured.
 * Requires API (8080) and frontend (3000) — e.g. after docker compose up -d or local dev.
 */
import { execSync } from "child_process";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const WEB_DIR = path.resolve(__dirname, "..");

const SCRIPTS = [
  "screenshot-settings.mjs",
  "screenshot-analytics.mjs",
  "screenshot-streams.mjs",
  "screenshot-stream-detail.mjs",
  "screenshot-admin-requests.mjs",
  "screenshot-admin-users.mjs",
  "screenshot-companies.mjs",
  "screenshot-overview.mjs",
  "screenshot-login.mjs",
  "screenshot-register.mjs"
];

function log(msg) {
  const ts = new Date().toISOString().slice(11, 23);
  console.log(`[${ts}] ${msg}`);
}

let failed = 0;
for (const script of SCRIPTS) {
  const name = script.replace(".mjs", "");
  log("Running " + name + " ...");
  try {
    execSync(`node scripts/${script}`, {
      cwd: WEB_DIR,
      stdio: "inherit",
      timeout: 120000
    });
  } catch (err) {
    failed += 1;
    log(name + " failed: " + (err.message || "exit non-zero"));
  }
}
log("Done. Failed: " + failed + " of " + SCRIPTS.length);
process.exit(failed > 0 ? 1 : 0);
