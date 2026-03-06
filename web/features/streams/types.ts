import type { CheckStatus } from "@/lib/api/types";

export type IsActiveFilter = "all" | "true" | "false";
export type StatusFilter = "all" | CheckStatus;

export type LatestStatusMap = Record<
  number,
  {
    status: CheckStatus | null;
    createdAt: string | null;
  }
>;

export const STORAGE_LAST_SECTION_KEY = "core.lastSection";
