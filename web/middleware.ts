import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

const PUBLIC_PATHS = new Set(["/login", "/register"]);
const ACCESS_COOKIE = "hm_access_token";
const ACTIVE_COMPANY_COOKIE = "hm_active_company_id";
const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

function isStaticAsset(pathname: string): boolean {
  return (
    pathname.startsWith("/_next") ||
    pathname.startsWith("/favicon") ||
    pathname.startsWith("/robots.txt") ||
    pathname.startsWith("/sitemap")
  );
}

async function buildWatchCSP(request: NextRequest): Promise<string> {
  const accessToken = request.cookies.get(ACCESS_COOKIE)?.value;
  if (!accessToken) {
    return "default-src 'self'; frame-src 'self';";
  }

  const meResponse = await fetch(`${API_BASE}/api/v1/auth/me`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
      Accept: "application/json"
    }
  });
  if (!meResponse.ok) {
    return "default-src 'self'; frame-src 'self';";
  }
  const me = (await meResponse.json()) as {
    role?: string;
    company_id?: number | null;
  };
  let companyID: number | null = null;
  if (me.role === "super_admin") {
    const selected = Number.parseInt(request.cookies.get(ACTIVE_COMPANY_COOKIE)?.value ?? "", 10);
    companyID = Number.isFinite(selected) && selected > 0 ? selected : null;
  } else if (typeof me.company_id === "number" && me.company_id > 0) {
    companyID = me.company_id;
  }
  if (!companyID) {
    return "default-src 'self'; frame-src 'self';";
  }

  const whitelistResponse = await fetch(`${API_BASE}/api/v1/companies/${companyID}/embed-whitelist`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
      Accept: "application/json"
    }
  });
  if (!whitelistResponse.ok) {
    return "default-src 'self'; frame-src 'self';";
  }
  const payload = (await whitelistResponse.json()) as {
    items?: Array<{ domain?: string; enabled?: boolean }>;
  };
  const domains = (payload.items ?? [])
    .filter((item) => item.enabled && typeof item.domain === "string" && item.domain.trim() !== "")
    .map((item) => item.domain!.toLowerCase().trim());
  const frameSrc = ["'self'", ...domains.map((domain) => `https://${domain}`), ...domains.map((domain) => `https://*.${domain}`)];
  return `default-src 'self'; frame-src ${frameSrc.join(" ")};`;
}

export async function middleware(request: NextRequest) {
  const { pathname, search } = request.nextUrl;

  if (isStaticAsset(pathname)) {
    return NextResponse.next();
  }

  const hasAccessCookie = Boolean(request.cookies.get(ACCESS_COOKIE)?.value);
  const isPublic = PUBLIC_PATHS.has(pathname);

  if (!hasAccessCookie && !isPublic) {
    const loginURL = request.nextUrl.clone();
    loginURL.pathname = "/login";
    loginURL.searchParams.set("next", `${pathname}${search}`);
    return NextResponse.redirect(loginURL);
  }

  if (hasAccessCookie && isPublic) {
    const homeURL = request.nextUrl.clone();
    homeURL.pathname = "/";
    homeURL.search = "";
    return NextResponse.redirect(homeURL);
  }

  const response = NextResponse.next();
  if (pathname.startsWith("/watch")) {
    try {
      const csp = await buildWatchCSP(request);
      response.headers.set("Content-Security-Policy", csp);
    } catch {
      response.headers.set("Content-Security-Policy", "default-src 'self'; frame-src 'self';");
    }
  }
  return response;
}

export const config = {
  matcher: ["/((?!api).*)"]
};
