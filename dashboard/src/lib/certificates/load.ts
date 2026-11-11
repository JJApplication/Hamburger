import { createPrivateKey, X509Certificate } from "node:crypto";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";

import type { HamburgerValue } from "@/lib/domain-config/parse";
import { errorMessage, readConfigValue, readQuotedField, repoRoot } from "@/lib/config-files";

const SOURCE = "config/config.hamburger";
const EXPIRING_DAYS = 30;

export type CertificateStatus = "valid" | "expiring" | "expired" | "missing" | "invalid" | "key";

export type KeyMatch = "matched" | "mismatch" | "missing" | "unread" | "unchecked";

export interface CertificateDomain {
  name: string;
  covered: boolean | null;
}

export interface CertificateIdentity {
  role: "leaf" | "intermediate";
  commonName: string;
  subject: string;
  issuer: string;
  subjectAltNames: string[];
  validFrom: string;
  validTo: string;
  daysRemaining: number;
  serialNumber: string;
  fingerprint: string;
}

export interface CertificateCheck {
  id: string;
  name: string;
  server: string;
  minVersion: string;
  autoTls: boolean | null;
  domains: CertificateDomain[];
  certFile: string;
  keyFile: string;
  status: CertificateStatus;
  keyMatch: KeyMatch;
  error: string;
  chain: CertificateIdentity[];
}

export interface CertificateModel {
  source: string;
  checkedAt: string;
  certificates: CertificateCheck[];
  error?: string;
}

interface ConfigRecord {
  [key: string]: HamburgerValue;
}

interface Ancestor {
  key: string;
  node: ConfigRecord;
}

const clock = new Intl.DateTimeFormat("zh-CN", {
  timeZone: "Asia/Shanghai",
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});

function isRecord(value: HamburgerValue | undefined): value is ConfigRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function asString(value: HamburgerValue | undefined): string {
  return typeof value === "string" ? value.trim() : "";
}

function formatInstant(date: Date): string {
  return clock.format(date);
}

function daysUntil(date: Date, now: number): number {
  return Math.floor((date.getTime() - now) / 86_400_000);
}

function namedField(source: string, name: string): string {
  return source.match(new RegExp(`(?:^|\\n)${name}=([^\\n]+)`))?.[1]?.trim() ?? "";
}

function oneLine(source: string): string {
  return source.replace(/\n/g, ", ");
}

function altNames(source: string | undefined): string[] {
  if (!source) return [];
  return source.split(",").flatMap((item) => {
    const name = item.trim().replace(/^(DNS|IP Address):/i, "").trim();
    return name ? [name] : [];
  });
}

function tlsLabel(value: string): string {
  const labels: Record<string, string> = { TLS10: "TLS 1.0", TLS11: "TLS 1.1", TLS12: "TLS 1.2", TLS13: "TLS 1.3" };
  return labels[value] ?? value;
}

function covers(pattern: string, domain: string): boolean {
  const san = pattern.toLowerCase();
  const host = domain.toLowerCase();
  if (san === host) return true;
  if (!san.startsWith("*.")) return false;
  const suffix = san.slice(1);
  if (!host.endsWith(suffix) || host.length <= suffix.length) return false;
  const prefix = host.slice(0, host.length - suffix.length);
  return prefix.length > 0 && !prefix.includes(".");
}

function resolvePath(file: string): string {
  return path.isAbsolute(file) ? file : path.resolve(repoRoot(), file);
}

function readChain(file: string): X509Certificate[] {
  const bytes = readFileSync(file);
  const text = bytes.toString("utf8");
  const pems = text.match(/-----BEGIN CERTIFICATE-----[\s\S]+?-----END CERTIFICATE-----/g);
  if (pems && pems.length > 0) return pems.map((pem) => new X509Certificate(pem));
  return [new X509Certificate(bytes)];
}

function describeCertificate(cert: X509Certificate, role: "leaf" | "intermediate", now: number): CertificateIdentity {
  const validFrom = new Date(cert.validFrom);
  const validTo = new Date(cert.validTo);
  return {
    role,
    commonName: namedField(cert.subject, "CN") || oneLine(cert.subject),
    subject: oneLine(cert.subject),
    issuer: oneLine(cert.issuer),
    subjectAltNames: altNames(cert.subjectAltName),
    validFrom: Number.isNaN(validFrom.getTime()) ? cert.validFrom : formatInstant(validFrom),
    validTo: Number.isNaN(validTo.getTime()) ? cert.validTo : formatInstant(validTo),
    daysRemaining: Number.isNaN(validTo.getTime()) ? 0 : daysUntil(validTo, now),
    serialNumber: cert.serialNumber,
    fingerprint: cert.fingerprint256,
  };
}

