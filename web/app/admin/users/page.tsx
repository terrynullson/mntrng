"use client";

import { useMemo } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { StatePanel } from "@/components/ui/state-panel";
import type { AuthUser } from "@/lib/api/types";

export default function AdminUsersPage() {
  const { user } = useAuth();

  const isSuperAdmin = user?.role === "super_admin";

  const tableUsers = useMemo(() => {
    const users: AuthUser[] = [];

    if (user) {
      users.push(user);
    }

    return users;
  }, [user]);

  return (
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">Users</h2>
        <p className="page-note">
          Baseline role-aware user view from authenticated context.
        </p>
      </header>

      {isSuperAdmin ? (
        <StatePanel>
          Super admin authenticated. User management actions are deferred to the
          next phase.
        </StatePanel>
      ) : (
        <StatePanel>Read-only mode for current role.</StatePanel>
      )}

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
