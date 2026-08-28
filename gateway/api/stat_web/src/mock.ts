import type {
  DomainSeriesPoint,
  DomainSummary,
  GeoData,
  ProcessSeriesPoint,
  Range,
  RequestSeriesPoint,
  StatResponse,
  SystemSeriesPoint,
  TrafficSeriesPoint,
} from "./types/stat";

const rangeConfig: Record<Range, { durationMinutes: number; bucketMinutes: number }> = {
  "1h": { durationMinutes: 60, bucketMinutes: 1 },
  "5h": { durationMinutes: 300, bucketMinutes: 5 },
  "24h": { durationMinutes: 1_440, bucketMinutes: 15 },
  "7d": { durationMinutes: 10_080, bucketMinutes: 60 },
  "30d": { durationMinutes: 43_200, bucketMinutes: 120 },
};

const mockDomainSeed = [
  { domain: "api.example.com", share: 0.32, errors: 0.018 },
  { domain: "www.example.com", share: 0.27, errors: 0.011 },
  { domain: "cdn.example.com", share: 0.18, errors: 0.007 },
  { domain: "console.example.com", share: 0.13, errors: 0.026 },
  { domain: "status.example.com", share: 0.06, errors: 0.003 },
  { domain: "images.example.com", share: 0.04, errors: 0.014 },
];

export const MOCK_GEO: GeoData = {
  CN: 10_000,
  US: 6_400,
  JP: 3_100,
  SG: 2_400,
  DE: 1_850,
  GB: 1_600,
  AU: 1_280,
  CA: 920,
  FR: 810,
  IN: 760,
  BR: 640,
  KR: 590,
};

function sum(values: number[]): number {
  return values.reduce((total, value) => total + value, 0);
}

function makeRequestSeries(range: Range): RequestSeriesPoint[] {
  const { durationMinutes, bucketMinutes } = rangeConfig[range];
  const count = Math.floor(durationMinutes / bucketMinutes) + 1;
  const end = Date.now();
  const start = end - durationMinutes * 60_000;
  return Array.from({ length: count }, (_, index) => {
    const wave = Math.sin(index / 4.5) * 0.16 + Math.cos(index / 10) * 0.08;
    const frontend = Math.max(0, Math.round(108 * (1 + wave + index / count * 0.12)));
    const backend = Math.max(0, Math.round(72 * (1 + wave * 0.7 + Math.sin(index / 7) * 0.08)));
    const errors = Math.max(0, Math.round((frontend + backend) * (0.012 + (index % 17 === 0 ? 0.025 : 0))));
    const total = frontend + backend;
    const status3xx = Math.round(total * 0.012);
    const status4xx = Math.round(errors * 0.72);
    const timestamp = new Date(start + index * bucketMinutes * 60_000).toISOString();
    const avgLatency = 22 + Math.sin(index / 6) * 4;
    const p95Latency = avgLatency * 2.8;
    return {
      timestamp,
      frontend,
      backend,
      unknown: 0,
      total,
      errors,
      status_1xx: 0,
      status_2xx: Math.max(0, total - status3xx - errors),
      status_3xx: status3xx,
      status_4xx: status4xx,
      status_5xx: errors - status4xx,
      rps: total / (bucketMinutes * 60),
      avg_latency_ms: avgLatency,
      p95_latency_ms: p95Latency,
      max_latency_ms: p95Latency * 2.1,
    };
  });
}

function makeTrafficSeries(requests: RequestSeriesPoint[]): TrafficSeriesPoint[] {
  return requests.map((point, index) => {
    const frontendRequestBytes = point.frontend * (320 + (index % 4) * 24);
    const frontendResponseBytes = point.frontend * (1_840 + (index % 5) * 90);
    const backendRequestBytes = point.backend * (640 + (index % 3) * 45);
    const backendResponseBytes = point.backend * (2_480 + (index % 6) * 110);
    return {
      timestamp: point.timestamp,
      frontend_request_bytes: frontendRequestBytes,
      frontend_response_bytes: frontendResponseBytes,
      backend_request_bytes: backendRequestBytes,
      backend_response_bytes: backendResponseBytes,
      request_bytes: frontendRequestBytes + backendRequestBytes,
      response_bytes: frontendResponseBytes + backendResponseBytes,
    };
  });
}

function makeSystemSeries(requests: RequestSeriesPoint[]): SystemSeriesPoint[] {
  return requests.map((point, index) => ({
    timestamp: point.timestamp,
    cpu_percent: 24 + Math.sin(index / 7) * 8 + point.total / 90,
    cpu_peak_percent: 38 + Math.sin(index / 7) * 10 + point.total / 70,
    memory_percent: 48 + Math.cos(index / 13) * 2,
    memory_peak_percent: 51 + Math.cos(index / 13) * 2,
    network_rx_bytes: point.total * 180,
    network_tx_bytes: point.total * 260,
    disk_read_bytes: point.total * 24,
    disk_write_bytes: point.total * 18,
  }));
}

