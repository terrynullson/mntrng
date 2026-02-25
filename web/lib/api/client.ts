import type { ApiErrorEnvelope } from "./types";

type HttpMethod = "GET" | "POST" | "PATCH" | "DELETE";

type ApiRequestOptions = {
  method?: HttpMethod;
  accessToken?: string | null;
  body?: unknown;
  signal?: AbortSignal;
};

const API_BASE_PATH = "/api/v1";

export class ApiError extends Error {
  status: number;
  code: string;
  details: Record<string, unknown> | undefined;

  constructor(
    status: number,
    code: string,
    message: string,
    details?: Record<string, unknown>
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function toErrorEnvelope(payload: unknown): ApiErrorEnvelope {
  if (!isRecord(payload)) {
    return {};
  }

  return {
    code: typeof payload.code === "string" ? payload.code : undefined,
    message: typeof payload.message === "string" ? payload.message : undefined,
    details: isRecord(payload.details)
      ? (payload.details as Record<string, unknown>)
      : undefined,
    request_id:
      typeof payload.request_id === "string" ? payload.request_id : undefined
  };
}

export function toErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.code ? `${error.message} (${error.code})` : error.message;
  }

  if (error instanceof Error) {
    return error.message;
  }

  return "Unexpected client error";
}

export async function apiRequest<T>(
  path: string,
  options: ApiRequestOptions = {}
): Promise<T> {
  const method = options.method ?? "GET";
  const headers: HeadersInit = {
    Accept: "application/json"
  };

  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }

  if (options.accessToken) {
    headers.Authorization = `Bearer ${options.accessToken}`;
  }

  const response = await fetch(`${API_BASE_PATH}${path}`, {
    method,
    headers,
    cache: "no-store",
    credentials: "same-origin",
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
    signal: options.signal
  });

  let payload: unknown = null;
  const hasJsonContentType =
    response.headers.get("content-type")?.includes("application/json") ?? false;

  if (hasJsonContentType) {
    try {
      payload = await response.json();
    } catch {
      payload = null;
    }
  }

  if (!response.ok) {
    const envelope = toErrorEnvelope(payload);
    throw new ApiError(
      response.status,
      envelope.code ?? "http_error",
      envelope.message ?? `Request failed with status ${response.status}`,
      envelope.details
    );
  }

  return payload as T;
}
