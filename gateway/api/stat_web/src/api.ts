import { normalizeGeoData, normalizeStatResponse } from "./lib/stat-utils";
import { createMockStat, MOCK_GEO } from "./mock";
import type { GeoData, Range, StatResponse } from "./types/stat";

const configuredBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? "").trim();
export const API_BASE_URL = configuredBaseUrl.replace(/\/+$/, "");
export const USE_MOCK = import.meta.env.MODE !== "test" && (import.meta.env.VITE_USE_MOCK ?? "").trim().toLowerCase() === "true";

export class ApiError extends Error {
  readonly status: number;

  constructor(message: string, status = 0) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

function apiUrl(path: string, query?: URLSearchParams): string {
  const url = `${API_BASE_URL}${path}`;
  return query && query.size > 0 ? `${url}?${query.toString()}` : url;
}

async function getJSON<T>(path: string, query?: URLSearchParams, signal?: AbortSignal): Promise<T> {
  let response: Response;
  try {
    response = await fetch(apiUrl(path, query), {
      headers: { Accept: "application/json" },
      signal,
    });
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      throw error;
    }
    throw new ApiError("无法连接统计服务");
  }
  if (!response.ok) {
    let message = `统计服务返回 ${response.status}`;
    try {
      const body = (await response.json()) as { message?: string };
      if (body.message) message = body.message;
    } catch {
      // Keep the status-based message when an error response is not JSON.
    }
    throw new ApiError(message, response.status);
  }
  return (await response.json()) as T;
}

export async function fetchStat(range: Range, domain?: string, signal?: AbortSignal): Promise<StatResponse> {
  if (USE_MOCK) {
    if (signal?.aborted) throw new DOMException("The operation was aborted", "AbortError");
    return createMockStat(range, domain);
  }
  const query = new URLSearchParams({ range });
  if (domain?.trim()) query.set("domain", domain.trim());
  return normalizeStatResponse(await getJSON<unknown>("/api/stat", query, signal));
}

export async function fetchGeo(signal?: AbortSignal): Promise<GeoData> {
  if (USE_MOCK) {
    if (signal?.aborted) throw new DOMException("The operation was aborted", "AbortError");
    return { ...MOCK_GEO };
  }
  return normalizeGeoData(await getJSON<unknown>("/api/geo", undefined, signal));
}