function keyMatch(cert: X509Certificate | undefined, keyFile: string): KeyMatch {
  if (!keyFile) return "unchecked";
  if (!existsSync(resolvePath(keyFile))) return "missing";
  if (!cert) return "unchecked";
  try {
    return cert.checkPrivateKey(createPrivateKey(readFileSync(resolvePath(keyFile)))) ? "matched" : "mismatch";
  } catch {
    return "unread";
  }
}

function stringList(value: HamburgerValue | undefined): string[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    const text = asString(item);
    return text ? [text] : [];
  });
}

function ancestorText(ancestors: Ancestor[], field: "name" | "min_version"): string {
  for (let index = ancestors.length - 1; index >= 0; index -= 1) {
    const text = asString(ancestors[index]?.node[field]);
    if (text) return text;
  }
  return "";
}

function ancestorFlag(ancestors: Ancestor[], field: "auto_tls"): boolean | null {
  for (let index = ancestors.length - 1; index >= 0; index -= 1) {
    const value = ancestors[index]?.node[field];
    if (typeof value === "boolean") return value;
  }
  return null;
}

function collect(value: HamburgerValue, key: string, ancestors: Ancestor[], found: { node: ConfigRecord; ancestors: Ancestor[] }[]): void {
  if (Array.isArray(value)) {
    value.forEach((item, index) => collect(item, String(index), ancestors, found));
    return;
  }
  if (!isRecord(value)) return;
  const chain = [...ancestors, { key, node: value }];
  if (asString(value.cert_file)) found.push({ node: value, ancestors: chain });
  Object.entries(value).forEach(([childKey, child]) => collect(child, childKey, chain, found));
}

function inspect(entry: { node: ConfigRecord; ancestors: Ancestor[] }, now: number): CertificateCheck {
  const certFile = asString(entry.node.cert_file);
  const keyFile = asString(entry.node.key_file);
  const label = entry.ancestors[entry.ancestors.length - 1]?.key ?? certFile;
  const server = ancestorText(entry.ancestors.slice(0, -1), "name");
  const configuredDomains = stringList(entry.node.domains);
  let parsed: X509Certificate[] = [];
  let error = "";
  let status: CertificateStatus = "valid";
  const certPath = resolvePath(certFile);
  if (!existsSync(certPath)) {
    status = "missing";
  } else {
    try {
      parsed = readChain(certPath);
    } catch (cause) {
      status = "invalid";
      error = errorMessage(cause, "无法解析证书").replace(/-----BEGIN[\s\S]*?-----END[^\n-]*-----/g, "").slice(0, 180);
    }
  }
  const chain = parsed.map((cert, index) => describeCertificate(cert, index === 0 ? "leaf" : "intermediate", now));
  const leaf = chain[0];
  const names = leaf ? [leaf.commonName, ...leaf.subjectAltNames].filter(Boolean) : [];
  const domains = configuredDomains.map((name) => ({
    name,
    covered: leaf ? names.some((pattern) => covers(pattern, name)) : null,
  }));
  const resolvedMatch = keyMatch(parsed[0], keyFile);
  if (status === "valid" && leaf && leaf.daysRemaining < 0) status = "expired";
  else if (status === "valid" && (resolvedMatch === "mismatch" || resolvedMatch === "missing" || resolvedMatch === "unread")) status = "key";
  else if (status === "valid" && leaf && leaf.daysRemaining <= EXPIRING_DAYS) status = "expiring";
  return {
    id: `${server}:${label}:${certFile}`,
    name: label,
    server,
    minVersion: tlsLabel(ancestorText(entry.ancestors, "min_version")),
    autoTls: ancestorFlag(entry.ancestors, "auto_tls"),
    domains,
    certFile,
    keyFile,
    status,
    keyMatch: resolvedMatch,
    error,
    chain,
  };
}

export function loadCertificates(): CertificateModel {
  const checkedAt = formatInstant(new Date());
  const found: { node: ConfigRecord; ancestors: Ancestor[] }[] = [];
  let error: string | undefined;
  try {
    collect(readConfigValue(SOURCE), "", [], found);
  } catch (cause) {
    error = errorMessage(cause, "无法读取证书配置");
  }
  const frontend = readQuotedField(SOURCE, "pxy_frontend_file") ?? "config/frontend.hamburger";
  try {
    collect(readConfigValue(frontend), "", [], found);
  } catch {
    // 前端文件的解析错误由静态配置页展示，证书页继续使用主配置里的证书。
  }
  return { source: SOURCE, checkedAt, certificates: found.map((entry) => inspect(entry, Date.now())), error };
}
