export function formatNumber(value: number, maximumFractionDigits = 0): string {
  return new Intl.NumberFormat("zh-CN", { maximumFractionDigits }).format(Number.isFinite(value) ? value : 0);
}

export function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  return `${(value / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

export function formatRate(value: number): string {
  if (!Number.isFinite(value)) return "0";
  if (value >= 100) return formatNumber(value);
  return formatNumber(value, 2);
}

export function formatPercent(value: number, fractionDigits = 2): string {
  return `${formatNumber(value * 100, fractionDigits)}%`;
}

export function formatMilliseconds(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return "0 ms";
  if (value < 1) return `${(value * 1000).toFixed(0)} μs`;
  if (value < 1000) return `${value.toFixed(value < 10 ? 1 : 0)} ms`;
  return `${(value / 1000).toFixed(2)} s`;
}

export function formatLocalTime(value: string): string {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "—" : new Intl.DateTimeFormat("zh-CN", { hour: "2-digit", minute: "2-digit", second: "2-digit" }).format(date);
}

export function formatDateTime(value: string): string {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "—" : new Intl.DateTimeFormat("zh-CN", { dateStyle: "short", timeStyle: "short" }).format(date);
}