function makeProcessSeries(requests: RequestSeriesPoint[]): ProcessSeriesPoint[] {
  return requests.map((point, index) => ({
    timestamp: point.timestamp,
    cpu_percent: 8 + Math.sin(index / 5) * 3 + point.backend / 40,
    cpu_peak_percent: 15 + Math.sin(index / 5) * 4 + point.backend / 32,
    memory_bytes: 184_000_000 + Math.round(Math.cos(index / 9) * 8_000_000),
    memory_peak_bytes: 196_000_000 + Math.round(Math.cos(index / 9) * 9_000_000),
    memory_percent: 2.8 + Math.cos(index / 9) * 0.15,
    memory_peak_percent: 3.1 + Math.cos(index / 9) * 0.16,
    disk_read_bytes: point.backend * 11,
    disk_write_bytes: point.backend * 8,
  }));
}

function makeDomains(totalRequests: number): DomainSummary[] {
  return mockDomainSeed.map(({ domain, share, errors }) => {
    const requests = Math.round(totalRequests * share);
    return {
      domain,
      requests,
      errors: Math.round(requests * errors),
      request_bytes: requests * 420,
      response_bytes: requests * 2_260,
    };
  });
}

function makeDomainSeries(domain: string, requests: RequestSeriesPoint[]): DomainSeriesPoint[] {
  const seed = mockDomainSeed.find((item) => item.domain === domain);
  if (!seed) return [];
  return requests.map((point) => {
    const domainRequests = Math.round(point.total * seed.share);
    return {
      timestamp: point.timestamp,
      requests: domainRequests,
      errors: Math.round(domainRequests * seed.errors),
      request_bytes: domainRequests * 420,
      response_bytes: domainRequests * 2_260,
    };
  });
}

export function createMockStat(range: Range, domain = ""): StatResponse {
  const requests = makeRequestSeries(range);
  const traffic = makeTrafficSeries(requests);
  const system = makeSystemSeries(requests);
  const process = makeProcessSeries(requests);
  const totalRequests = sum(requests.map((point) => point.total));
  const frontendRequests = sum(requests.map((point) => point.frontend));
  const backendRequests = sum(requests.map((point) => point.backend));
  const errors = sum(requests.map((point) => point.errors));
  const requestBytes = sum(traffic.map((point) => point.request_bytes));
  const responseBytes = sum(traffic.map((point) => point.response_bytes));
  const startTime = requests[0]?.timestamp ?? new Date().toISOString();
  const endTime = requests.at(-1)?.timestamp ?? new Date().toISOString();
  const bucketSeconds = rangeConfig[range].bucketMinutes * 60;
  const domainSeries = domain ? makeDomainSeries(domain, requests) : undefined;

  return {
    total: totalRequests + 120_000,
    api: backendRequests + 68_000,
    static: frontendRequests + 52_000,
    fail: errors + 2_100,
    today: totalRequests,
    meta: {
      range,
      start_time: startTime,
      end_time: endTime,
      bucket_seconds: bucketSeconds,
      generated_at: new Date().toISOString(),
      retention_days: 30,
      capabilities: {
        system_cpu: true,
        system_memory: true,
        system_network: true,
        system_disk_io: true,
        process_cpu: true,
        process_memory: true,
        process_disk_io: true,
        program_traffic: true,
      },
    },
    summary: {
      total_requests: totalRequests,
      frontend_requests: frontendRequests,
      backend_requests: backendRequests,
      unknown_requests: 0,
      error_requests: errors,
      rps: totalRequests / (rangeConfig[range].durationMinutes * 60),
      error_rate: errors / Math.max(totalRequests, 1),
      status: {
        "1xx": sum(requests.map((point) => point.status_1xx)),
        "2xx": sum(requests.map((point) => point.status_2xx)),
        "3xx": sum(requests.map((point) => point.status_3xx)),
        "4xx": sum(requests.map((point) => point.status_4xx)),
        "5xx": sum(requests.map((point) => point.status_5xx)),
      },
      latency: { avg_ms: 22, p95_ms: 62, max_ms: 132 },
      frontend_traffic: {
        request_bytes: sum(traffic.map((point) => point.frontend_request_bytes)),
        response_bytes: sum(traffic.map((point) => point.frontend_response_bytes)),
        total_bytes: sum(traffic.map((point) => point.frontend_request_bytes + point.frontend_response_bytes)),
      },
      backend_traffic: {
        request_bytes: sum(traffic.map((point) => point.backend_request_bytes)),
        response_bytes: sum(traffic.map((point) => point.backend_response_bytes)),
        total_bytes: sum(traffic.map((point) => point.backend_request_bytes + point.backend_response_bytes)),
      },
      total_traffic: { request_bytes: requestBytes, response_bytes: responseBytes, total_bytes: requestBytes + responseBytes },
    },
    series: { requests, traffic, system, process },
    connections: {
      gateway: { new: 128, active: 42, idle: 86, hijacked: 3, closed: 2_418 },
      front: { new: 96, active: 31, idle: 65, hijacked: 1, closed: 1_932 },
    },
    domains: makeDomains(totalRequests),
    ...(domainSeries ? { domain_series: domainSeries } : {}),
  };
}
