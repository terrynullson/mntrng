"use client";

import { motion } from "framer-motion";
import { FormEvent, useEffect, useState } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { AppButton } from "@/components/ui/app-button";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import type {
  AdminUsersListFilters,
  ApiListResponse,
  AuthUser,
  ChangeUserRoleRequest,
  ChangeUserStatusRequest,
  Role,
  UserStatus
} from "@/lib/api/types";

type RoleFilter = "all" | Role;
type StatusFilter = "all" | UserStatus;
type EditableRole = Extract<Role, "company_admin" | "viewer">;

type AppliedFilters = {
  companyID: number | null;
  role: RoleFilter;
  status: StatusFilter;
  limit: number;
};

const DEFAULT_LIMIT = 50;

function formatTimestamp(timestamp: string): string {
  const parsed = new Date(timestamp);
  return Number.isNaN(parsed.getTime()) ? timestamp : parsed.toLocaleString();
}

function asEditableRole(value: string): EditableRole {
  return value === "company_admin" ? "company_admin" : "viewer";
}

function buildUsersQuery(filters: AppliedFilters): string {
  const params = new URLSearchParams();
  const requestFilters: AdminUsersListFilters = { limit: filters.limit };

  if (filters.companyID !== null) {
    requestFilters.company_id = filters.companyID;
  }
  if (filters.role !== "all") {
    requestFilters.role = filters.role;
  }
  if (filters.status !== "all") {
    requestFilters.status = filters.status;
  }

  if (requestFilters.company_id !== undefined) {
    params.set("company_id", String(requestFilters.company_id));
  }
  if (requestFilters.role) {
    params.set("role", requestFilters.role);
  }
  if (requestFilters.status) {
    params.set("status", requestFilters.status);
  }
  if (requestFilters.limit !== undefined) {
    params.set("limit", String(requestFilters.limit));
  }

  const query = params.toString();
  return query ? `/admin/users?${query}` : "/admin/users";
}

