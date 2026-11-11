import { mkdtempSync, readFileSync, cpSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";

import { configVersion } from "../src/lib/config-files";
import { applyBindings, applyServices, applyStatic, commitEditedFile, ConfigEditError, renderEdited } from "../src/lib/config-edit/commit";
import { bindingToDraft, serviceToDraft, staticToDraft } from "../src/lib/config-edit/model";
import { normalizeDomainConfig, parseHamburger, type HamburgerValue } from "../src/lib/domain-config/parse";
import { loadStaticConfig } from "../src/lib/static-config/load";

const root = path.resolve(process.cwd(), "..");
const files = ["config/services.hamburger", "config/domains.hamburger", "config/frontend.hamburger"];

function diff(left: unknown, right: unknown, at = "$"): string[] {
  if (Object.is(left, right)) return [];
  if (typeof left !== typeof right || Array.isArray(left) !== Array.isArray(right)) return [`${at}: ${JSON.stringify(left)} != ${JSON.stringify(right)}`];
  if (Array.isArray(left) && Array.isArray(right)) {
    const found: string[] = [];
    if (left.length !== right.length) found.push(`${at}.length ${left.length} != ${right.length}`);
    for (let index = 0; index < Math.max(left.length, right.length); index += 1) found.push(...diff(left[index], right[index], `${at}[${index}]`));
    return found;
  }
  if (left && right && typeof left === "object") {
    const a = left as Record<string, unknown>;
    const b = right as Record<string, unknown>;
    const found: string[] = [];
    for (const key of new Set([...Object.keys(a), ...Object.keys(b)])) found.push(...diff(a[key], b[key], `${at}.${key}`));
    return found;
  }
  return [`${at}: ${JSON.stringify(left)} != ${JSON.stringify(right)}`];
}

function assertEqual(left: unknown, right: unknown, label: string) {
  const found = diff(left, right).slice(0, 12);
  if (found.length > 0) throw new Error(`${label}\n${found.join("\n")}`);
}

function read(relativePath: string): string {
  return readFileSync(path.join(root, relativePath), "utf8");
}

function contains(value: HamburgerValue, expected: string): boolean {
  if (value === expected) return true;
  if (Array.isArray(value)) return value.some((item) => contains(item, expected));
  if (value && typeof value === "object") return Object.values(value).some((item) => contains(item, expected));
  return false;
}

const before = Object.fromEntries(files.map((file) => [file, configVersion(read(file))]));

for (const file of files) {
  const text = read(file);
  const rendered = renderEdited(text, (value) => {
    if (file.endsWith("frontend.hamburger")) {
      const loaded = loadStaticConfig();
      applyStatic(value, staticToDraft(loaded));
      return;
    }
    const model = normalizeDomainConfig(parseHamburger(text), file);
    if (file.endsWith("domains.hamburger")) applyBindings(value, model.bindings.map(bindingToDraft));
    applyServices(value, model.services.map(serviceToDraft));
  });
  assertEqual(parseHamburger(text), parseHamburger(rendered), file);
}

const frontend = parseHamburger(read("config/frontend.hamburger"));
const frontendAgain = parseHamburger(renderEdited(read("config/frontend.hamburger"), (value) => applyStatic(value, staticToDraft(loadStaticConfig()))));
if (!contains(frontendAgain, "@Localhost")) throw new Error("丢失 @Localhost");
if (!contains(frontendAgain, "50 * 1<<20")) throw new Error("丢失缓存表达式");
assertEqual(frontend, frontendAgain, "frontend identity");

const preserved = `{
  services: [
    {
      service_name: "A",
      service_type: "frontend",
      pre_check: { enabled: true, ttl_seconds: 10, cache_max_mb: 4 },
      proxy_pass: [
        { api: "/a", static_direct: { static_root: "/tmp", allow_ext: ["png"], direct_access: true } }
      ]
    }
  ]
}`;
const preservedModel = normalizeDomainConfig(parseHamburger(preserved), "memory");
assertEqual(parseHamburger(preserved), parseHamburger(renderEdited(preserved, (value) => applyServices(value, preservedModel.services.map(serviceToDraft)))), "preserve hidden fields");

const serviceText = read("config/services.hamburger");
const serviceModel = normalizeDomainConfig(parseHamburger(serviceText), "config/services.hamburger");
const serviceDrafts = serviceModel.services.map(serviceToDraft);
const target = serviceDrafts.find((service) => service.name === "BlogNext")?.rules.find((rule) => rule.api === "/images");
if (!target) throw new Error("找不到 BlogNext 静态规则");
target.staticRoot = "/tmp/edited-images";
const editedServices = parseHamburger(renderEdited(serviceText, (value) => applyServices(value, serviceDrafts)));
const expectedServices = parseHamburger(serviceText) as { services: Array<{ service_name?: string; proxy_pass?: Array<{ api?: string; static_direct?: { static_root?: string } }> }> };
const expectedRule = expectedServices.services.find((service) => service.service_name === "BlogNext")?.proxy_pass?.find((rule) => rule.api === "/images");
if (!expectedRule?.static_direct) throw new Error("原规则缺少静态目录");
expectedRule.static_direct.static_root = "/tmp/edited-images";
assertEqual(expectedServices, editedServices, "service route edit");

const domainText = read("config/domains.hamburger");
const domainModel = normalizeDomainConfig(parseHamburger(domainText), "config/domains.hamburger");
const bindings = domainModel.bindings.map(bindingToDraft);
const binding = bindings.find((item) => item.domain === "renj.io");
if (!binding) throw new Error("找不到 renj.io");
binding.domain = "edited.renj.io";
const editedDomains = parseHamburger(renderEdited(domainText, (value) => {
  applyBindings(value, bindings);
  applyServices(value, domainModel.services.map(serviceToDraft));
}));
const expectedDomains = parseHamburger(domainText) as { domain_service: Array<{ domain?: string }> };
const expectedBinding = expectedDomains.domain_service.find((item) => item.domain === "renj.io");
if (!expectedBinding) throw new Error("原映射缺少 renj.io");
expectedBinding.domain = "edited.renj.io";
assertEqual(expectedDomains, editedDomains, "domain mapping edit");

const staticText = read("config/frontend.hamburger");
const staticDraft = staticToDraft(loadStaticConfig());
const alias = staticDraft.sites.find((site) => site.name === "BlogFront")?.alias.find((item) => item.path === "/images");
if (!alias) throw new Error("找不到 BlogFront 别名");
alias.root = "/tmp/edited-alias";
const editedStatic = parseHamburger(renderEdited(staticText, (value) => applyStatic(value, staticDraft)));
const expectedStatic = parseHamburger(staticText) as { servers: Array<{ name?: string; alias?: Record<string, string> }> };
const expectedSite = expectedStatic.servers.find((site) => site.name === "BlogFront");
if (!expectedSite?.alias) throw new Error("原站点缺少别名");
expectedSite.alias["/images"] = "/tmp/edited-alias";
assertEqual(expectedStatic, editedStatic, "static alias edit");

const temp = mkdtempSync(path.join(tmpdir(), "hamburger-config-"));
try {
  cpSync(path.join(root, "config"), path.join(temp, "config"), { recursive: true });
  const version = configVersion(read("config/services.hamburger"));
  const fresh = commitEditedFile("config/services.hamburger", version, (value) => applyServices(value, serviceModel.services.map(serviceToDraft)), temp);
  let conflicted = false;
  try {
    commitEditedFile("config/services.hamburger", version, (value) => applyServices(value, serviceDrafts), temp);
  } catch (error) {
    conflicted = error instanceof ConfigEditError && error.failure.code === "conflict";
  }
  if (!conflicted) throw new Error("版本冲突没有被拒绝");
  const saved = commitEditedFile("config/services.hamburger", fresh, (value) => applyServices(value, serviceDrafts), temp);
  const written = parseHamburger(readFileSync(path.join(temp, "config/services.hamburger"), "utf8"));
  assertEqual(editedServices, written, "temp file write");
  if (!saved) throw new Error("保存没有返回版本");
} finally {
  rmSync(temp, { recursive: true, force: true });
}

const after = Object.fromEntries(files.map((file) => [file, configVersion(read(file))]));
assertEqual(before, after, "real files unchanged");
console.log("config edit checks passed");
