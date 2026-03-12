import type { GatewayData } from "@/types/gateway";

const mockGatewayData: GatewayData = {
  overview: {
    totalConnections: 1284,
    activeDomains: 6,
    warningDomains: 2,
    totalMappings: 11,
    criticalAlerts: 1,
    latestChange: "2026-03-11 09:35:22",
    trend: [
      { timestamp: "09:00", totalConnections: 952, activeDomains: 4 },
      { timestamp: "09:10", totalConnections: 1018, activeDomains: 4 },
      { timestamp: "09:20", totalConnections: 1142, activeDomains: 5 },
      { timestamp: "09:30", totalConnections: 1210, activeDomains: 6 },
      { timestamp: "09:40", totalConnections: 1284, activeDomains: 6 },
      { timestamp: "09:50", totalConnections: 1240, activeDomains: 6 },
    ],
  },
  domains: [
    {
      domain: "api.hamburger.local",
      status: "online",
      currentConnections: 302,
      peakConnections: 420,
      lastHeartbeat: "3 秒前",
      mappings: [
        { frontendPort: 443, backendTarget: "10.10.1.21:8080", protocol: "https" },
        { frontendPort: 8443, backendTarget: "10.10.1.22:8080", protocol: "https" },
      ],
    },
    {
      domain: "gateway.hamburger.local",
      status: "online",
      currentConnections: 460,
      peakConnections: 615,
      lastHeartbeat: "2 秒前",
      mappings: [{ frontendPort: 443, backendTarget: "10.10.2.11:9000", protocol: "https" }],
    },
    {
      domain: "edge.hamburger.local",
      status: "warning",
      currentConnections: 177,
      peakConnections: 300,
      lastHeartbeat: "16 秒前",
      mappings: [
        { frontendPort: 80, backendTarget: "10.10.3.8:8081", protocol: "http" },
        { frontendPort: 443, backendTarget: "10.10.3.8:8443", protocol: "https" },
      ],
    },
    {
      domain: "stream.hamburger.local",
      status: "warning",
      currentConnections: 98,
      peakConnections: 201,
      lastHeartbeat: "45 秒前",
      mappings: [{ frontendPort: 10443, backendTarget: "10.10.4.3:18080", protocol: "tcp" }],
    },
    {
      domain: "static.hamburger.local",
      status: "online",
      currentConnections: 166,
      peakConnections: 240,
      lastHeartbeat: "5 秒前",
      mappings: [
        { frontendPort: 80, backendTarget: "10.10.5.5:8080", protocol: "http" },
        { frontendPort: 443, backendTarget: "10.10.5.5:8443", protocol: "https" },
      ],
    },
    {
      domain: "legacy.hamburger.local",
      status: "offline",
      currentConnections: 0,
      peakConnections: 40,
      lastHeartbeat: "8 分钟前",
      mappings: [{ frontendPort: 10080, backendTarget: "10.10.6.6:7000", protocol: "tcp" }],
    },
  ],
  configs: {
    core: {
      mode: "gateway",
      listener: "0.0.0.0:443",
      logLevel: "info",
      maxConnections: 50000,
      readTimeoutMs: 5000,
      writeTimeoutMs: 5000,
    },
    frontendProxy: {
      upstreamScheme: "https",
      gzip: true,
      websocket: true,
      cors: true,
      errorPage: "/assets/500.html",
    },
    backendServices: [
      {
        name: "user-service",
        target: "10.10.1.21:8080",
        healthCheckPath: "/health",
        healthStatus: "healthy",
        weight: 100,
      },
      {
        name: "order-service",
        target: "10.10.1.22:8080",
        healthCheckPath: "/actuator/health",
        healthStatus: "degraded",
        weight: 90,
      },
      {
        name: "media-service",
        target: "10.10.4.3:18080",
        healthCheckPath: "/healthz",
        healthStatus: "healthy",
        weight: 80,
      },
    ],
  },
  experiments: [
    {
      key: "anytls",
      name: "AnyTLS",
      enabled: true,
      riskLevel: "medium",
      description: "启用 AnyTLS 入站实验能力，支持加密流量封装与转发。",
      params: {
        listen: "0.0.0.0:7443",
        handshakeTimeoutMs: 8000,
        certSource: "memory",
      },
    },
    {
      key: "trojan",
      name: "Trojan",
      enabled: false,
      riskLevel: "high",
      description: "启用 Trojan 隧道实验能力，建议在灰度环境验证后再全量开启。",
      params: {
        listen: "0.0.0.0:7444",
        allowInsecure: false,
        fallbackTarget: "10.10.9.9:8443",
      },
    },
  ],
};

const wait = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

export async function fetchGatewayData(): Promise<GatewayData> {
  await wait(280);
  return structuredClone(mockGatewayData);
}
