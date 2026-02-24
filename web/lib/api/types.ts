export type Role = "super_admin" | "company_admin" | "viewer";

export type UserStatus = "active" | "disabled";

export type ApiErrorEnvelope = {
  code?: string;
  message?: string;
  details?: Record<string, unknown>;
  request_id?: string;
};

export type ApiListResponse<T> = {
  items: T[];
  next_cursor: string | null;
};

export type AuthUser = {
  id: number;
  company_id: number | null;
  email: string;
  login: string;
  role: Role;
  status: UserStatus;
  created_at: string;
  updated_at: string;
};

export type AuthTokensResponse = {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  user: AuthUser;
};

export type LoginRequest = {
  login_or_email: string;
  password: string;
};

export type RegisterRequest = {
  company_id: number;
  email: string;
  login: string;
  password: string;
  requested_role: Extract<Role, "company_admin" | "viewer">;
};

export type RegistrationRequest = {
  id: number;
  company_id: number;
  email: string;
  login: string;
  requested_role: Extract<Role, "company_admin" | "viewer">;
  status: "pending" | "approved" | "rejected";
  created_at: string;
  updated_at: string;
  processed_at?: string | null;
  processed_by_user_id?: number | null;
  decision_reason?: string | null;
};

export type ApproveRegistrationRequest = {
  company_id: number;
  role: Extract<Role, "company_admin" | "viewer">;
};

export type RejectRegistrationRequest = {
  reason: string;
};

export type ChangeUserRoleRequest = {
  role: Extract<Role, "company_admin" | "viewer">;
  company_id: number;
};

export type ChangeUserStatusRequest = {
  status: UserStatus;
};

export type AdminUsersListFilters = {
  company_id?: number;
  role?: Role;
  status?: UserStatus;
  limit?: number;
};

export type Company = {
  id: number;
  name: string;
  created_at: string;
};

export type Project = {
  id: number;
  company_id: number;
  name: string;
  created_at: string;
  updated_at: string;
};

export type Stream = {
  id: number;
  company_id: number;
  project_id: number;
  name: string;
  source_type: "HLS" | "EMBED";
  source_url: string;
  url: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type StreamCreateRequest = {
  project_id: number;
  name: string;
  source_type: "HLS" | "EMBED";
  source_url: string;
  is_active: boolean;
};

export type StreamPatchRequest = {
  name?: string;
  source_type?: "HLS" | "EMBED";
  source_url?: string;
  is_active?: boolean;
};

export type EmbedWhitelistItem = {
  id: number;
  company_id: number;
  domain: string;
  enabled: boolean;
  created_at: string;
  created_by_user_id?: number | null;
};

export type CheckJob = {
  id: number;
  company_id: number;
  stream_id: number;
  planned_at: string;
  status: "queued" | "running" | "done" | "failed";
  created_at: string;
  started_at: string | null;
  finished_at: string | null;
  error_message: string | null;
};

export type EnqueueCheckJobResponse = {
  job: CheckJob;
};

export type CheckStatus = "OK" | "WARN" | "FAIL";

export type CheckResultChecks = {
  playlist?: CheckStatus;
  segments?: CheckStatus;
  freshness?: CheckStatus;
  declared_bitrate?: CheckStatus;
  effective_bitrate?: CheckStatus;
  freeze?: CheckStatus;
  blackframe?: CheckStatus;
};

export type CheckResult = {
  id: number;
  company_id: number;
  job_id: number;
  stream_id: number;
  status: CheckStatus;
  checks: CheckResultChecks;
  screenshot_path: string | null;
  created_at: string;
};

export type AiIncident = {
  cause: string;
  summary: string;
};

export type TelegramLinkPayload = Record<string, string>;

export type TelegramDeliverySettings = {
  is_enabled: boolean;
  chat_id: string;
  send_recovered: boolean;
  created_at?: string;
  updated_at?: string;
};

export type TelegramDeliverySettingsPatch = {
  is_enabled?: boolean;
  chat_id?: string;
  send_recovered?: boolean;
};

export type StreamWithFavorite = {
  stream: Stream;
  is_pinned: boolean;
  sort_order: number;
};

export type Incident = {
  id: number;
  company_id: number;
  stream_id: number;
  stream_name?: string;
  status: "open" | "resolved";
  severity: "warn" | "fail";
  started_at: string;
  last_event_at: string;
  resolved_at?: string | null;
  fail_reason?: string | null;
  sample_screenshot_path?: string | null;
  has_screenshot?: boolean;
  screenshot_taken_at?: string | null;
  diag_code?: "BLACKFRAME" | "FREEZE" | "CAPTURE_FAIL" | "UNKNOWN" | null;
  diag_details?: Record<string, unknown> | null;
  last_check_id?: number | null;
};

export type IncidentListResponse = {
  items: Incident[];
  next_cursor?: string | null;
  total: number;
};
