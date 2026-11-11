import { copyFileSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import path from "node:path";

import { configVersion, repoRoot } from "@/lib/config-files";
import { isBareHamburger, parseHamburger, serializeHamburger, type HamburgerValue } from "@/lib/domain-config/parse";
import { meaningfulRule, type BindingDraft, type HeaderDraft, type RuleDraft, type SaveFailure, type ServiceDraft, type SiteDraft, type StaticDraft } from "@/lib/config-edit/model";

export class ConfigEditError extends Error {
  failure: SaveFailure;

  constructor(failure: SaveFailure) {
    super(failure.code);
    this.failure = failure;
  }
}

function isRecord(value: HamburgerValue | undefined): value is { [key: string]: HamburgerValue } {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function asString(value: HamburgerValue | undefined): string {
  return typeof value === "string" ? value.trim() : "";
}

function fail(failure: SaveFailure): never {
  throw new ConfigEditError(failure);
}

function assignText(record: { [key: string]: HamburgerValue }, key: string, text: string) {
  const trimmed = text.trim();
  if (!trimmed) delete record[key];
  else record[key] = trimmed;
}

function assignScalar(record: { [key: string]: HamburgerValue }, key: string, text: string, label: string) {
  const trimmed = text.trim();
  if (!trimmed) {
    delete record[key];
    return;
  }
  if (/^-?\d+(\.\d+)?$/.test(trimmed)) {
    record[key] = Number(trimmed);
    return;
  }
  if (isBareHamburger(trimmed)) {
    record[key] = trimmed;
    return;
  }
  fail({ code: "invalid-port", name: label });
}

function assignBool(record: { [key: string]: HamburgerValue }, key: string, value: boolean) {
  if (value) record[key] = true;
  else if (record[key] !== undefined) record[key] = false;
}

function activeRules(rules: RuleDraft[]): RuleDraft[] {
  return rules.filter(meaningfulRule);
}

function applyRule(base: { [key: string]: HamburgerValue }, rule: RuleDraft, label: string): { [key: string]: HamburgerValue } {
  const next = { ...base };
  assignText(next, "api", rule.api);
  assignBool(next, "use_rewrite", rule.useRewrite);
  assignText(next, "rewrite", rule.rewrite);
  if (rule.mode === "static") {
    if (!rule.staticRoot.trim()) fail({ code: "invalid-rule", name: label });
    delete next.service;
    delete next.proxy_direct;
    const direct = isRecord(next.static_direct) ? { ...next.static_direct } : {};
    direct.static_root = rule.staticRoot.trim();
    next.static_direct = direct;
  } else if (rule.mode === "host") {
    if (!rule.proxyHost.trim()) fail({ code: "invalid-rule", name: label });
    delete next.service;
    delete next.static_direct;
    const direct = isRecord(next.proxy_direct) ? { ...next.proxy_direct } : {};
    direct.proxy_host = rule.proxyHost.trim();
    assignScalar(direct, "proxy_port", rule.proxyPort, label);
    next.proxy_direct = direct;
  } else {
    if (!rule.service.trim()) fail({ code: "invalid-rule", name: label });
    delete next.static_direct;
    delete next.proxy_direct;
    next.service = rule.service.trim();
  }
  return next;
}

function mergeRules(existing: HamburgerValue | undefined, rules: RuleDraft[], label: string): HamburgerValue[] {
  const source = Array.isArray(existing) ? existing.flatMap((item) => (isRecord(item) ? [item] : [])) : [];
  const used = new Set<number>();
  const slots: Array<{ [key: string]: HamburgerValue }> = rules.map(() => ({}));
  rules.forEach((rule, index) => {
    const api = rule.api.trim();
    if (!api) return;
    const found = source.findIndex((item, itemIndex) => !used.has(itemIndex) && asString(item.api) === api);
    if (found >= 0) {
      used.add(found);
      slots[index] = { ...source[found] };
    }
  });
  rules.forEach((rule, index) => {
    if (Object.keys(slots[index]).length > 0) return;
    if (index < source.length && !used.has(index)) {
      used.add(index);
      slots[index] = { ...source[index] };
    }
  });
  return rules.map((rule, index) => applyRule(slots[index], rule, label));
}

function validateServices(services: ServiceDraft[]) {
  const names = new Set<string>();
  services.forEach((service) => {
    const name = service.name.trim();
    if (!name) fail({ code: "empty-name" });
    if (names.has(name)) fail({ code: "duplicate-service", name });
    names.add(name);
    if (service.port.trim()) assignScalar({}, "port", service.port, name);
  });
}

export function applyServices(root: { [key: string]: HamburgerValue }, services: ServiceDraft[]) {
  validateServices(services);
  const source = Array.isArray(root.services) ? root.services.flatMap((item) => (isRecord(item) ? [item] : [])) : [];
  const used = new Set<number>();
  const bases = services.map((service) => {
    if (!service.originalName) return {};
    const found = source.findIndex((item, index) => !used.has(index) && asString(item.service_name) === service.originalName);
    if (found < 0) return {};
    used.add(found);
    return { ...source[found] };
  });
  root.services = services.map((service, index) => {
    const next = bases[index];
    const label = service.name.trim();
    next.service_name = label;
    next.service_type = service.type.trim() || "custom";
    assignText(next, "service_domain", service.domain);
    assignText(next, "host", service.host);
    assignScalar(next, "port", service.port, label);
    if (service.preCheck) {
      const pre = isRecord(next.pre_check) ? { ...next.pre_check } : {};
      pre.enabled = true;
      next.pre_check = pre;
    } else if (isRecord(next.pre_check)) {
      next.pre_check = { ...next.pre_check, enabled: false };
    }
    const rules = mergeRules(next.proxy_pass, activeRules(service.rules), label);
    if (rules.length > 0) next.proxy_pass = rules;
    else delete next.proxy_pass;
    return next;
  });
}

export function applyBindings(root: { [key: string]: HamburgerValue }, bindings: BindingDraft[]) {
  const domains = new Set<string>();
  bindings.forEach((binding) => {
    const domain = binding.domain.trim();
    if (!domain) fail({ code: "empty-domain" });
    if (!binding.service.trim()) fail({ code: "empty-binding" });
    if (domains.has(domain)) fail({ code: "duplicate-domain", name: domain });
    domains.add(domain);
  });
  const source = Array.isArray(root.domain_service) ? root.domain_service.flatMap((item) => (isRecord(item) ? [item] : [])) : [];
  const used = new Set<number>();
  root.domain_service = bindings.map((binding) => {
    let base: { [key: string]: HamburgerValue } = {};
    if (binding.originalDomain) {
      const found = source.findIndex((item, index) => !used.has(index) && asString(item.domain) === binding.originalDomain);
      if (found >= 0) {
        used.add(found);
        base = { ...source[found] };
      }
    }
    base.domain = binding.domain.trim();
    base.service = binding.service.trim();
    return base;
  });
}

function mergeHeaders(existing: HamburgerValue | undefined, headers: HeaderDraft[]): HamburgerValue[] {
  const source = Array.isArray(existing) ? existing.flatMap((item) => (isRecord(item) ? [item] : [])) : [];
  const used = new Set<number>();
  return headers.filter((header) => header.name.trim()).map((header, index) => {
    const name = header.name.trim();
    let found = source.findIndex((item, itemIndex) => !used.has(itemIndex) && (asString(item.name) === header.originalName || asString(item.name) === name));
    if (found < 0 && index < source.length && !used.has(index)) found = index;
    const base = found >= 0 ? { ...source[found] } : {};
    if (found >= 0) used.add(found);
    base.name = name;
    base.value = header.value;
    return base;
  });
}

function applySite(base: { [key: string]: HamburgerValue }, site: SiteDraft): { [key: string]: HamburgerValue } {
  const next = { ...base };
  const label = site.name.trim();
  assignText(next, "type", site.type);
  next.name = label;
  assignText(next, "root", site.root);
  assignText(next, "index", site.index);
  assignText(next, "try_file", site.tryFile);
  assignBool(next, "access", site.access);
  assignBool(next, "compress", site.compress);
  const alias: { [key: string]: HamburgerValue } = {};
  site.alias.forEach((item) => {
    const route = item.path.trim();
    if (route) alias[route] = item.root;
  });
  if (Object.keys(alias).length > 0) next.alias = alias;
  else delete next.alias;
  const rules = mergeRules(next.backends, activeRules(site.backends), label);
  if (rules.length > 0) next.backends = rules;
  else delete next.backends;
  return next;
}

export function applyStatic(root: { [key: string]: HamburgerValue }, draft: StaticDraft) {
  const names = new Set<string>();
  draft.sites.forEach((site) => {
    const name = site.name.trim();
    if (!name) fail({ code: "empty-site" });
    if (names.has(name)) fail({ code: "duplicate-site", name });
    names.add(name);
  });
  assignText(root, "host", draft.host);
  assignScalar(root, "port", draft.port, "port");
  assignText(root, "balancer", draft.balancer);
  const fast = isRecord(root.exp_fast_connect) ? { ...root.exp_fast_connect } : {};
  assignBool(fast, "enabled", draft.fastConnect);
  const http3 = isRecord(fast.http3) ? { ...fast.http3 } : {};
  assignBool(http3, "enabled", draft.http3);
  if (Object.keys(http3).length > 0 || isRecord(fast.http3)) fast.http3 = http3;
  if (Object.keys(fast).length > 0 || isRecord(root.exp_fast_connect)) root.exp_fast_connect = fast;
  const cache = isRecord(root.cache) ? { ...root.cache } : {};
  assignBool(cache, "enable", draft.cacheEnabled);
  assignText(cache, "dir", draft.cacheDir);
  assignScalar(cache, "expire", draft.cacheExpire, "expire");
  assignBool(cache, "memory_enable", draft.memoryEnabled);
  assignScalar(cache, "memory_max_mb", draft.memoryMaxMb, "memory");
  assignScalar(cache, "memory_max_file_mb", draft.memoryMaxFileMb, "memory");
  assignScalar(cache, "memory_max_entries", draft.memoryMaxEntries, "memory");
  assignScalar(cache, "memory_revalidate_seconds", draft.memoryRevalidateSeconds, "memory");
  cache.matcher = draft.matchers.map((item) => item.trim()).filter(Boolean);
  if (Object.keys(cache).length > 0 || isRecord(root.cache)) root.cache = cache;
  const headers = mergeHeaders(root.custom_headers, draft.headers);
  if (headers.length > 0 || Array.isArray(root.custom_headers)) root.custom_headers = headers;
  const source = Array.isArray(root.servers) ? root.servers.flatMap((item) => (isRecord(item) ? [item] : [])) : [];
  const used = new Set<number>();
  root.servers = draft.sites.map((site) => {
    let base: { [key: string]: HamburgerValue } = {};
    if (site.originalName) {
      const found = source.findIndex((item, index) => !used.has(index) && asString(item.name) === site.originalName);
      if (found >= 0) {
        used.add(found);
        base = { ...source[found] };
      }
    }
    return applySite(base, site);
  });
}

function requireRoot(value: HamburgerValue): { [key: string]: HamburgerValue } {
  if (!isRecord(value)) fail({ code: "parse" });
  return value;
}

export function renderEdited(text: string, mutate: (root: { [key: string]: HamburgerValue }) => void): string {
  const root = requireRoot(parseHamburger(text));
  mutate(root);
  const rendered = serializeHamburger(root);
  try {
    parseHamburger(rendered);
  } catch {
    fail({ code: "parse" });
  }
  return rendered;
}

function replaceFile(file: string, text: string) {
  const temporary = `${file}.${process.pid}.tmp`;
  writeFileSync(temporary, text, "utf8");
  try {
    copyFileSync(temporary, file);
  } finally {
    rmSync(temporary, { force: true });
  }
}

export function commitEditedFile(relativePath: string, version: string, mutate: (root: { [key: string]: HamburgerValue }) => void, rootDir = repoRoot()): string {
  const file = path.join(rootDir, relativePath);
  let text: string;
  try {
    text = readFileSync(file, "utf8");
  } catch {
    fail({ code: "parse" });
  }
  if (version.trim() && configVersion(text) !== version.trim()) fail({ code: "conflict" });
  const rendered = renderEdited(text, mutate);
  replaceFile(file, rendered);
  return configVersion(rendered);
}

export const SERVICE_FILE = "config/services.hamburger";
export const DOMAIN_FILE = "config/domains.hamburger";
export const STATIC_FILE = "config/frontend.hamburger";
