export type DomainHealthStatus = "online" | "offline" | "warning";

export interface ConnectionTrendPoint {
  timestamp: string;
  totalConnections: number;
  activeDomains: number;
}

export interface DomainMapping {
  frontendPort: number;
  backendTarget: string;
  protocol: "http" | "https" | "tcp";
}

export interface DomainConnection {
  domain: string;
  status: DomainHealthStatus;
  currentConnections: number;
  peakConnections: number;
  lastHeartbeat: string;
  mappings: DomainMapping[];
}

export interface GatewayOverview {
  totalConnections: number;
  activeDomains: number;
  warningDomains: number;
  totalMappings: number;
  criticalAlerts: number;
  latestChange: string;
  trend: ConnectionTrendPoint[];
}

export interface CoreGatewayConfig {
  mode: string;
  listener: string;
  logLevel: string;
  maxConnections: number;
  readTimeoutMs: number;
  writeTimeoutMs: number;
}

export interface FrontendProxyConfig {
  upstreamScheme: string;
  gzip: boolean;
  websocket: boolean;
  cors: boolean;
  errorPage: string;
}

export interface BackendServiceConfig {
  name: string;
  target: string;
  healthCheckPath: string;
  healthStatus: "healthy" | "degraded" | "down";
  weight: number;
}

export interface GatewayConfigs {
  core: CoreGatewayConfig;
  frontendProxy: FrontendProxyConfig;
  backendServices: BackendServiceConfig[];
}

export interface ExperimentFeature {
  key: "anytls" | "trojan";
  name: string;
  enabled: boolean;
  riskLevel: "low" | "medium" | "high";
  description: string;
  params: Record<string, string | number | boolean>;
}

export interface GatewayData {
  overview: GatewayOverview;
  domains: DomainConnection[];
  configs: GatewayConfigs;
  experiments: ExperimentFeature[];
}
