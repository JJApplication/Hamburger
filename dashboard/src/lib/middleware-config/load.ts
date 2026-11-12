import { parseHamburger, type HamburgerValue } from "@/lib/domain-config/parse";
import { configVersion, errorMessage, readConfigText } from "@/lib/config-files";
import { blankErrorConfig, blankMiddleware, type CompressionDraft, type ErrorConfigDraft, type MiddlewareDraft } from "@/lib/middleware-config/model";
import { MIDDLEWARE_FILE } from "@/lib/middleware-config/commit";

export interface MiddlewareConfigModel {
  source: string;
  version: string;
  draft: MiddlewareDraft;
  errorConfig: ErrorConfigDraft;
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

function compression(value: HamburgerValue | undefined, fallback: CompressionDraft): CompressionDraft {
  if (!isRecord(value)) return fallback;
  return {
    enabled: flag(value.enabled, fallback.enabled),
    level: numberText(value.level, fallback.level),
    threshold: numberText(value.threshold, fallback.threshold),
    types: textList(value.types),
  };
}

export function middlewareFromValue(value: HamburgerValue | undefined): MiddlewareDraft {
  const draft = blankMiddleware();
  if (!isRecord(value)) return draft;
  const cors = isRecord(value.cors) ? value.cors : {};
  const trace = isRecord(value.trace) ? value.trace : {};
  const image = isRecord(value.image_protect) ? value.image_protect : {};
  return {
    cors: {
      enabled: flag(cors.enabled, draft.cors.enabled),
      methods: textList(cors.method),
      origins: textList(cors.origin),
      headers: textList(cors.header),
    },
    trace: {
      enabled: flag(trace.enabled, draft.trace.enabled),
      traceId: typeof trace.trace_id === "string" ? trace.trace_id : draft.trace.traceId,
    },
    secureHeader: flag(value.secure_header, draft.secureHeader),
    noCache: flag(value.no_cache, draft.noCache),
    gzip: compression(value.gzip, draft.gzip),
    zstd: compression(value.zstd, draft.zstd),
    sanitizer: flag(isRecord(value.sanitizer) ? value.sanitizer.enabled : undefined, draft.sanitizer),
    domainCheck: flag(isRecord(value.domain_check) ? value.domain_check.enabled : undefined, draft.domainCheck),
    imageProtect: {
      enabled: flag(image.enabled, draft.imageProtect.enabled),
      imageTypes: textList(image.image_type),
      allowReferers: textList(image.allow_referer),
    },
  };
}

export function errorConfigFromValue(value: HamburgerValue | undefined): ErrorConfigDraft {
  const draft = blankErrorConfig();
  if (!isRecord(value)) return draft;
  const pages = isRecord(value.error_page)
    ? Object.entries(value.error_page).flatMap(([code, path], index) => (typeof path === "string" ? [{ key: `${code}-${index}`, code, path }] : []))
    : [];
  return {
    mode: typeof value.error_mode === "string" && value.error_mode.trim() ? value.error_mode : draft.mode,
    cache: flag(value.enable_page_cache, draft.cache),
    pages,
  };
}

export function loadMiddlewareConfig(): MiddlewareConfigModel {
  try {
    const text = readConfigText(MIDDLEWARE_FILE);
    const root = parseHamburger(text);
    const middleware = isRecord(root) ? root.middleware : undefined;
    const errorConfig = isRecord(root) ? root.error_config : undefined;
    return { source: MIDDLEWARE_FILE, version: configVersion(text), draft: middlewareFromValue(middleware), errorConfig: errorConfigFromValue(errorConfig) };
  } catch (error) {
    return { source: MIDDLEWARE_FILE, version: "", draft: blankMiddleware(), errorConfig: blankErrorConfig(), error: errorMessage(error, "中间件配置读取失败") };
  }
}