export default function AdminUsersPage() {
  const { user, accessToken, companies } = useAuth();
  const isSuperAdmin = user?.role === "super_admin";

  const [items, setItems] = useState<AuthUser[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionSuccess, setActionSuccess] = useState<string | null>(null);

  const [companyFilterDraft, setCompanyFilterDraft] = useState<string>("");
  const [roleFilterDraft, setRoleFilterDraft] = useState<RoleFilter>("all");
  const [statusFilterDraft, setStatusFilterDraft] = useState<StatusFilter>("all");
  const [limitDraft, setLimitDraft] = useState<string>(String(DEFAULT_LIMIT));

  const [appliedFilters, setAppliedFilters] = useState<AppliedFilters>({
    companyID: null,
    role: "all",
    status: "all",
    limit: DEFAULT_LIMIT
  });

  const [roleDrafts, setRoleDrafts] = useState<Record<number, EditableRole>>({});
  const [companyDrafts, setCompanyDrafts] = useState<Record<number, string>>({});
  const [statusDrafts, setStatusDrafts] = useState<Record<number, UserStatus>>({});
  const [busyRoleUserID, setBusyRoleUserID] = useState<number | null>(null);
  const [busyStatusUserID, setBusyStatusUserID] = useState<number | null>(null);

  useEffect(() => {
    const loadUsers = async () => {
      if (!isSuperAdmin) {
        setItems(user ? [user] : []);
        setError(null);
        setIsLoading(false);
        return;
      }

      if (!accessToken) {
        setItems([]);
        setIsLoading(false);
        return;
      }

      setIsLoading(true);
      setError(null);
      setActionError(null);

      try {
        const response = await apiRequest<ApiListResponse<AuthUser>>(
          buildUsersQuery(appliedFilters),
          { accessToken }
        );
        setItems(Array.isArray(response.items) ? response.items : []);
      } catch (loadError) {
        setError(toErrorMessage(loadError));
      } finally {
        setIsLoading(false);
      }
    };

    void loadUsers();
  }, [accessToken, appliedFilters, isSuperAdmin, user]);

  useEffect(() => {
    const nextRoleDrafts: Record<number, EditableRole> = {};
    const nextCompanyDrafts: Record<number, string> = {};
    const nextStatusDrafts: Record<number, UserStatus> = {};

    items.forEach((item) => {
      if (item.role === "company_admin" || item.role === "viewer") {
        nextRoleDrafts[item.id] = item.role;
        nextCompanyDrafts[item.id] = item.company_id ? String(item.company_id) : "";
      }
      nextStatusDrafts[item.id] = item.status;
    });

    setRoleDrafts(nextRoleDrafts);
    setCompanyDrafts(nextCompanyDrafts);
    setStatusDrafts(nextStatusDrafts);
  }, [items]);

  const applyFilters = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const parsedLimit = Number.parseInt(limitDraft, 10);
    if (!Number.isFinite(parsedLimit) || parsedLimit <= 0) {
      setError("limit must be a positive number.");
      return;
    }

    let parsedCompanyID: number | null = null;
    const companyValue = companyFilterDraft.trim();
    if (companyValue) {
      const candidate = Number.parseInt(companyValue, 10);
      if (!Number.isFinite(candidate) || candidate <= 0) {
        setError("company_id must be a positive number.");
        return;
      }
      parsedCompanyID = candidate;
    }

    setError(null);
    setActionSuccess(null);
    setAppliedFilters({
      companyID: parsedCompanyID,
      role: roleFilterDraft,
      status: statusFilterDraft,
      limit: parsedLimit
    });
  };

  const resetFilters = () => {
    setCompanyFilterDraft("");
    setRoleFilterDraft("all");
    setStatusFilterDraft("all");
    setLimitDraft(String(DEFAULT_LIMIT));
    setError(null);
    setAppliedFilters({
      companyID: null,
      role: "all",
      status: "all",
      limit: DEFAULT_LIMIT
    });
  };

  const applyUserUpdate = (updated: AuthUser) => {
    setItems((previous) =>
      previous.map((item) => (item.id === updated.id ? updated : item))
    );
  };

  const handleRoleChange = async (targetUser: AuthUser) => {
    if (!accessToken || !isSuperAdmin) {
      return;
    }
    if (targetUser.role === "super_admin") {
      setActionError("super_admin role cannot be edited from this panel.");
      return;
    }

    const roleDraft = roleDrafts[targetUser.id] ?? asEditableRole(targetUser.role);
    const companyRaw = (companyDrafts[targetUser.id] ?? "").trim();
    const companyID = Number.parseInt(companyRaw, 10);
    if (!Number.isFinite(companyID) || companyID <= 0) {
      setActionError("company_id must be a positive number for role update.");
      return;
    }

    const payload: ChangeUserRoleRequest = {
      role: roleDraft,
      company_id: companyID
    };

    setBusyRoleUserID(targetUser.id);
    setActionError(null);
    setActionSuccess(null);

    try {
      const updated = await apiRequest<AuthUser>(`/admin/users/${targetUser.id}/role`, {
        method: "PATCH",
        accessToken,
        body: payload
      });
      applyUserUpdate(updated);
      setActionSuccess(`Role updated for user #${updated.id}.`);
    } catch (requestError) {
      setActionError(toErrorMessage(requestError));
    } finally {
      setBusyRoleUserID(null);
    }
  };

  const handleStatusChange = async (targetUser: AuthUser) => {
    if (!accessToken || !isSuperAdmin) {
      return;
    }
    if (targetUser.role === "super_admin") {
      setActionError("super_admin status is not managed in tenant scope.");
      return;
    }

    const status = statusDrafts[targetUser.id] ?? targetUser.status;
    const payload: ChangeUserStatusRequest = { status };

    setBusyStatusUserID(targetUser.id);
    setActionError(null);
    setActionSuccess(null);

    try {
      const updated = await apiRequest<AuthUser>(
        `/admin/users/${targetUser.id}/status`,
        {
          method: "PATCH",
          accessToken,
          body: payload
        }
      );
      applyUserUpdate(updated);
      setActionSuccess(`Status updated for user #${updated.id}.`);
    } catch (requestError) {
      setActionError(toErrorMessage(requestError));
    } finally {
      setBusyStatusUserID(null);
    }
  };

  return (
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">Users</h2>
        <p className="page-note">
          Admin users list with filters and RBAC-limited management actions.
        </p>
      </header>

      {!isSuperAdmin ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel>
            Read-only mode. Role and status mutations are available only for super_admin.
          </StatePanel>
        </motion.div>
      ) : (
        <form className="filters-grid users-filters" onSubmit={applyFilters}>
          <label className="form-field" htmlFor="users-company-filter">
            <span>Company</span>
            <select
              id="users-company-filter"
              value={companyFilterDraft}
              onChange={(event) => setCompanyFilterDraft(event.target.value)}
              disabled={isLoading}
            >
              <option value="">All companies</option>
              {companies.map((company) => (
                <option key={company.id} value={company.id}>
                  {company.name} ({company.id})
                </option>
              ))}
            </select>
          </label>

          <label className="form-field" htmlFor="users-role-filter">
            <span>Role</span>
            <select
              id="users-role-filter"
              value={roleFilterDraft}
              onChange={(event) => setRoleFilterDraft(event.target.value as RoleFilter)}
              disabled={isLoading}
            >
              <option value="all">All roles</option>
              <option value="super_admin">super_admin</option>
              <option value="company_admin">company_admin</option>
              <option value="viewer">viewer</option>
            </select>
          </label>

          <label className="form-field" htmlFor="users-status-filter">
            <span>Status</span>
            <select
              id="users-status-filter"
              value={statusFilterDraft}
              onChange={(event) =>
                setStatusFilterDraft(event.target.value as StatusFilter)
              }
              disabled={isLoading}
            >
              <option value="all">All statuses</option>
              <option value="active">active</option>
              <option value="disabled">disabled</option>
            </select>
          </label>

          <label className="form-field" htmlFor="users-limit-filter">
            <span>Limit</span>
            <input
              id="users-limit-filter"
              type="number"
              min={1}
              max={200}
              value={limitDraft}
              onChange={(event) => setLimitDraft(event.target.value)}
              disabled={isLoading}
            />
          </label>

          <div className="users-filter-actions">
            <AppButton type="submit" variant="secondary" disabled={isLoading}>
              Apply filters
            </AppButton>
            <AppButton
              type="button"
              variant="ghost"
              disabled={isLoading}
              onClick={resetFilters}
            >
              Reset
            </AppButton>
          </div>
        </form>
      )}

      {error ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel kind="error">{error}</StatePanel>
        </motion.div>
      ) : null}
      {actionError ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel kind="error">{actionError}</StatePanel>
        </motion.div>
      ) : null}
      {actionSuccess ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel>{actionSuccess}</StatePanel>
        </motion.div>
      ) : null}
      {isLoading ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.2, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <SkeletonBlock lines={7} />
        </motion.div>
      ) : null}

      {!isLoading && !error && isSuperAdmin && items.length === 0 ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <StatePanel>No users found for selected filters.</StatePanel>
        </motion.div>
      ) : null}

      {!isLoading && !error && items.length > 0 ? (
        <motion.div
          className="table-wrap"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.28, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Login</th>
                <th>Email</th>
                <th>Role</th>
                <th>Status</th>
                <th>Company</th>
                <th>Updated at</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => {
                const canMutateRow = isSuperAdmin && item.role !== "super_admin";
                const roleValue =
                  roleDrafts[item.id] ?? asEditableRole(item.role);
                const companyValue =
                  companyDrafts[item.id] ??
                  (item.company_id ? String(item.company_id) : "");
                const statusValue = statusDrafts[item.id] ?? item.status;

                return (
                  <tr key={item.id}>
                    <td>{item.id}</td>
                    <td>{item.login}</td>
                    <td>{item.email}</td>
                    <td>{item.role}</td>
                    <td>{item.status}</td>
                    <td>{item.company_id ?? "-"}</td>
                    <td>{formatTimestamp(item.updated_at)}</td>
                    <td>
                      {canMutateRow ? (
                        <div className="users-actions">
                          <div className="users-action-row">
                            <select
                              aria-label={`Role for user ${item.id}`}
                              value={roleValue}
                              onChange={(event) =>
                                setRoleDrafts((previous) => ({
                                  ...previous,
                                  [item.id]: asEditableRole(event.target.value)
                                }))
                              }
                              disabled={
                                busyRoleUserID === item.id ||
                                busyStatusUserID === item.id
                              }
                            >
                              <option value="company_admin">company_admin</option>
                              <option value="viewer">viewer</option>
                            </select>
                            <input
                              aria-label={`Company for user ${item.id}`}
                              type="number"
                              min={1}
                              value={companyValue}
                              onChange={(event) =>
                                setCompanyDrafts((previous) => ({
                                  ...previous,
                                  [item.id]: event.target.value
                                }))
                              }
                              disabled={
                                busyRoleUserID === item.id ||
                                busyStatusUserID === item.id
                              }
                              placeholder="Company ID"
                            />
                            <AppButton
                              type="button"
                              variant="secondary"
                              disabled={
                                busyRoleUserID === item.id ||
                                busyStatusUserID === item.id
                              }
                              onClick={() => {
                                void handleRoleChange(item);
                              }}
                            >
                              {busyRoleUserID === item.id ? "Saving..." : "Save role"}
                            </AppButton>
                          </div>

                          <div className="users-action-row">
                            <select
                              aria-label={`Status for user ${item.id}`}
                              value={statusValue}
                              onChange={(event) =>
                                setStatusDrafts((previous) => ({
                                  ...previous,
                                  [item.id]: event.target.value as UserStatus
                                }))
                              }
                              disabled={
                                busyRoleUserID === item.id ||
                                busyStatusUserID === item.id
                              }
                            >
                              <option value="active">active</option>
                              <option value="disabled">disabled</option>
                            </select>
                            <AppButton
                              type="button"
                              variant="secondary"
                              disabled={
                                busyRoleUserID === item.id ||
                                busyStatusUserID === item.id
                              }
                              onClick={() => {
                                void handleStatusChange(item);
                              }}
                            >
                              {busyStatusUserID === item.id
                                ? "Saving..."
                                : "Save status"}
                            </AppButton>
                          </div>
                        </div>
                      ) : (
                        <span className="status-muted">Read-only</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </motion.div>
      ) : null}
    </section>
  );
}
