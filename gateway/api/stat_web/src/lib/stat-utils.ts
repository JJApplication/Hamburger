import type {
	Capabilities,
	DomainSummary,
	GeoData,
	GCSeriesPoint,
	ProcessSeriesPoint,
	Range,
	RequestSeriesPoint,
	StatResponse,
	SystemSeriesPoint,
	TrafficSeriesPoint,
} from "../types/stat";

export const RANGES: Range[] = ["1h", "5h", "24h", "7d", "30d"];

const emptyCapabilities: Capabilities = {
  system_cpu: false,
  system_memory: false,
  system_network: false,
  system_disk_io: false,
	process_cpu: false,
	process_memory: false,
	process_disk_io: false,
	runtime_gc: false,
	program_traffic: false,
};

function finiteNumber(value: unknown, fallback = 0): number {
  return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}

function nullableNumber(value: unknown): number | null {
  return value === null ? null : finiteNumber(value);
}

function arrayOf<T>(value: unknown): T[] {
  return Array.isArray(value) ? value as T[] : [];
}

export function normalizeStatResponse(input: unknown): StatResponse {
  const value = (input && typeof input === "object" ? input : {}) as Partial<StatResponse>;
  const meta = value.meta ?? {} as StatResponse["meta"];
  const summary = value.summary ?? {} as StatResponse["summary"];
  const status = summary.status ?? {} as StatResponse["summary"]["status"];
  const latency = summary.latency ?? {} as StatResponse["summary"]["latency"];
  const gc = summary.gc ?? {} as StatResponse["summary"]["gc"];
  const traffic = summary.total_traffic ?? {} as StatResponse["summary"]["total_traffic"];
  const series = value.series ?? {} as StatResponse["series"];
  const connections = value.connections ?? {} as StatResponse["connections"];

  return {
    total: finiteNumber(value.total),
    api: finiteNumber(value.api),
    static: finiteNumber(value.static),
    fail: finiteNumber(value.fail),
    today: finiteNumber(value.today),
    meta: {
      range: RANGES.includes(meta.range as Range) ? meta.range as Range : "1h",
      start_time: typeof meta.start_time === "string" ? meta.start_time : "",
      end_time: typeof meta.end_time === "string" ? meta.end_time : "",
      bucket_seconds: finiteNumber(meta.bucket_seconds, 60),
      generated_at: typeof meta.generated_at === "string" ? meta.generated_at : "",
      retention_days: finiteNumber(meta.retention_days, 30),
      capabilities: { ...emptyCapabilities, ...(meta.capabilities ?? {}) },
    },
    summary: {
      total_requests: finiteNumber(summary.total_requests),
      frontend_requests: finiteNumber(summary.frontend_requests),
      backend_requests: finiteNumber(summary.backend_requests),
      unknown_requests: finiteNumber(summary.unknown_requests),
      error_requests: finiteNumber(summary.error_requests),
      rps: finiteNumber(summary.rps),
      error_rate: finiteNumber(summary.error_rate),
      status: {
        "1xx": finiteNumber(status["1xx"]),
        "2xx": finiteNumber(status["2xx"]),
        "3xx": finiteNumber(status["3xx"]),
        "4xx": finiteNumber(status["4xx"]),
        "5xx": finiteNumber(status["5xx"]),
      },
      latency: {
        avg_ms: finiteNumber(latency.avg_ms),
        p95_ms: finiteNumber(latency.p95_ms),
        max_ms: finiteNumber(latency.max_ms),
      },
      gc: normalizeGC(gc),
      frontend_traffic: normalizeTraffic(summary.frontend_traffic),
      backend_traffic: normalizeTraffic(summary.backend_traffic),
      total_traffic: normalizeTraffic(traffic),
	    },
	    series: {
	      requests: arrayOf<RequestSeriesPoint>(series.requests),
	      traffic: arrayOf<TrafficSeriesPoint>(series.traffic),
	      gc: arrayOf<GCSeriesPoint>(series.gc).map((point) => ({
        timestamp: typeof point.timestamp === "string" ? point.timestamp : "",
        cycles: finiteNumber(point.cycles),
        forced_cycles: finiteNumber(point.forced_cycles),
        pressure_percent: finiteNumber(point.pressure_percent),
        pause_total_ms: finiteNumber(point.pause_total_ms),
        pause_avg_ms: finiteNumber(point.pause_avg_ms),
        pause_p95_ms: finiteNumber(point.pause_p95_ms),
        pause_max_ms: finiteNumber(point.pause_max_ms),
      })),
	      system: arrayOf<SystemSeriesPoint>(series.system).map((point) => ({
        ...point,
        cpu_percent: nullableNumber(point.cpu_percent),
        cpu_peak_percent: nullableNumber(point.cpu_peak_percent),
        memory_percent: nullableNumber(point.memory_percent),
        memory_peak_percent: nullableNumber(point.memory_peak_percent),
        network_rx_bytes: nullableNumber(point.network_rx_bytes),
        network_tx_bytes: nullableNumber(point.network_tx_bytes),
        disk_read_bytes: nullableNumber(point.disk_read_bytes),
        disk_write_bytes: nullableNumber(point.disk_write_bytes),
      })),
	      process: arrayOf<ProcessSeriesPoint>(series.process).map((point) => ({
        ...point,
        cpu_percent: nullableNumber(point.cpu_percent),
        cpu_peak_percent: nullableNumber(point.cpu_peak_percent),
        memory_bytes: nullableNumber(point.memory_bytes),
        memory_peak_bytes: nullableNumber(point.memory_peak_bytes),
        memory_percent: nullableNumber(point.memory_percent),
        memory_peak_percent: nullableNumber(point.memory_peak_percent),
        disk_read_bytes: nullableNumber(point.disk_read_bytes),
        disk_write_bytes: nullableNumber(point.disk_write_bytes),
      })),
    },
    connections: {
      gateway: normalizeConnection(connections.gateway),
      front: normalizeConnection(connections.front),
    },
    domains: arrayOf<DomainSummary>(value.domains).map((domain) => ({
      domain: typeof domain.domain === "string" ? domain.domain : "未知域名",
      requests: finiteNumber(domain.requests),
      errors: finiteNumber(domain.errors),
      request_bytes: finiteNumber(domain.request_bytes),
      response_bytes: finiteNumber(domain.response_bytes),
    })),
    domain_series: Array.isArray(value.domain_series) ? value.domain_series : undefined,
  };
}

