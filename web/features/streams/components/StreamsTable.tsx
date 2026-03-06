"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { Eye, Pencil, Pin, Play, Star, Trash2 } from "lucide-react";
import { IconButton } from "@/components/navigation/icon-button";
import { StatusBadge } from "@/components/ui/status-badge";
import type { Stream } from "@/lib/api/types";
import type { LatestStatusMap } from "../types";
import { formatTimestamp } from "../utils/format";

type FavoriteInfo = { isPinned: boolean; sortOrder: number };

type StreamsTableProps = {
  streams: Stream[];
  latestStatusMap: LatestStatusMap;
  favoriteMap: Map<number, FavoriteInfo>;
  isViewer: boolean;
  busyStreamID: number | null;
  busyFavoriteStreamID: number | null;
  onToggleFavorite: (stream: Stream) => void;
  onTogglePin: (stream: Stream) => void;
  onRunCheck: (stream: Stream) => void;
  onEdit: (stream: Stream) => void;
  onDelete: (stream: Stream) => void;
};

export function StreamsTable({
  streams,
  latestStatusMap,
  favoriteMap,
  isViewer,
  busyStreamID,
  busyFavoriteStreamID,
  onToggleFavorite,
  onTogglePin,
  onRunCheck,
  onEdit,
  onDelete
}: StreamsTableProps) {
  return (
    <motion.div
      className="card-table-wrap"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.28, ease: "easeOut" }}
    >
      <table>
        <thead>
          <tr>
            <th aria-label="Избранное и закрепление" />
            <th>ID</th>
            <th>Название</th>
            <th>Тип</th>
            <th>Проект</th>
            <th>Статус</th>
            <th>Последняя проверка</th>
            <th>Активен</th>
            <th>Обновлён</th>
            <th>Действия</th>
          </tr>
        </thead>
        <tbody>
          {streams.map((stream) => {
            const latestStatus = latestStatusMap[stream.id]?.status ?? null;
            const lastCheckAt = latestStatusMap[stream.id]?.createdAt ?? null;
            const fav = favoriteMap.get(stream.id);
            const isPinned = fav?.isPinned ?? false;
            const isFavorite = fav != null;
            const busyFav = busyFavoriteStreamID === stream.id;

            return (
              <tr
                key={stream.id}
                className={isPinned ? "stream-row-pinned" : undefined}
              >
                <td className="fav-pin-cell">
                  <IconButton
                    onClick={() => onToggleFavorite(stream)}
                    disabled={isViewer || busyFav}
                    aria-pressed={isFavorite}
                    label={isFavorite ? "Убрать из избранного" : "В избранное"}
                    tooltip={isFavorite ? "Убрать из избранного" : "В избранное"}
                  >
                    <Star size={16} fill={isFavorite ? "currentColor" : "none"} />
                  </IconButton>
                  <IconButton
                    onClick={() => onTogglePin(stream)}
                    disabled={isViewer || busyFav}
                    aria-pressed={isPinned}
                    label={isPinned ? "Открепить" : "Закрепить"}
                    tooltip={isPinned ? "Открепить" : "Закрепить"}
                  >
                    <Pin size={16} />
                  </IconButton>
                </td>
                <td>{stream.id}</td>
                <td>
                  <Link className="stream-link" href={`/monitoring/streams/${stream.id}`}>
                    {stream.name}
                  </Link>
                </td>
                <td>{stream.source_type}</td>
                <td>{stream.project_id}</td>
                <td>
                  {latestStatus ? (
                    <StatusBadge status={latestStatus} />
                  ) : (
                    <span className="status-muted">Нет данных</span>
                  )}
                </td>
                <td>{formatTimestamp(lastCheckAt)}</td>
                <td>{stream.is_active ? "Да" : "Нет"}</td>
                <td>{formatTimestamp(stream.updated_at)}</td>
                <td>
                  <div className="stream-actions">
                    <Link
                      className="icon-button"
                      href={`/watch?streamId=${stream.id}`}
                      aria-label={`Смотреть поток ${stream.name}`}
                      title="Смотреть"
                    >
                      <Eye size={16} aria-hidden />
                    </Link>
                    <IconButton
                      disabled={isViewer || busyStreamID === stream.id}
                      onClick={() => onRunCheck(stream)}
                      label={
                        busyStreamID === stream.id
                          ? "В очереди"
                          : `Запустить проверку: ${stream.name}`
                      }
                      tooltip="Запустить проверку"
                    >
                      <Play size={16} />
                    </IconButton>
                    <IconButton
                      disabled={isViewer}
                      onClick={() => onEdit(stream)}
                      label={`Редактировать поток ${stream.name}`}
                      tooltip="Редактировать"
                    >
                      <Pencil size={16} />
                    </IconButton>
                    <IconButton
                      disabled={isViewer || busyStreamID === stream.id}
                      onClick={() => onDelete(stream)}
                      label={`Удалить поток ${stream.name}`}
                      tooltip="Удалить"
                      destructive
                    >
                      <Trash2 size={16} />
                    </IconButton>
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </motion.div>
  );
}
