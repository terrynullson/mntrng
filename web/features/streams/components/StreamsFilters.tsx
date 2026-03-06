"use client";

import type { Project } from "@/lib/api/types";
import type { IsActiveFilter, StatusFilter } from "../types";

type StreamsFiltersProps = {
  search: string;
  onSearchChange: (value: string) => void;
  projectId: string;
  onProjectIdChange: (value: string) => void;
  isActiveFilter: IsActiveFilter;
  onIsActiveFilterChange: (value: IsActiveFilter) => void;
  statusFilter: StatusFilter;
  onStatusFilterChange: (value: StatusFilter) => void;
  projects: Project[];
  disabled?: boolean;
};

export function StreamsFilters({
  search,
  onSearchChange,
  projectId,
  onProjectIdChange,
  isActiveFilter,
  onIsActiveFilterChange,
  statusFilter,
  onStatusFilterChange,
  projects,
  disabled = false
}: StreamsFiltersProps) {
  return (
    <div className="premium-filters">
      <label className="form-field" htmlFor="streams-search">
        <span>Поиск</span>
        <input
          id="streams-search"
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Название, ID потока или проекта"
          disabled={disabled}
        />
      </label>
      <label className="form-field" htmlFor="streams-project-filter">
        <span>Проект</span>
        <select
          id="streams-project-filter"
          value={projectId}
          onChange={(e) => onProjectIdChange(e.target.value)}
          disabled={disabled}
        >
          <option value="">Все проекты</option>
          {projects.map((project) => (
            <option key={project.id} value={project.id}>
              {project.name} ({project.id})
            </option>
          ))}
        </select>
      </label>
      <label className="form-field" htmlFor="streams-active-filter">
        <span>Активен</span>
        <select
          id="streams-active-filter"
          value={isActiveFilter}
          onChange={(e) => onIsActiveFilterChange(e.target.value as IsActiveFilter)}
          disabled={disabled}
        >
          <option value="all">Все</option>
          <option value="true">Активные</option>
          <option value="false">Неактивные</option>
        </select>
      </label>
      <label className="form-field" htmlFor="streams-status-filter">
        <span>Статус</span>
        <select
          id="streams-status-filter"
          value={statusFilter}
          onChange={(e) => onStatusFilterChange(e.target.value as StatusFilter)}
          disabled={disabled}
        >
          <option value="all">Все</option>
          <option value="OK">OK</option>
          <option value="WARN">WARN</option>
          <option value="FAIL">FAIL</option>
        </select>
      </label>
    </div>
  );
}
