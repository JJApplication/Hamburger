import type { RequestSeriesPoint, StatResponse } from "../types/stat";

const HOUR_MS = 60 * 60 * 1000;
const ACTIVITY_HOURS = 7 * 24;

export type ActivityLevel = 0 | 1 | 2 | 3 | 4;

export interface ActivityCell {
  timestamp: string;
  requests: number;
  level: ActivityLevel;
}

function logScale(value: number, maximum: number): number {
  if (value <= 0 || maximum <= 0) return 0;
  return Math.log1p(value) / Math.log1p(maximum);
}

function levelForRequests(requests: number, maximum: number): ActivityLevel {
  if (requests <= 0 || maximum <= 0) return 0;
  return Math.max(1, Math.min(4, Math.ceil(logScale(requests, maximum) * 4))) as ActivityLevel;
}

function hourTimestamp(timestamp: string): number | null {
  const value = Date.parse(timestamp);
  return Number.isFinite(value) ? Math.floor(value / HOUR_MS) * HOUR_MS : null;
}

function requestTotal(point: RequestSeriesPoint): number {
  return Number.isFinite(point.total) ? Math.max(0, point.total) : 0;
}

export function makeHourlyActivityCells(stat?: StatResponse): ActivityCell[] {
  const points = stat?.series.requests ?? [];
  const buckets = new Map<number, number>();
  for (const point of points) {
    const bucket = hourTimestamp(point.timestamp);
    if (bucket === null) continue;
    buckets.set(bucket, (buckets.get(bucket) ?? 0) + requestTotal(point));
  }

  const lastPoint = points.at(-1);
  const lastBucket = lastPoint ? hourTimestamp(lastPoint.timestamp) : null;
  if (lastBucket === null) return [];

  const cells = Array.from({ length: ACTIVITY_HOURS }, (_, index) => {
    const bucket = lastBucket - (ACTIVITY_HOURS - 1 - index) * HOUR_MS;
    return {
      timestamp: new Date(bucket).toISOString(),
      requests: buckets.get(bucket) ?? 0,
      level: 0 as ActivityLevel,
    };
  });
  const maximum = Math.max(...cells.map((cell) => cell.requests), 0);
  return cells.map((cell) => ({ ...cell, level: levelForRequests(cell.requests, maximum) }));
}
