"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState
} from "react";

import { ApiError, apiRequest } from "@/lib/api/client";
import type {
  AuthTokensResponse,
  AuthUser,
  Company,
  LoginRequest
} from "@/lib/api/types";
import {
  clearAuthTokens,
  persistAuthTokens,
  readAccessToken,
  readRefreshToken
} from "@/lib/auth/tokens";

type AuthContextValue = {
  isReady: boolean;
  isAuthenticated: boolean;
  user: AuthUser | null;
  accessToken: string | null;
  companies: Company[];
  activeCompanyId: number | null;
  setActiveCompanyId: (value: number | null) => void;
  loginWithPassword: (request: LoginRequest) => Promise<void>;
  logout: () => Promise<void>;
  refreshMe: () => Promise<void>;
};

const ACTIVE_COMPANY_STORAGE_KEY = "hm.active_company_id";
const ACTIVE_COMPANY_COOKIE = "hm_active_company_id";

const AuthContext = createContext<AuthContextValue | null>(null);

function sortCompanies(companies: Company[]): Company[] {
  return [...companies].sort((left, right) => left.name.localeCompare(right.name));
}

async function requestMe(accessToken: string): Promise<AuthUser> {
  return apiRequest<AuthUser>("/auth/me", {
    accessToken
  });
}

async function requestRefresh(refreshToken: string): Promise<AuthTokensResponse> {
  return apiRequest<AuthTokensResponse>("/auth/refresh", {
    method: "POST",
    body: {
      refresh_token: refreshToken
    }
  });
}

async function requestCompanies(accessToken: string): Promise<Company[]> {
  const response = await apiRequest<{ items: Company[] }>("/companies", {
    accessToken
  });
  return Array.isArray(response.items) ? sortCompanies(response.items) : [];
}

function loadStoredActiveCompanyId(): number | null {
  if (typeof window === "undefined") {
    return null;
  }

  const rawValue = localStorage.getItem(ACTIVE_COMPANY_STORAGE_KEY);
  if (!rawValue) {
    return null;
  }

  const parsed = Number.parseInt(rawValue, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
}

function storeActiveCompanyId(value: number | null): void {
  if (typeof window === "undefined") {
    return;
  }

  if (value === null) {
    localStorage.removeItem(ACTIVE_COMPANY_STORAGE_KEY);
    document.cookie = `${ACTIVE_COMPANY_COOKIE}=; path=/; max-age=0; samesite=lax`;
    return;
  }

  localStorage.setItem(ACTIVE_COMPANY_STORAGE_KEY, String(value));
  document.cookie = `${ACTIVE_COMPANY_COOKIE}=${encodeURIComponent(String(value))}; path=/; max-age=${60 * 60 * 24 * 30}; samesite=lax`;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isReady, setIsReady] = useState<boolean>(false);
  const [user, setUser] = useState<AuthUser | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [companies, setCompanies] = useState<Company[]>([]);
  const [activeCompanyId, setActiveCompanyIdState] = useState<number | null>(null);

  const setActiveCompanyId = useCallback((value: number | null) => {
    setActiveCompanyIdState(value);
    storeActiveCompanyId(value);
  }, []);

  const hydrateSuperAdminState = useCallback(
    async (token: string, authUser: AuthUser) => {
      if (authUser.role !== "super_admin") {
        setCompanies([]);
        setActiveCompanyId(null);
        return;
      }

      try {
        const loadedCompanies = await requestCompanies(token);
        setCompanies(loadedCompanies);

        const stored = loadStoredActiveCompanyId();
        const fallback = loadedCompanies[0]?.id ?? null;
        const matched = loadedCompanies.find((company) => company.id === stored);
        setActiveCompanyId(matched?.id ?? fallback);
      } catch {
        setCompanies([]);
        setActiveCompanyId(null);
      }
    },
    [setActiveCompanyId]
  );

  const resetAuthState = useCallback(() => {
    setUser(null);
    setAccessToken(null);
    setCompanies([]);
    setActiveCompanyId(null);
    clearAuthTokens();
  }, [setActiveCompanyId]);

  const refreshMe = useCallback(async () => {
    const token = readAccessToken();
    const refreshToken = readRefreshToken();

    if (!token) {
      resetAuthState();
      setIsReady(true);
      return;
    }

    try {
      const authUser = await requestMe(token);
      setUser(authUser);
      setAccessToken(token);
      await hydrateSuperAdminState(token, authUser);
      setIsReady(true);
      return;
    } catch (error) {
      if (!(error instanceof ApiError) || error.status !== 401 || !refreshToken) {
        resetAuthState();
        setIsReady(true);
        return;
      }
    }

    try {
      const refreshed = await requestRefresh(refreshToken);
      persistAuthTokens(
        refreshed.access_token,
        refreshed.refresh_token,
        refreshed.expires_in
      );
      setUser(refreshed.user);
      setAccessToken(refreshed.access_token);
      await hydrateSuperAdminState(refreshed.access_token, refreshed.user);
    } catch {
      resetAuthState();
    } finally {
      setIsReady(true);
    }
  }, [hydrateSuperAdminState, resetAuthState]);

  useEffect(() => {
    void refreshMe();
  }, [refreshMe]);

  const loginWithPassword = useCallback(
    async (request: LoginRequest) => {
      const response = await apiRequest<AuthTokensResponse>("/auth/login", {
        method: "POST",
        body: request
      });

      persistAuthTokens(
        response.access_token,
        response.refresh_token,
        response.expires_in
      );
      setUser(response.user);
      setAccessToken(response.access_token);
      await hydrateSuperAdminState(response.access_token, response.user);
      setIsReady(true);
    },
    [hydrateSuperAdminState]
  );

  const logout = useCallback(async () => {
    const token = readAccessToken();
    const refreshToken = readRefreshToken();

    try {
      if (token) {
        await apiRequest<void>("/auth/logout", {
          method: "POST",
          accessToken: token,
          body: refreshToken ? { refresh_token: refreshToken } : undefined
        });
      }
    } catch {
      // client-side logout still clears local session
    }

    resetAuthState();
    setIsReady(true);
  }, [resetAuthState]);

  const contextValue = useMemo<AuthContextValue>(
    () => ({
      isReady,
      isAuthenticated: user !== null,
      user,
      accessToken,
      companies,
      activeCompanyId,
      setActiveCompanyId,
      loginWithPassword,
      logout,
      refreshMe
    }),
    [
      activeCompanyId,
      accessToken,
      companies,
      isReady,
      loginWithPassword,
      logout,
      refreshMe,
      setActiveCompanyId,
      user
    ]
  );

  const Provider = AuthContext.Provider as React.ComponentType<{
    value: AuthContextValue | null;
    children?: React.ReactNode;
  }>;
  return <Provider value={contextValue}>{children}</Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}
