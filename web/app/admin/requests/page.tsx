"use client";

import { motion } from "framer-motion";
import { useEffect, useState } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { AppButton } from "@/components/ui/app-button";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import type {
  ApproveRegistrationRequest,
  AuthUser,
  RegistrationRequest
} from "@/lib/api/types";

function formatTimestamp(timestamp: string): string {
  const parsed = new Date(timestamp);
  return Number.isNaN(parsed.getTime()) ? timestamp : parsed.toLocaleString();
}

export default function AdminRequestsPage() {
  const { user, accessToken } = useAuth();

  const [items, setItems] = useState<RegistrationRequest[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [busyRequestID, setBusyRequestID] = useState<number | null>(null);

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

  const handleApprove = async (requestItem: RegistrationRequest) => {
    if (!accessToken || !isSuperAdmin) {
      return;
    }

    const payload: ApproveRegistrationRequest = {
      company_id: requestItem.company_id,
      role: requestItem.requested_role
    };

    setBusyRequestID(requestItem.id);
    try {
      await apiRequest<AuthUser>(
        `/admin/registration-requests/${requestItem.id}/approve`,
        {
          method: "POST",
          accessToken,
          body: payload
        }
      );
      await loadRequests();
    } catch (approveError) {
      setError(toErrorMessage(approveError));
    } finally {
      setBusyRequestID(null);
    }
  };

  const handleReject = async (requestItem: RegistrationRequest) => {
    if (!accessToken || !isSuperAdmin) {
      return;
    }

    const reason = window.prompt("Reject reason (optional):", "");

    setBusyRequestID(requestItem.id);
    try {
      await apiRequest<void>(`/admin/registration-requests/${requestItem.id}/reject`, {
        method: "POST",
        accessToken,
        body: {
          reason: reason ?? ""
        }
      });
      await loadRequests();
    } catch (rejectError) {
      setError(toErrorMessage(rejectError));
    } finally {
      setBusyRequestID(null);
    }
  };

  return (
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">Pending Registration Requests</h2>
        <p className="page-note">Approve or reject controlled sign-up requests.</p>
      </header>

      {!isSuperAdmin ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel>
            Read-only mode. Request moderation actions are available only for
            super_admin.
          </StatePanel>
        </motion.div>
      ) : null}

      {isLoading ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.2, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <SkeletonBlock lines={6} />
        </motion.div>
      ) : null}
      {error ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel kind="error">{error}</StatePanel>
        </motion.div>
      ) : null}

      {!isLoading && !error && isSuperAdmin && items.length === 0 ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <StatePanel>No pending registration requests.</StatePanel>
        </motion.div>
      ) : null}

      {!isLoading && !error && isSuperAdmin && items.length > 0 ? (
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
                <th>Company</th>
                <th>Email</th>
                <th>Login</th>
                <th>Requested role</th>
                <th>Created at</th>
                <th>Actions</th>
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
                  <td className="table-actions">
                    <AppButton
                      type="button"
                      variant="secondary"
                      disabled={busyRequestID === item.id}
                      onClick={() => {
                        void handleApprove(item);
                      }}
                    >
                      Approve
                    </AppButton>
                    <AppButton
                      type="button"
                      variant="danger"
                      disabled={busyRequestID === item.id}
                      onClick={() => {
                        void handleReject(item);
                      }}
                    >
                      Reject
                    </AppButton>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </motion.div>
      ) : null}
    </section>
  );
}
