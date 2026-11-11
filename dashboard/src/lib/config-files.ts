import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import path from "node:path";

import { parseHamburger, type HamburgerValue } from "@/lib/domain-config/parse";

export function repoRoot(): string {
  return path.resolve(process.cwd(), "..");
}

export function readConfigText(relativePath: string): string {
  return readFileSync(path.join(repoRoot(), relativePath), "utf8");
}

export function configVersion(text: string): string {
  return createHash("sha256").update(text).digest("hex");
}

export function readConfigValue(relativePath: string): HamburgerValue {
  return parseHamburger(readConfigText(relativePath));
}

export function readQuotedField(relativePath: string, field: "domain_map" | "pxy_frontend_file"): string | undefined {
  try {
    const pattern = field === "domain_map" ? /domain_map\s*:\s*"([^"]+)"/ : /pxy_frontend_file\s*:\s*"([^"]+)"/;
    return readConfigText(relativePath).match(pattern)?.[1];
  } catch {
    return undefined;
  }
}

export function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}
