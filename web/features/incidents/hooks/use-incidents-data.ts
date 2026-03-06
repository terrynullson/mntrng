"use client";

import { useCallback, useEffect, useState } from "react";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import type { IncidentListResponse } from "@/lib/api/types";
import type { SeverityFilter, StatusFilter } from "../types";

type UseIncidentsDataParams = {
  accessToken: string | null;
  scopeCompanyId: number | null;
  statusFilter: StatusFilter;
  severityFilter: SeverityFilter;
  searchApplied: string;
  page: number;
  pageSize: number;
};

export function useIncidentsData({
  accessToken,
  scopeCompanyId,
  statusFilter,
  severityFilter,
  searchApplied,
  page,
  pageSize
}: UseIncidentsDataParams) {
  const [data, setData] = useState<IncidentListResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!accessToken || !scopeCompanyId) {
      setData(null);
      setIsLoading(false);
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (statusFilter) params.set("status", statusFilter);
      if (severityFilter) params.set("severity", severityFilter);
      if (searchApplied.trim()) params.set("q", searchApplied.trim());
      params.set("page", String(page));
      params.set("page_size", String(pageSize));

      const response = await apiRequest<IncidentListResponse>(
        `/companies/${scopeCompanyId}/incidents?${params.toString()}`,
        { accessToken }
      );
      setData(response);
    } catch (e) {
      setError(toErrorMessage(e));
      setData(null);
    } finally {
      setIsLoading(false);
    }
  }, [
    accessToken,
    scopeCompanyId,
    statusFilter,
    severityFilter,
    searchApplied,
    page,
    pageSize
  ]);

  useEffect(() => {
    void load();
  }, [load]);

  return { data, isLoading, error, reload: load };
}
