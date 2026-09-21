import type { ConnectionSnapshot, DomainConnection, GatewayData, ManagementConfig, StatResponse } from "@/types/gateway";

const API_BASE = (process.env.NEXT_PUBLIC_GATEWAY_API_BASE_URL ?? "").replace(/\/+$/, "");
const AUTH_HEADER = process.env.NEXT_PUBLIC_GATEWAY_API_AUTH_HEADER ?? "Authorization";
const AUTH_PREFIX = process.env.NEXT_PUBLIC_GATEWAY_API_AUTH_PREFIX ?? "Bearer";

export interface AuthUser { username: string; nickname: string; avatar?: string }
interface LoginResponse { token: string; user: AuthUser }

function apiUrl(path: string): string { return `${API_BASE}${path}` }

async function request<T>(path: string, init: RequestInit = {}, token?: string, signal?: AbortSignal): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  if (init.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  if (token) headers.set(AUTH_HEADER, AUTH_PREFIX.trim() ? `${AUTH_PREFIX.trim()} ${token}` : token);
  let response: Response;
  try { response = await fetch(apiUrl(path), { ...init, headers, cache: "no-store", signal }); }
  catch { throw new Error("无法连接网关 API"); }
  if (!response.ok) {
    if (response.status === 401 && typeof window !== "undefined") window.dispatchEvent(new Event("hamburger-auth-expired"));
    let message = `网关 API 返回 ${response.status}`;
    try { const body = await response.json() as { message?: string }; if (body.message) message = body.message; } catch { /* status is enough */ }
    const error = new Error(message); (error as Error & { status?: number }).status = response.status; throw error;
  }
  return response.json() as Promise<T>;
}

export async function login(username: string, password: string): Promise<LoginResponse> {
  return request<LoginResponse>("/api/login", { method: "POST", body: JSON.stringify({ username, password }) });
}
export async function fetchCurrentUser(token: string): Promise<AuthUser> { return request<AuthUser>("/api/user", {}, token); }
export async function logout(token: string): Promise<void> { await request("/api/logout", { method: "POST" }, token); }
export async function fetchStat(range = "1h", domain?: string, token?: string, signal?: AbortSignal): Promise<StatResponse> {
  const query = new URLSearchParams({ range }); if (domain) query.set("domain", domain);
  return request<StatResponse>(`/api/stat?${query.toString()}`, {}, token, signal);
}
export async function fetchGeo(token?: string, signal?: AbortSignal): Promise<Record<string, number>> { return request<Record<string, number>>("/api/geo", {}, token, signal); }
export async function fetchManagementDomains(token: string): Promise<{ domains: DomainConnection[] }> {
  const response = await request<{ domains?: Array<Record<string, unknown>> }>("/api/management/domains", {}, token);
  const domains = (response.domains ?? []).map((value) => {
    const mappings = Array.isArray(value.mappings) ? value.mappings : [];
    return {
      domain: String(value.domain ?? ""),
      service: typeof value.service === "string" ? value.service : undefined,
      status: (value.status === "online" || value.status === "offline" || value.status === "warning" ? value.status : "unknown") as DomainConnection["status"],
      currentConnections: Number(value.current_connections ?? value.currentConnections ?? 0),
      peakConnections: Number(value.peak_connections ?? value.peakConnections ?? 0),
      lastHeartbeat: String(value.last_probe ?? value.lastHeartbeat ?? ""),
      mappings: mappings.map((mapping) => {
        const item = (mapping ?? {}) as Record<string, unknown>;
        return { frontendPort: Number(item.frontend_port ?? item.frontendPort ?? 0), backendTarget: String(item.backend_target ?? item.backendTarget ?? ""), protocol: String(item.protocol ?? "") };
      }),
    };
  });
  return { domains };
}
export async function setDomainState(domain: string, state: "start" | "stop", token: string): Promise<void> { await request("/api/management/domains/state", { method: "POST", body: JSON.stringify({ domain, state }) }, token); }
export async function fetchManagementConfig(token: string): Promise<ManagementConfig> { return request("/api/management/config", {}, token); }
export async function saveManagementConfig(payload: { version: string; values: Record<string, unknown> }, token: string): Promise<ManagementConfig> { return request("/api/management/config", { method: "PUT", body: JSON.stringify(payload) }, token); }
export async function applyManagementConfig(services: string[], token: string): Promise<{ operation_id: string }> { return request("/api/management/config/apply", { method: "POST", body: JSON.stringify({ services }) }, token); }
export async function fetchOperation(id: string, token: string): Promise<{ status: string; error?: string }> { return request(`/api/management/operations/${encodeURIComponent(id)}`, {}, token); }

function snapshot(value: ConnectionSnapshot | undefined): ConnectionSnapshot { return value ?? { new: 0, active: 0, idle: 0, hijacked: 0, closed: 0 }; }

