export type Range = "1h" | "5h" | "24h" | "7d" | "30d";

export interface Capabilities {
  system_cpu: boolean;
  system_memory: boolean;
  system_network: boolean;
  system_disk_io: boolean;
  process_cpu: boolean;
  process_memory: boolean;
  process_disk_io: boolean;
  runtime_gc: boolean;
  program_traffic: boolean;
  [key: string]: boolean;
}

export interface StatMeta {
  range: Range;
  start_time: string;
  end_time: string;
  bucket_seconds: number;
  generated_at: string;
  retention_days: number;
  capabilities: Capabilities;
}

export interface StatusSummary {
  "1xx": number;
  "2xx": number;
  "3xx": number;
  "4xx": number;
  "5xx": number;
}

export interface LatencySummary {
  avg_ms: number;
  p95_ms: number;
  max_ms: number;
}

export interface TrafficSummary {
  request_bytes: number;
  response_bytes: number;
  total_bytes: number;
}

export interface GCSummary {
  cycles: number;
  forced_cycles: number;
  pressure_percent: number;
  pause_total_ms: number;
  pause_avg_ms: number;
  pause_p95_ms: number;
  pause_max_ms: number;
}

export interface StatSummary {
  total_requests: number;
  frontend_requests: number;
  backend_requests: number;
  unknown_requests: number;
  error_requests: number;
  rps: number;
  error_rate: number;
  status: StatusSummary;
  latency: LatencySummary;
  gc: GCSummary;
  frontend_traffic: TrafficSummary;
  backend_traffic: TrafficSummary;
  total_traffic: TrafficSummary;
}

export interface RequestSeriesPoint {
  timestamp: string;
  frontend: number;
  backend: number;
  unknown: number;
  total: number;
  errors: number;
  status_1xx: number;
  status_2xx: number;
  status_3xx: number;
  status_4xx: number;
  status_5xx: number;
  rps: number;
  avg_latency_ms: number;
  p95_latency_ms: number;
  max_latency_ms: number;
}

export interface TrafficSeriesPoint {
  timestamp: string;
  frontend_request_bytes: number;
  frontend_response_bytes: number;
  backend_request_bytes: number;
  backend_response_bytes: number;
  request_bytes: number;
  response_bytes: number;
}

export interface GCSeriesPoint {
  timestamp: string;
  cycles: number;
  forced_cycles: number;
  pressure_percent: number;
  pause_total_ms: number;
  pause_avg_ms: number;
  pause_p95_ms: number;
  pause_max_ms: number;
}

export interface SystemSeriesPoint {
  timestamp: string;
  cpu_percent: number | null;
  cpu_peak_percent: number | null;
  memory_percent: number | null;
  memory_peak_percent: number | null;
  network_rx_bytes: number | null;
  network_tx_bytes: number | null;
  disk_read_bytes: number | null;
  disk_write_bytes: number | null;
}

export interface ProcessSeriesPoint {
  timestamp: string;
  cpu_percent: number | null;
  cpu_peak_percent: number | null;
  memory_bytes: number | null;
  memory_peak_bytes: number | null;
  memory_percent: number | null;
  memory_peak_percent: number | null;
  disk_read_bytes: number | null;
  disk_write_bytes: number | null;
}

export interface StatSeries {
  requests: RequestSeriesPoint[];
  traffic: TrafficSeriesPoint[];
  gc: GCSeriesPoint[];
  system: SystemSeriesPoint[];
  process: ProcessSeriesPoint[];
}

export interface DomainSummary {
  domain: string;
  requests: number;
  errors: number;
  request_bytes: number;
  response_bytes: number;
}

export interface DomainSeriesPoint {
  timestamp: string;
  requests: number;
  errors: number;
  request_bytes: number;
  response_bytes: number;
}

export interface ConnectionSnapshot {
  new: number;
  active: number;
  idle: number;
  hijacked: number;
  closed: number;
}

export interface StatResponse {
  total: number;
  api: number;
  static: number;
  fail: number;
  today: number;
  meta: StatMeta;
  summary: StatSummary;
  series: StatSeries;
  connections: {
    gateway: ConnectionSnapshot;
    front: ConnectionSnapshot;
    [key: string]: ConnectionSnapshot;
  };
  domains: DomainSummary[];
  domain_series?: DomainSeriesPoint[];
}

export type GeoData = Record<string, number>;
