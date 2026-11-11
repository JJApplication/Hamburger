import type { HamburgerValue } from "@/lib/domain-config/parse";
import { configVersion, errorMessage, readConfigText, readQuotedField } from "@/lib/config-files";
import { parseHamburger } from "@/lib/domain-config/parse";

const SOURCE = "config/frontend.hamburger";

export interface StaticRoute {
  api: string;
  service?: string;
  staticRoot?: string;
  useRewrite: boolean;
  rewrite?: string;
}

export interface StaticSite {
  name: string;
  type: string;
  root: string;
  index: string;
  tryFile: string;
  access: boolean;
  compress: boolean;
  alias: { path: string; root: string }[];
  backends: StaticRoute[];
}

export interface StaticConfigModel {
  source: string;
  configuredFile?: string;
  host: string;
  port: string;
  balancer: string;
  fastConnect: boolean;
  http3: boolean;
  cacheEnabled: boolean;
  cacheDir: string;
  cacheExpire: string;
  memoryEnabled: boolean;
  memoryMaxMb: string;
  memoryMaxFileMb: string;
  memoryMaxEntries: string;
  memoryRevalidateSeconds: string;
  matchers: string[];
  headers: { name: string; value: string }[];
  sites: StaticSite[];
  error?: string;
  version: string;
}

function isRecord(value: HamburgerValue | undefined): value is { [key: string]: HamburgerValue } {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function scalar(value: HamburgerValue | undefined): string {
  if (typeof value === "string") return value.trim();
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return "";
}

function stringList(value: HamburgerValue | undefined): string[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    const text = scalar(item);
    return text ? [text] : [];
  });
}

function routes(value: HamburgerValue | undefined): StaticRoute[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    if (!isRecord(item)) return [];
    const staticDirect = isRecord(item.static_direct) ? item.static_direct : undefined;
    const api = scalar(item.api);
    const service = scalar(item.service);
    const staticRoot = scalar(staticDirect?.static_root);
    if (!api && !service && !staticRoot) return [];
    return [{
      api,
      service: service || undefined,
      staticRoot: staticRoot || undefined,
      useRewrite: item.use_rewrite === true,
      rewrite: scalar(item.rewrite) || undefined,
    }];
  });
}

function sites(value: HamburgerValue | undefined): StaticSite[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    if (!isRecord(item)) return [];
    const name = scalar(item.name);
    if (!name) return [];
    const alias = isRecord(item.alias)
      ? Object.entries(item.alias).flatMap(([route, root]) => {
          const text = scalar(root);
          return text ? [{ path: route, root: text }] : [];
        })
      : [];
    return [{
      name,
      type: scalar(item.type),
      root: scalar(item.root),
      index: scalar(item.index),
      tryFile: scalar(item.try_file),
      access: item.access === true,
      compress: item.compress === true,
      alias,
      backends: routes(item.backends),
    }];
  });
}

export function loadStaticConfig(): StaticConfigModel {
  const configuredFile = readQuotedField("config/config.hamburger", "pxy_frontend_file");
  const empty: StaticConfigModel = {
    source: SOURCE,
    configuredFile,
    host: "",
    port: "",
    balancer: "",
    fastConnect: false,
    http3: false,
    cacheEnabled: false,
    cacheDir: "",
    cacheExpire: "",
    memoryEnabled: false,
    memoryMaxMb: "",
    memoryMaxFileMb: "",
    memoryMaxEntries: "",
    memoryRevalidateSeconds: "",
    matchers: [],
    headers: [],
    sites: [],
    version: "",
  };
  try {
    const text = readConfigText(SOURCE);
    const version = configVersion(text);
    try {
      const value = parseHamburger(text);
      if (!isRecord(value)) throw new Error("静态配置根节点必须是对象");
      const cache = isRecord(value.cache) ? value.cache : undefined;
      const fastConnect = isRecord(value.exp_fast_connect) ? value.exp_fast_connect : undefined;
      const http3 = fastConnect && isRecord(fastConnect.http3) ? fastConnect.http3 : undefined;
      const headerList = Array.isArray(value.custom_headers) ? value.custom_headers.flatMap((item) => {
        if (!isRecord(item)) return [];
        const name = scalar(item.name);
        const headerValue = scalar(item.value);
        return name ? [{ name, value: headerValue }] : [];
      }) : [];
      return {
        ...empty,
        host: scalar(value.host),
        port: scalar(value.port),
        balancer: scalar(value.balancer),
        fastConnect: fastConnect?.enabled === true,
        http3: http3?.enabled === true,
        cacheEnabled: cache?.enable === true,
        cacheDir: scalar(cache?.dir),
        cacheExpire: scalar(cache?.expire),
        memoryEnabled: cache?.memory_enable === true,
        memoryMaxMb: scalar(cache?.memory_max_mb),
        memoryMaxFileMb: scalar(cache?.memory_max_file_mb),
        memoryMaxEntries: scalar(cache?.memory_max_entries),
        memoryRevalidateSeconds: scalar(cache?.memory_revalidate_seconds),
        matchers: stringList(cache?.matcher),
        headers: headerList,
        sites: sites(value.servers),
        version,
      };
    } catch (error) {
      return { ...empty, version, error: errorMessage(error, "无法读取静态配置") };
    }
  } catch (error) {
    return { ...empty, error: errorMessage(error, "无法读取静态配置") };
  }
}
