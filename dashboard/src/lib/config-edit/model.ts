import type { DomainBinding, DomainServiceDef, ProxyRule } from "@/lib/domain-config/parse";
import type { StaticConfigModel, StaticSite } from "@/lib/static-config/load";

export type RuleMode = "service" | "static" | "host";

export interface RuleDraft {
  api: string;
  mode: RuleMode;
  service: string;
  staticRoot: string;
  proxyHost: string;
  proxyPort: string;
  useRewrite: boolean;
  rewrite: string;
}

export interface ServiceDraft {
  key: string;
  originalName: string;
  name: string;
  type: string;
  domain: string;
  host: string;
  port: string;
  preCheck: boolean;
  rules: RuleDraft[];
}

export interface BindingDraft {
  key: string;
  originalDomain: string;
  domain: string;
  service: string;
}

export interface HeaderDraft {
  key: string;
  originalName: string;
  name: string;
  value: string;
}

export interface AliasDraft {
  path: string;
  root: string;
}

export interface SiteDraft {
  key: string;
  originalName: string;
  name: string;
  type: string;
  root: string;
  index: string;
  tryFile: string;
  access: boolean;
  compress: boolean;
  alias: AliasDraft[];
  backends: RuleDraft[];
}

export interface StaticDraft {
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
  headers: HeaderDraft[];
  sites: SiteDraft[];
}

export type SaveFailure =
  | { code: "conflict" }
  | { code: "auth" }
  | { code: "parse" }
  | { code: "empty-name" }
  | { code: "duplicate-service"; name: string }
  | { code: "duplicate-domain"; name: string }
  | { code: "empty-domain" }
  | { code: "empty-binding" }
  | { code: "invalid-port"; name: string }
  | { code: "empty-site" }
  | { code: "duplicate-site"; name: string }
  | { code: "invalid-rule"; name: string }
  | { code: "failed"; message: string };

export type SaveResult = { ok: true; version: string } | { ok: false; error: SaveFailure };

export function createKey(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) return crypto.randomUUID();
  return `id-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export function blankRule(): RuleDraft {
  return { api: "", mode: "service", service: "", staticRoot: "", proxyHost: "", proxyPort: "", useRewrite: false, rewrite: "" };
}

export function blankService(): ServiceDraft {
  return { key: createKey(), originalName: "", name: "", type: "frontend", domain: "", host: "", port: "", preCheck: false, rules: [] };
}

export function blankBinding(service = ""): BindingDraft {
  return { key: createKey(), originalDomain: "", domain: "", service };
}

export function blankHeader(): HeaderDraft {
  return { key: createKey(), originalName: "", name: "", value: "" };
}

export function blankAlias(): AliasDraft {
  return { path: "", root: "" };
}

export function blankSite(): SiteDraft {
  return {
    key: createKey(),
    originalName: "",
    name: "",
    type: "WebServer",
    root: "",
    index: "index.html",
    tryFile: "index.html",
    access: false,
    compress: false,
    alias: [],
    backends: [],
  };
}

function ruleMode(rule: ProxyRule): RuleMode {
  if (rule.staticRoot) return "static";
  if (rule.proxyHost) return "host";
  return "service";
}

export function ruleToDraft(rule: ProxyRule): RuleDraft {
  return {
    api: rule.api,
    mode: ruleMode(rule),
    service: rule.service ?? "",
    staticRoot: rule.staticRoot ?? "",
    proxyHost: rule.proxyHost ?? "",
    proxyPort: rule.proxyPort === undefined ? "" : String(rule.proxyPort),
    useRewrite: rule.useRewrite,
    rewrite: rule.rewrite ?? "",
  };
}

export function serviceToDraft(service: DomainServiceDef): ServiceDraft {
  return {
    key: service.name,
    originalName: service.name,
    name: service.name,
    type: service.type || "custom",
    domain: service.domain ?? "",
    host: service.host ?? "",
    port: service.port === undefined ? "" : String(service.port),
    preCheck: service.preCheck === true,
    rules: service.proxyPass.map(ruleToDraft),
  };
}

export function bindingToDraft(binding: DomainBinding): BindingDraft {
  return { key: `${binding.domain}-${binding.service}`, originalDomain: binding.domain, domain: binding.domain, service: binding.service };
}

export function siteToDraft(site: StaticSite): SiteDraft {
  return {
    key: site.name,
    originalName: site.name,
    name: site.name,
    type: site.type,
    root: site.root,
    index: site.index,
    tryFile: site.tryFile,
    access: site.access,
    compress: site.compress,
    alias: site.alias.map((item) => ({ path: item.path, root: item.root })),
    backends: site.backends.map((item) => ruleToDraft({
      api: item.api,
      service: item.service,
      staticRoot: item.staticRoot,
      useRewrite: item.useRewrite,
      rewrite: item.rewrite,
    })),
  };
}

export function staticToDraft(config: StaticConfigModel): StaticDraft {
  return {
    host: config.host,
    port: config.port,
    balancer: config.balancer,
    fastConnect: config.fastConnect,
    http3: config.http3,
    cacheEnabled: config.cacheEnabled,
    cacheDir: config.cacheDir,
    cacheExpire: config.cacheExpire,
    memoryEnabled: config.memoryEnabled,
    memoryMaxMb: config.memoryMaxMb,
    memoryMaxFileMb: config.memoryMaxFileMb,
    memoryMaxEntries: config.memoryMaxEntries,
    memoryRevalidateSeconds: config.memoryRevalidateSeconds,
    matchers: [...config.matchers],
    headers: config.headers.map((header) => ({ key: header.name, originalName: header.name, name: header.name, value: header.value })),
    sites: config.sites.map(siteToDraft),
  };
}

export function meaningfulRule(rule: RuleDraft): boolean {
  return Boolean(rule.api.trim() || rule.service.trim() || rule.staticRoot.trim() || rule.proxyHost.trim() || rule.rewrite.trim() || rule.proxyPort.trim());
}
