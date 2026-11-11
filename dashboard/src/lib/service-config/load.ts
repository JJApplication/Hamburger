import { errorMessage, configVersion, readConfigText, readQuotedField } from "@/lib/config-files";
import { normalizeDomainConfig, parseHamburger, type DomainConfigModel } from "@/lib/domain-config/parse";

const SOURCE = "config/services.hamburger";

export function loadServiceConfig(): DomainConfigModel {
  const runtimeMap = readQuotedField("config/config.hamburger", "domain_map");
  try {
    const text = readConfigText(SOURCE);
    try {
      return { ...normalizeDomainConfig(parseHamburger(text), SOURCE, runtimeMap), version: configVersion(text) };
    } catch (error) {
      return { source: SOURCE, runtimeMap, bindings: [], services: [], version: configVersion(text), error: errorMessage(error, "无法读取服务配置") };
    }
  } catch (error) {
    return { source: SOURCE, runtimeMap, bindings: [], services: [], version: "", error: errorMessage(error, "无法读取服务配置") };
  }
}
