import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

const PUBLIC_PATHS = new Set(["/login", "/register"]);
const ACCESS_COOKIE = "hm_access_token";

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

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!api).*)"]
};
