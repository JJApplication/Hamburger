import { copyFileSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import path from "node:path";

import { parseHamburger, type HamburgerValue } from "@/lib/domain-config/parse";
import { ConfigEditError } from "@/lib/config-edit/commit";
import { configVersion, repoRoot } from "@/lib/config-files";
import { replaceRootField } from "@/lib/middleware-config/splice";
import type { SecurityDraft } from "@/lib/security-config/model";

export const SECURITY_FILE = "config/config.hamburger";

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

export function applySecurity(existing: HamburgerValue | undefined, draft: SecurityDraft): { [key: string]: HamburgerValue } {
  const next = isRecord(existing) ? { ...existing } : {};
  next.strict_mode = draft.strictMode;
  next.allow_ips = lines(draft.allowIps);
  next.deny_ips = lines(draft.denyIps);
  next.rate_limit = integer("rate_limit", draft.rateLimit);
  next.max_query_size = integer("max_query_size", draft.maxQuerySize);
  next.white_list_domain = lines(draft.whiteList);
  next.hsts = draft.hsts;
  next.hsts_subdomain = draft.hstsSubdomain;
  next.hsts_preload = draft.hstsPreload;
  next.xss_protection = draft.xss;
  next.iframe_protection = draft.iframe;
  next.same_site = draft.sameSite;
  return next;
}

export function renderSecurity(text: string, draft: SecurityDraft): string {
  const root = parseHamburger(text);
  if (!isRecord(root)) throw new ConfigEditError({ code: "parse" });
  const rendered = replaceRootField(text, "security", applySecurity(root.security, draft));
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

export function commitSecurity(version: string, draft: SecurityDraft, rootDir = repoRoot()): string {
  const file = path.join(rootDir, SECURITY_FILE);
  let text: string;
  try {
    text = readFileSync(file, "utf8");
  } catch {
    throw new ConfigEditError({ code: "parse" });
  }
  if (version.trim() && configVersion(text) !== version.trim()) throw new ConfigEditError({ code: "conflict" });
  const rendered = renderSecurity(text, draft);
  replaceFile(file, rendered);
  return configVersion(rendered);
}
