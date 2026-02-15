"use client";

import { useEffect, useState } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import type { RegistrationRequest } from "@/lib/api/types";

function formatTimestamp(timestamp: string): string {
  const parsed = new Date(timestamp);
  return Number.isNaN(parsed.getTime()) ? timestamp : parsed.toLocaleString();
}

export default function AdminRequestsPage() {
  const { user, accessToken } = useAuth();

  const [items, setItems] = useState<RegistrationRequest[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const isSuperAdmin = user?.role === "super_admin";

  const loadRequests = async () => {
    if (!accessToken || !isSuperAdmin) {
      setItems([]);
      setIsLoading(false);
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await apiRequest<{ items: RegistrationRequest[] }>(
        "/admin/registration-requests",
        { accessToken }
      );
      setItems(Array.isArray(response.items) ? response.items : []);
    } catch (loadError) {
      setError(toErrorMessage(loadError));
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    void loadRequests();
    // accessToken change should reload list
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [accessToken, isSuperAdmin]);

  return (
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">Pending Registration Requests</h2>
        <p className="page-note">
          Baseline read-only queue for controlled sign-up requests.
        </p>
      </header>

      {!isSuperAdmin ? (
        <StatePanel kind="error">
          Access denied. Only super_admin can manage registration requests.
        </StatePanel>
      ) : null}

      {isLoading ? <SkeletonBlock lines={6} /> : null}
      {error ? <StatePanel kind="error">{error}</StatePanel> : null}

      {!isLoading && !error && isSuperAdmin && items.length === 0 ? (
        <StatePanel>No pending registration requests.</StatePanel>
      ) : null}

      {!isLoading && !error && isSuperAdmin && items.length > 0 ? (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Company</th>
                <th>Email</th>
                <th>Login</th>
                <th>Requested role</th>
                <th>Created at</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.id}>
                  <td>{item.id}</td>
                  <td>{item.company_id}</td>
                  <td>{item.email}</td>
                  <td>{item.login}</td>
                  <td>{item.requested_role}</td>
                  <td>{formatTimestamp(item.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}
