import { parseHamburger, type HamburgerValue } from "@/lib/domain-config/parse";
import { configVersion, errorMessage, readConfigText } from "@/lib/config-files";
import { blankSecurity, type SecurityDraft } from "@/lib/security-config/model";
import { SECURITY_FILE } from "@/lib/security-config/commit";

export interface SecurityConfigModel {
  source: string;
  version: string;
  draft: SecurityDraft;
  error?: string;
}

function isRecord(value: HamburgerValue | undefined): value is { [key: string]: HamburgerValue } {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function flag(value: HamburgerValue | undefined, fallback: boolean): boolean {
  return typeof value === "boolean" ? value : fallback;
}

function textList(value: HamburgerValue | undefined): string[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => (typeof item === "string" ? [item] : typeof item === "number" ? [String(item)] : []));
}

function numberText(value: HamburgerValue | undefined, fallback: string): string {
  return typeof value === "number" && Number.isFinite(value) ? String(value) : fallback;
}

export function securityFromValue(value: HamburgerValue | undefined): SecurityDraft {
  const draft = blankSecurity();
  if (!isRecord(value)) return draft;
  return {
    strictMode: flag(value.strict_mode, draft.strictMode),
    allowIps: textList(value.allow_ips),
    denyIps: textList(value.deny_ips),
    rateLimit: numberText(value.rate_limit, draft.rateLimit),
    maxQuerySize: numberText(value.max_query_size, draft.maxQuerySize),
    whiteList: textList(value.white_list_domain),
    hsts: flag(value.hsts, draft.hsts),
    hstsSubdomain: flag(value.hsts_subdomain, draft.hstsSubdomain),
    hstsPreload: flag(value.hsts_preload, draft.hstsPreload),
    xss: flag(value.xss_protection, draft.xss),
    iframe: flag(value.iframe_protection, draft.iframe),
    sameSite: flag(value.same_site, draft.sameSite),
  };
}

export function loadSecurityConfig(): SecurityConfigModel {
  try {
    const text = readConfigText(SECURITY_FILE);
    const root = parseHamburger(text);
    const security = isRecord(root) ? root.security : undefined;
    return { source: SECURITY_FILE, version: configVersion(text), draft: securityFromValue(security) };
  } catch (error) {
    return { source: SECURITY_FILE, version: "", draft: blankSecurity(), error: errorMessage(error, "安全配置读取失败") };
  }
}
