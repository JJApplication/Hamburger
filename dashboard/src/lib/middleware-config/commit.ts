import { copyFileSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import path from "node:path";

import { parseHamburger, type HamburgerValue } from "@/lib/domain-config/parse";
import { ConfigEditError } from "@/lib/config-edit/commit";
import { configVersion, repoRoot } from "@/lib/config-files";
import type { CompressionDraft, ErrorConfigDraft, MiddlewareDraft } from "@/lib/middleware-config/model";
import { replaceRootField } from "@/lib/middleware-config/splice";

export const MIDDLEWARE_FILE = "config/config.hamburger";

function isRecord(value: HamburgerValue | undefined): value is { [key: string]: HamburgerValue } {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function lines(items: string[]): string[] {
  return items.map((item) => item.trim()).filter(Boolean);
}

function integer(name: string, text: string): number {
  const value = text.trim();
  if (!/^\d+$/.test(value)) throw new ConfigEditError({ code: "failed", message: `${name} 必须是非负整数` });
  return Number(value);
}

function compression(existing: HamburgerValue | undefined, draft: CompressionDraft, name: string): { [key: string]: HamburgerValue } {
  const next = isRecord(existing) ? { ...existing } : {};
  next.enabled = draft.enabled;
  next.level = integer(`${name}.level`, draft.level);
  next.threshold = integer(`${name}.threshold`, draft.threshold);
  next.types = lines(draft.types);
  return next;
}

export function applyMiddleware(existing: HamburgerValue | undefined, draft: MiddlewareDraft): { [key: string]: HamburgerValue } {
  const next = isRecord(existing) ? { ...existing } : {};
  const cors = isRecord(next.cors) ? { ...next.cors } : {};
  cors.enabled = draft.cors.enabled;
  cors.method = lines(draft.cors.methods);
  cors.origin = lines(draft.cors.origins);
  cors.header = lines(draft.cors.headers);
  next.cors = cors;

  const trace = isRecord(next.trace) ? { ...next.trace } : {};
  trace.enabled = draft.trace.enabled;
  trace.trace_id = draft.trace.traceId.trim();
  next.trace = trace;

  next.secure_header = draft.secureHeader;
  next.no_cache = draft.noCache;
  next.gzip = compression(next.gzip, draft.gzip, "gzip");
  next.zstd = compression(next.zstd, draft.zstd, "zstd");

  const sanitizer = isRecord(next.sanitizer) ? { ...next.sanitizer } : {};
  sanitizer.enabled = draft.sanitizer;
  next.sanitizer = sanitizer;

  const domainCheck = isRecord(next.domain_check) ? { ...next.domain_check } : {};
  domainCheck.enabled = draft.domainCheck;
  next.domain_check = domainCheck;

  const image = isRecord(next.image_protect) ? { ...next.image_protect } : {};
  image.enabled = draft.imageProtect.enabled;
  image.image_type = lines(draft.imageProtect.imageTypes);
  image.allow_referer = lines(draft.imageProtect.allowReferers);
  next.image_protect = image;
  delete next.fail_response;
  return next;
}

export function applyErrorConfig(existing: HamburgerValue | undefined, draft: ErrorConfigDraft): { [key: string]: HamburgerValue } {
  const next = isRecord(existing) ? { ...existing } : {};
  const mode = draft.mode.trim();
  if (!mode) throw new ConfigEditError({ code: "failed", message: "error_mode 不能为空" });
  next.error_mode = mode;
  next.enable_page_cache = draft.cache;
  const pages: { [key: string]: HamburgerValue } = {};
  const seen = new Set<string>();
  draft.pages.forEach((page) => {
    const code = page.code.trim();
    const file = page.path.trim();
    if (!code && !file) return;
    if (!/^\d{3}$/.test(code)) throw new ConfigEditError({ code: "failed", message: `错误页状态码无效：${code || "空"}` });
    if (!file) throw new ConfigEditError({ code: "failed", message: `${code} 的页面路径不能为空` });
    if (seen.has(code)) throw new ConfigEditError({ code: "failed", message: `错误页状态码重复：${code}` });
    seen.add(code);
    pages[code] = file;
  });
  next.error_page = pages;
  return next;
}

export function renderMiddleware(text: string, draft: MiddlewareDraft, errorConfig: ErrorConfigDraft): string {
  const root = parseHamburger(text);
  if (!isRecord(root)) throw new ConfigEditError({ code: "parse" });
  const rendered = replaceRootField(replaceRootField(text, "middleware", applyMiddleware(root.middleware, draft)), "error_config", applyErrorConfig(root.error_config, errorConfig));
  try {
    parseHamburger(rendered);
  } catch {
    throw new ConfigEditError({ code: "parse" });
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

export function commitMiddleware(version: string, draft: MiddlewareDraft, errorConfig: ErrorConfigDraft, rootDir = repoRoot()): string {
  const file = path.join(rootDir, MIDDLEWARE_FILE);
  let text: string;
  try {
    text = readFileSync(file, "utf8");
  } catch {
    throw new ConfigEditError({ code: "parse" });
  }
  if (version.trim() && configVersion(text) !== version.trim()) throw new ConfigEditError({ code: "conflict" });
  const rendered = renderMiddleware(text, draft, errorConfig);
  replaceFile(file, rendered);
  return configVersion(rendered);
}
