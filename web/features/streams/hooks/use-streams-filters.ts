"use client";

import { useMemo, useState } from "react";
import type { Stream } from "@/lib/api/types";
import type { LatestStatusMap } from "../types";
import type { IsActiveFilter, StatusFilter } from "../types";

type FavoriteMap = Map<number, { isPinned: boolean; sortOrder: number }>;

function buildFavoriteMap(
  favorites: Array<{ stream: { id: number }; is_pinned: boolean; sort_order: number }>
): FavoriteMap {
  const map = new Map<number, { isPinned: boolean; sortOrder: number }>();
  favorites.forEach((fav) => {
    map.set(fav.stream.id, {
      isPinned: fav.is_pinned,
      sortOrder: fav.sort_order
    });
  });
  return map;
}

export function useStreamsFilters(
  streams: Stream[],
  latestStatusMap: LatestStatusMap,
  favorites: Array<{ stream: { id: number }; is_pinned: boolean; sort_order: number }>
) {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");

  const favoriteMap = useMemo(() => buildFavoriteMap(favorites), [favorites]);

  const filteredStreams = useMemo(() => {
    const needle = search.trim().toLowerCase();

    const filtered = streams.filter((stream) => {
      const streamStatus = latestStatusMap[stream.id]?.status;

      if (statusFilter !== "all" && streamStatus !== statusFilter) {
        return false;
      }

      if (!needle) {
        return true;
      }

      return (
        stream.name.toLowerCase().includes(needle) ||
        String(stream.id).includes(needle) ||
        String(stream.project_id).includes(needle)
      );
    });

    return [...filtered].sort((a, b) => {
      const favA = favoriteMap.get(a.id);
      const favB = favoriteMap.get(b.id);
      const pinnedA = favA?.isPinned ?? false;
      const pinnedB = favB?.isPinned ?? false;
      if (pinnedA && !pinnedB) return -1;
      if (!pinnedA && pinnedB) return 1;
      if (pinnedA && pinnedB) {
        return (favA?.sortOrder ?? 0) - (favB?.sortOrder ?? 0);
      }
      const favOnlyA = favA != null;
      const favOnlyB = favB != null;
      if (favOnlyA && !favOnlyB) return -1;
      if (!favOnlyA && favOnlyB) return 1;
      return a.id - b.id;
    });
  }, [latestStatusMap, search, statusFilter, streams, favoriteMap]);

  return {
    search,
    setSearch,
    statusFilter,
    setStatusFilter,
    favoriteMap,
    filteredStreams
  };
}
