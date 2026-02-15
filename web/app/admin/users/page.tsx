"use client";

import { FormEvent, useMemo, useState } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { AppButton } from "@/components/ui/app-button";
import { StatePanel } from "@/components/ui/state-panel";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import type { AuthUser, ChangeUserRoleRequest } from "@/lib/api/types";

export default function AdminUsersPage() {
  const { user, accessToken } = useAuth();

  const [targetUserID, setTargetUserID] = useState<string>("");
  const [targetCompanyID, setTargetCompanyID] = useState<string>("");
  const [targetRole, setTargetRole] = useState<"company_admin" | "viewer">("viewer");
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [managedUsers, setManagedUsers] = useState<AuthUser[]>([]);

  const isSuperAdmin = user?.role === "super_admin";

  const tableUsers = useMemo(() => {
    const users: AuthUser[] = [];

    if (user) {
      users.push(user);
    }

    managedUsers.forEach((managedUser) => {
      if (!users.some((item) => item.id === managedUser.id)) {
        users.push(managedUser);
      }
    });

    return users;
  }, [managedUsers, user]);

  const handleRoleChange = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!accessToken || !isSuperAdmin) {
      setError("Only super_admin can change user roles.");
      return;
    }

    const userID = Number.parseInt(targetUserID, 10);
    const companyID = Number.parseInt(targetCompanyID, 10);
    if (!Number.isFinite(userID) || userID <= 0) {
      setError("user_id must be a positive number.");
      return;
    }
    if (!Number.isFinite(companyID) || companyID <= 0) {
      setError("company_id must be a positive number.");
      return;
    }

    const payload: ChangeUserRoleRequest = {
      role: targetRole,
      company_id: companyID
    };

    setError(null);
    setSuccess(null);
    setIsSubmitting(true);

    try {
      const updated = await apiRequest<AuthUser>(`/admin/users/${userID}/role`, {
        method: "PATCH",
        accessToken,
        body: payload
      });

      setManagedUsers((previous) => {
        const withoutCurrent = previous.filter((item) => item.id !== updated.id);
        return [updated, ...withoutCurrent];
      });
      setSuccess(`Updated user #${updated.id} role to ${updated.role}.`);
    } catch (submitError) {
      setError(toErrorMessage(submitError));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">Users</h2>
        <p className="page-note">
          Role and status management. Role editing is available only for super_admin.
        </p>
      </header>

      {!isSuperAdmin ? (
        <StatePanel>
          Read-only mode. Your role does not permit changing user role assignments.
        </StatePanel>
      ) : (
        <form className="role-form" onSubmit={handleRoleChange}>
          <label className="form-field" htmlFor="role-user-id">
            <span>User ID</span>
            <input
              id="role-user-id"
              type="number"
              value={targetUserID}
              onChange={(event) => setTargetUserID(event.target.value)}
              disabled={isSubmitting}
            />
          </label>

          <label className="form-field" htmlFor="role-company-id">
            <span>Company ID</span>
            <input
              id="role-company-id"
              type="number"
              value={targetCompanyID}
              onChange={(event) => setTargetCompanyID(event.target.value)}
              disabled={isSubmitting}
            />
          </label>

          <label className="form-field" htmlFor="role-value">
            <span>Role</span>
            <select
              id="role-value"
              value={targetRole}
              onChange={(event) =>
                setTargetRole(event.target.value as "company_admin" | "viewer")
              }
              disabled={isSubmitting}
            >
              <option value="viewer">viewer</option>
              <option value="company_admin">company_admin</option>
            </select>
          </label>

          <div className="role-form-action">
            <AppButton type="submit" disabled={isSubmitting}>
              {isSubmitting ? "Updating..." : "Apply role"}
            </AppButton>
          </div>
        </form>
      )}

      {error ? <StatePanel kind="error">{error}</StatePanel> : null}
      {success ? <StatePanel>{success}</StatePanel> : null}

      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>User ID</th>
              <th>Login</th>
              <th>Email</th>
              <th>Role</th>
              <th>Status</th>
              <th>Company ID</th>
            </tr>
          </thead>
          <tbody>
            {tableUsers.length === 0 ? (
              <tr>
                <td colSpan={6}>No user data available.</td>
              </tr>
            ) : (
              tableUsers.map((item) => (
                <tr key={item.id}>
                  <td>{item.id}</td>
                  <td>{item.login}</td>
                  <td>{item.email}</td>
                  <td>{item.role}</td>
                  <td>{item.status}</td>
                  <td>{item.company_id ?? "-"}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