export async function fetchGatewayData(token: string): Promise<GatewayData> {
  const [stat, domainsResult, connections, geo, management] = await Promise.all([
    fetchStat("1h", undefined, token),
    fetchManagementDomains(token),
    request<{ gateway: ConnectionSnapshot; front: ConnectionSnapshot }>("/api/conn", {}, token),
    fetchGeo(token),
    fetchManagementConfig(token),
  ]);
  const domains = domainsResult.domains ?? [];
  const gatewayConnections = snapshot(connections.gateway);
  const overview = {
    totalConnections: gatewayConnections.active + gatewayConnections.idle,
    activeDomains: domains.filter((item) => item.status === "online").length,
    warningDomains: domains.filter((item) => item.status === "warning").length,
    totalMappings: domains.reduce((sum, item) => sum + item.mappings.length, 0),
    criticalAlerts: domains.filter((item) => item.status === "offline").length,
    latestChange: management.pending ? "待应用配置" : management.source_file,
    trend: [],
  };
  const values = management.values as Record<string, unknown>;
  const proxy = (values.proxy ?? {}) as Record<string, unknown>;
  const middleware = (values.middleware ?? {}) as Record<string, unknown>;
  const features = (values.features ?? {}) as Record<string, unknown>;
  const backend = (values.pxy_backend ?? {}) as Record<string, unknown>;
  const servers = Array.isArray(values.servers) ? values.servers : [];
  const firstServer = (servers[0] ?? {}) as Record<string, unknown>;
  const log = (values.log ?? {}) as Record<string, unknown>;
  const gzip = (middleware.gzip ?? {}) as Record<string, unknown>;
  const websocket = (features.websocket ?? {}) as Record<string, unknown>;
  const cache = (features.proxy_cache ?? {}) as Record<string, unknown>;
  const anytls = (values.exp_config as Record<string, unknown> | undefined)?.any_tls_server as Record<string, unknown> | undefined;
  const trojanFlag = (values.exp_config as Record<string, unknown> | undefined)?.trojan_enabled;
  const trojanPath = String((values.exp_config as Record<string, unknown> | undefined)?.trojan_server ?? "");
  const backendServices = (Array.isArray(backend.servers) ? backend.servers : []).map((value) => {
    const server = (value ?? {}) as Record<string, unknown>;
    const http = (server.http ?? {}) as Record<string, unknown>;
    const transparent = (server.transparent ?? {}) as Record<string, unknown>;
    const tcp = (server.tcp ?? {}) as Record<string, unknown>;
    const host = String(server.host ?? "");
    const port = Number(server.port ?? 0);
    const target = String(transparent.target ?? tcp.target ?? (host ? `${host}${port > 0 ? `:${port}` : ""}` : "未知"));
    return { name: String(server.service_name ?? "未命名服务"), target, healthCheckPath: String(http.health_check_path ?? http.path ?? "未配置"), healthStatus: "unknown" as const, weight: Number(server.weight ?? 1) };
  });
  const experiments = [
    { key: "anytls" as const, name: "AnyTLS", enabled: Boolean(anytls?.enabled), riskLevel: "medium" as const, description: "加密流量封装与转发。", params: anytls ? Object.fromEntries(Object.entries(anytls).filter(([key]) => key !== "password")) as Record<string, string | number | boolean> : {} },
    { key: "trojan" as const, name: "Trojan", enabled: trojanPath !== "" && trojanFlag !== false, riskLevel: "high" as const, description: "Trojan 隧道实验能力。", params: { config: trojanPath } },
  ];
  return {
    overview,
    domains,
    stat,
    geo,
    connections: { gateway: gatewayConnections, front: snapshot(connections.front) },
    configs: {
      core: { mode: String(proxy.proxy_mode ?? "gateway"), listener: `${String(firstServer.host ?? "")}:${Number(firstServer.port ?? 0)}`, logLevel: String(log.log_level ?? "未配置"), maxConnections: Number(proxy.max_conns_per_host ?? 0), readTimeoutMs: Number(firstServer.read_timeout ?? 0), writeTimeoutMs: Number(firstServer.write_timeout ?? 0) },
      frontendProxy: { upstreamScheme: String((values.pxy_frontend as Record<string, unknown> | undefined)?.balancer ?? "http"), gzip: Boolean(gzip.enabled), cache: Boolean(cache.enabled), websocket: Boolean(websocket.enabled), cors: Boolean((middleware.cors as Record<string, unknown> | undefined)?.enabled), errorPage: typeof (values.error_config as Record<string, unknown> | undefined)?.error_page === "string" ? String((values.error_config as Record<string, unknown>).error_page) : "结构化映射" },
      backendServices,
      raw: management,
    },
    experiments,
  };
}
