export function formatTimestamp(timestamp: string | null): string {
  if (!timestamp) return "-";
  const parsed = new Date(timestamp);
  return Number.isNaN(parsed.getTime()) ? timestamp : parsed.toLocaleString();
}