function normalizeTraffic(value: unknown) {
  const traffic = (value && typeof value === "object" ? value : {}) as Record<string, unknown>;
  return {
    request_bytes: finiteNumber(traffic.request_bytes),
    response_bytes: finiteNumber(traffic.response_bytes),
    total_bytes: finiteNumber(traffic.total_bytes),
  };
}

function normalizeGC(value: unknown) {
  const gc = (value && typeof value === "object" ? value : {}) as Record<string, unknown>;
  return {
    cycles: finiteNumber(gc.cycles),
    forced_cycles: finiteNumber(gc.forced_cycles),
    pressure_percent: finiteNumber(gc.pressure_percent),
    pause_total_ms: finiteNumber(gc.pause_total_ms),
    pause_avg_ms: finiteNumber(gc.pause_avg_ms),
    pause_p95_ms: finiteNumber(gc.pause_p95_ms),
    pause_max_ms: finiteNumber(gc.pause_max_ms),
  };
}

function normalizeConnection(value: unknown) {
  const connection = (value && typeof value === "object" ? value : {}) as Record<string, unknown>;
  return {
    new: finiteNumber(connection.new),
    active: finiteNumber(connection.active),
    idle: finiteNumber(connection.idle),
    hijacked: finiteNumber(connection.hijacked),
    closed: finiteNumber(connection.closed),
  };
}

export function normalizeGeoData(value: unknown): GeoData {
  if (!value || typeof value !== "object" || Array.isArray(value)) return {};
  const result: GeoData = {};
  for (const [code, count] of Object.entries(value)) {
    const normalizedCode = code.trim().toUpperCase();
    const normalizedCount = finiteNumber(count);
    if (normalizedCode && normalizedCount >= 0) result[normalizedCode] = normalizedCount;
  }
  return result;
}

export function filterDomains(domains: DomainSummary[], query: string): DomainSummary[] {
  const normalized = query.trim().toLowerCase();
  if (!normalized) return domains;
  return domains.filter((domain) => domain.domain.toLowerCase().includes(normalized));
}

export function topDomains(domains: DomainSummary[], limit = 10): DomainSummary[] {
  return [...domains].sort((left, right) => right.requests - left.requests || left.domain.localeCompare(right.domain)).slice(0, limit);
}

export function logScale(value: number, maximum: number): number {
  if (value <= 0 || maximum <= 0) return 0;
  return Math.log1p(value) / Math.log1p(maximum);
}

export function totalGeoRequests(geo: GeoData): number {
  return Object.values(geo).reduce((total, count) => total + count, 0);
}

const countryNames: Record<string, string> = {
  AE: "阿联酋", AR: "阿根廷", AU: "澳大利亚", BR: "巴西", CA: "加拿大", CN: "中国",
  DE: "德国", EG: "埃及", ES: "西班牙", FR: "法国", GB: "英国", ID: "印度尼西亚",
  IN: "印度", IT: "意大利", JP: "日本", KR: "韩国", MX: "墨西哥", NG: "尼日利亚",
  NL: "荷兰", NO: "挪威", RU: "俄罗斯", SA: "沙特阿拉伯", SG: "新加坡", SE: "瑞典",
  TR: "土耳其", US: "美国", ZA: "南非", NZ: "新西兰",
};

export function countryName(code: string): string {
  const normalized = code.trim().toUpperCase();
  return countryNames[normalized] ?? `国家 ${normalized || "未知"}`;
}
