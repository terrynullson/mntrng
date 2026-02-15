"use client";

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
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">Companies</h2>
        <p className="page-note">Super-admin company inventory.</p>
      </header>

      {!isSuperAdmin ? (
        <StatePanel kind="error">
          Access denied. Companies inventory is available only for super_admin.
        </StatePanel>
      ) : null}

      {isSuperAdmin ? (
        <label className="form-field company-search" htmlFor="company-search-input">
          <span>Search</span>
          <input
            id="company-search-input"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Find by id or name"
          />
        </label>
      ) : null}

      {isLoading ? <SkeletonBlock lines={6} /> : null}
      {error ? <StatePanel kind="error">{error}</StatePanel> : null}

      {!isLoading && !error && isSuperAdmin && filtered.length === 0 ? (
        <StatePanel>No companies found.</StatePanel>
      ) : null}

      {!isLoading && !error && isSuperAdmin && filtered.length > 0 ? (
        <div className="table-wrap">
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
        </div>
      ) : null}
    </section>
  );
}
