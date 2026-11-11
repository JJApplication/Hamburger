import { readFileSync } from "node:fs";
import path from "node:path";

import { configVersion } from "@/lib/config-files";
import { normalizeDomainConfig, parseHamburger, type DomainConfigModel } from "@/lib/domain-config/parse";

const SOURCE = "config/domains.hamburger";

function readRuntimeMap(configPath: string): string | undefined {
  try {
    const text = readFileSync(configPath, "utf8");
    const matched = text.match(/domain_map\s*:\s*"([^"]+)"/);
    return matched?.[1];
  } catch {
    return undefined;
  }
}

export function loadDomainConfig(): DomainConfigModel {
  const configDir = path.resolve(process.cwd(), "..", "config");
  const filePath = path.join(configDir, "domains.hamburger");
  const runtimeMap = readRuntimeMap(path.join(configDir, "config.hamburger"));
  try {
    const text = readFileSync(filePath, "utf8");
    try {
      return { ...normalizeDomainConfig(parseHamburger(text), SOURCE, runtimeMap), version: configVersion(text) };
    } catch (error) {
      return {
        source: SOURCE,
        runtimeMap,
        bindings: [],
        services: [],
        version: configVersion(text),
        error: error instanceof Error ? error.message : "无法读取域名配置",
      };
    }
  } catch (error) {
    return {
      source: SOURCE,
      runtimeMap,
      bindings: [],
      services: [],
      version: "",
      error: error instanceof Error ? error.message : "无法读取域名配置",
    };
  }
}
