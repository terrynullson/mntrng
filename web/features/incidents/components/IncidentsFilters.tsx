"use client";

import type { SeverityFilter, StatusFilter } from "../types";

type IncidentsFiltersProps = {
  search: string;
  onSearchChange: (value: string) => void;
  statusFilter: StatusFilter;
  onStatusFilterChange: (value: StatusFilter) => void;
  severityFilter: SeverityFilter;
  onSeverityFilterChange: (value: SeverityFilter) => void;
  disabled?: boolean;
};

export function IncidentsFilters({
  search,
  onSearchChange,
  statusFilter,
  onStatusFilterChange,
  severityFilter,
  onSeverityFilterChange,
  disabled = false
}: IncidentsFiltersProps) {
  return (
    <div className="premium-filters">
      <label className="form-field" htmlFor="incidents-search">
        <span>Поиск</span>
        <input
          id="incidents-search"
          type="search"
          value={search}
          onChange={(e: { target: { value: string } }) => onSearchChange(e.target.value)}
          placeholder="Название потока или причина"
          disabled={disabled}
          aria-label="Поиск по названию потока или причине"
        />
      </label>
      <label className="form-field" htmlFor="incidents-status">
        <span>Статус</span>
        <select
          id="incidents-status"
          value={statusFilter}
          onChange={(e: { target: { value: string } }) =>
            onStatusFilterChange(e.target.value as StatusFilter)
          }
          disabled={disabled}
          aria-label="Фильтр по статусу инцидента"
        >
          <option value="">Все</option>
          <option value="open">Открыт</option>
          <option value="resolved">Закрыт</option>
        </select>
      </label>
      <label className="form-field" htmlFor="incidents-severity">
        <span>Серьёзность</span>
        <select
          id="incidents-severity"
          value={severityFilter}
          onChange={(e: { target: { value: string } }) =>
            onSeverityFilterChange(e.target.value as SeverityFilter)
          }
          disabled={disabled}
          aria-label="Фильтр по серьёзности"
        >
          <option value="">Все</option>
          <option value="warn">WARN</option>
          <option value="fail">FAIL</option>
        </select>
      </label>
    </div>
  );
}
