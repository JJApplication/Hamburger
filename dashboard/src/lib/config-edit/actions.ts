"use server";

import { fetchCurrentUser } from "@/lib/api/gateway";
import { applyBindings, applyServices, applyStatic, commitEditedFile, ConfigEditError, DOMAIN_FILE, SERVICE_FILE, STATIC_FILE } from "@/lib/config-edit/commit";
import type { BindingDraft, SaveResult, ServiceDraft, StaticDraft } from "@/lib/config-edit/model";
import { commitMiddleware } from "@/lib/middleware-config/commit";
import type { ErrorConfigDraft, MiddlewareDraft } from "@/lib/middleware-config/model";
import { commitSecurity } from "@/lib/security-config/commit";
import type { SecurityDraft } from "@/lib/security-config/model";

async function authorize(token: string) {
  if (!token.trim()) throw new ConfigEditError({ code: "auth" });
  try {
    await fetchCurrentUser(token);
  } catch {
    throw new ConfigEditError({ code: "auth" });
  }
}

async function run(token: string, work: () => string): Promise<SaveResult> {
  try {
    await authorize(token);
    return { ok: true, version: work() };
  } catch (error) {
    if (error instanceof ConfigEditError) return { ok: false, error: error.failure };
    return { ok: false, error: { code: "failed", message: error instanceof Error ? error.message : "保存失败" } };
  }
}

export async function saveServiceConfig(token: string, version: string, services: ServiceDraft[]): Promise<SaveResult> {
  return run(token, () => commitEditedFile(SERVICE_FILE, version, (root) => applyServices(root, services)));
}

export async function saveDomainConfig(token: string, version: string, bindings: BindingDraft[], services: ServiceDraft[]): Promise<SaveResult> {
  return run(token, () => commitEditedFile(DOMAIN_FILE, version, (root) => {
    applyBindings(root, bindings);
    applyServices(root, services);
  }));
}

export async function saveStaticConfig(token: string, version: string, draft: StaticDraft): Promise<SaveResult> {
  return run(token, () => commitEditedFile(STATIC_FILE, version, (root) => applyStatic(root, draft)));
}

export async function saveMiddlewareConfig(token: string, version: string, draft: MiddlewareDraft, errorConfig: ErrorConfigDraft): Promise<SaveResult> {
  return run(token, () => commitMiddleware(version, draft, errorConfig));
}

export async function saveSecurityConfig(token: string, version: string, draft: SecurityDraft): Promise<SaveResult> {
  return run(token, () => commitSecurity(version, draft));
}
