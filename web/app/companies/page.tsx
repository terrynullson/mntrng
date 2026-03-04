"use client";

import { motion } from "framer-motion";
import { useEffect, useMemo, useState } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import type { Company } from "@/lib/api/types";

export default function CompaniesPage() {
  const { user, accessToken } = useAuth();

  const [companies, setCompanies] = useState<Company[]>([]);
  const [search, setSearch] = useState<string>("");
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const isSuperAdmin = user?.role === "super_admin";

  useEffect(() => {
    if (!accessToken || !isSuperAdmin) {
      setIsLoading(false);
      return;
    }

    const abortController = new AbortController();

    setIsLoading(true);
    setError(null);

    apiRequest<{ items: Company[] }>("/companies", {
      accessToken,
      signal: abortController.signal
    })
      .then((response) => {
        setCompanies(Array.isArray(response.items) ? response.items : []);
      })
      .catch((loadError) => {
        if (abortController.signal.aborted) {
          return;
        }
        setError(toErrorMessage(loadError));
      })
      .finally(() => {
        if (!abortController.signal.aborted) {
          setIsLoading(false);
        }
      });

    return () => abortController.abort();
  }, [accessToken, isSuperAdmin]);

  const filtered = useMemo(() => {
    const needle = search.trim().toLowerCase();
    if (!needle) {
      return companies;
    }

    return companies.filter((company) => {
      return (
        company.name.toLowerCase().includes(needle) ||
        String(company.id).includes(needle)
      );
    });
  }, [companies, search]);

  return (
    <section className="panel premium-panel">
      <header className="page-header compact">
        <div>
          <h2 className="page-title">Companies</h2>
          <p className="page-note">Super-admin company inventory.</p>
        </div>
      </header>

      {!isSuperAdmin ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel kind="error">
            Access denied. Companies inventory is available only for super_admin.
          </StatePanel>
        </motion.div>
      ) : null}

      {isSuperAdmin ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.22, ease: "easeOut" }}
        >
          <label className="form-field company-search" htmlFor="company-search-input">
            <span>Search</span>
            <input
              id="company-search-input"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Find by id or name"
            />
          </label>
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

      {!isLoading && !error && isSuperAdmin && filtered.length === 0 ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <StatePanel>No companies found.</StatePanel>
        </motion.div>
      ) : null}

      {!isLoading && !error && isSuperAdmin && filtered.length > 0 ? (
        <motion.div
          className="card-table-wrap"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.28, ease: "easeOut" }}
        >
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Created at</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((company) => (
                <tr key={company.id}>
                  <td>{company.id}</td>
                  <td>{company.name}</td>
                  <td>{new Date(company.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </motion.div>
      ) : null}
    </section>
  );
}
