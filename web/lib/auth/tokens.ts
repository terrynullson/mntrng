const ACCESS_COOKIE = "hm_access_token";
const REFRESH_COOKIE = "hm_refresh_token";

const ACCESS_STORAGE_KEY = "hm.access_token";
const REFRESH_STORAGE_KEY = "hm.refresh_token";

function setCookie(name: string, value: string, maxAgeSeconds: number): void {
  if (typeof document === "undefined") {
    return;
  }

  document.cookie = `${name}=${encodeURIComponent(
    value
  )}; path=/; max-age=${maxAgeSeconds}; samesite=lax`;
}

function clearCookie(name: string): void {
  if (typeof document === "undefined") {
    return;
  }

  document.cookie = `${name}=; path=/; max-age=0; samesite=lax`;
}

function readCookie(name: string): string | null {
  if (typeof document === "undefined") {
    return null;
  }

  const parts = document.cookie.split(";").map((item) => item.trim());
  const found = parts.find((part) => part.startsWith(`${name}=`));
  if (!found) {
    return null;
  }

  return decodeURIComponent(found.slice(name.length + 1));
}

export function persistAuthTokens(
  accessToken: string,
  refreshToken: string,
  accessTtlSeconds: number
): void {
  if (typeof window !== "undefined") {
    localStorage.setItem(ACCESS_STORAGE_KEY, accessToken);
    localStorage.setItem(REFRESH_STORAGE_KEY, refreshToken);
  }

  const safeAccessTtl = Math.max(accessTtlSeconds, 60);
  setCookie(ACCESS_COOKIE, accessToken, safeAccessTtl);
  setCookie(REFRESH_COOKIE, refreshToken, 60 * 60 * 24 * 30);
}

export function clearAuthTokens(): void {
  if (typeof window !== "undefined") {
    localStorage.removeItem(ACCESS_STORAGE_KEY);
    localStorage.removeItem(REFRESH_STORAGE_KEY);
  }

  clearCookie(ACCESS_COOKIE);
  clearCookie(REFRESH_COOKIE);
}

export function readAccessToken(): string | null {
  if (typeof window !== "undefined") {
    const fromStorage = localStorage.getItem(ACCESS_STORAGE_KEY);
    if (fromStorage) {
      return fromStorage;
    }
  }

  return readCookie(ACCESS_COOKIE);
}

export function readRefreshToken(): string | null {
  if (typeof window !== "undefined") {
    const fromStorage = localStorage.getItem(REFRESH_STORAGE_KEY);
    if (fromStorage) {
      return fromStorage;
    }
  }

  return readCookie(REFRESH_COOKIE);
}
