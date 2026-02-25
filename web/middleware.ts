import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

const PUBLIC_PATHS = new Set(["/login", "/register"]);
const ACCESS_COOKIE = "hm_access_token";
const REFRESH_COOKIE = "hm_refresh_token";
const WATCH_BASE_CSP =
  "default-src 'self'; frame-src 'self' https:; img-src 'self' data: blob: https:; media-src 'self' https:; script-src 'self'; style-src 'self' 'unsafe-inline';";

function isStaticAsset(pathname: string): boolean {
  return (
    pathname.startsWith("/_next") ||
    pathname.startsWith("/favicon") ||
    pathname.startsWith("/robots.txt") ||
    pathname.startsWith("/sitemap")
  );
}

export function middleware(request: NextRequest) {
  const { pathname, search } = request.nextUrl;

  if (isStaticAsset(pathname)) {
    return NextResponse.next();
  }

  const hasSessionCookie =
    Boolean(request.cookies.get(ACCESS_COOKIE)?.value) ||
    Boolean(request.cookies.get(REFRESH_COOKIE)?.value);
  const isPublic = PUBLIC_PATHS.has(pathname);

  if (!hasSessionCookie && !isPublic) {
    const loginURL = request.nextUrl.clone();
    loginURL.pathname = "/login";
    loginURL.searchParams.set("next", `${pathname}${search}`);
    return NextResponse.redirect(loginURL);
  }

  if (hasSessionCookie && isPublic) {
    const homeURL = request.nextUrl.clone();
    homeURL.pathname = "/";
    homeURL.search = "";
    return NextResponse.redirect(homeURL);
  }

  const response = NextResponse.next();
  if (pathname.startsWith("/watch")) {
    response.headers.set("Content-Security-Policy", WATCH_BASE_CSP);
  }
  return response;
}

export const config = {
  matcher: ["/((?!api).*)"]
};
