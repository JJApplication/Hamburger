import { normalizeDomainConfig, type DomainConfigModel } from "@/lib/domain-config/parse";
import { errorMessage, readConfigValue, readQuotedField } from "@/lib/config-files";

const SOURCE = "config/services.hamburger";

export function loadServiceConfig(): DomainConfigModel {
  const runtimeMap = readQuotedField("config/config.hamburger", "domain_map");
  try {
    return normalizeDomainConfig(readConfigValue(SOURCE), SOURCE, runtimeMap);
  } catch (error) {
    return {
      source: SOURCE,
      runtimeMap,
      bindings: [],
      services: [],
      error: errorMessage(error, "无法读取服务配置"),
    };
  }
}
