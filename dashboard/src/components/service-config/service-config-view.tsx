"use client";

import { useMemo, useState } from "react";

import { Panel } from "@/components/common/panel";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import type { DomainConfigModel, DomainServiceDef, ProxyRule } from "@/lib/domain-config/parse";
import { usePreferences, type MessageKey } from "@/lib/preferences/preferences-context";

const filters = ["all", "frontend", "backend", "custom"] as const;
type TypeFilter = (typeof filters)[number];

const filterLabel: Record<TypeFilter, MessageKey> = {
  all: "domainConfig.filterAll",
  frontend: "domainConfig.frontend",
  backend: "domainConfig.backend",
  custom: "domainConfig.custom",
};

function typeLabel(type: string, t: (key: MessageKey) => string): string {
  if (type === "frontend" || type === "backend" || type === "custom") return t(filterLabel[type]);
  return type;
}

function typeClass(type: string): string {
  if (type === "frontend") return "bg-emerald-500/15 text-emerald-300 [html.theme-light_&]:text-emerald-700";
  if (type === "backend") return "bg-sky-500/15 text-sky-300 [html.theme-light_&]:text-sky-700";
  return "bg-amber-500/15 text-amber-300 [html.theme-light_&]:text-amber-700";
}

function listenAddress(service: DomainServiceDef): string {
  if (service.host && service.port) return `${service.host}:${service.port}`;
  return service.host ?? (service.port ? String(service.port) : "");
}

function routeText(rule: ProxyRule, t: (key: MessageKey) => string): string {
  const target = rule.staticRoot
    ? `${t("domainConfig.static")} ${rule.staticRoot}`
    : rule.service
      ? `${t("domainConfig.forward")} ${rule.service}`
      : rule.proxyHost
        ? `${t("domainConfig.forward")} ${rule.proxyHost}${rule.proxyPort ? `:${rule.proxyPort}` : ""}`
        : t("domainConfig.noProxy");
  return `${target}${rule.useRewrite ? ` · ${t("domainConfig.rewrite")}` : ""}${rule.rewrite ? ` ${rule.rewrite}` : ""}`;
}

export function ServiceConfigView({ config }: { config: DomainConfigModel }) {
  const { t } = usePreferences();
  const [keyword, setKeyword] = useState("");
  const [type, setType] = useState<TypeFilter>("all");
  const query = keyword.trim().toLowerCase();

  const visible = useMemo(() => config.services.filter((service) => {
    if (type !== "all" && service.type !== type) return false;
    if (!query) return true;
    const routes = service.proxyPass.map((rule) => [rule.api, rule.service, rule.staticRoot, rule.proxyHost].filter(Boolean).join(" ")).join(" ");
    return [service.name, service.domain, service.type, listenAddress(service), routes].join(" ").toLowerCase().includes(query);
  }), [config.services, query, type]);

  const counts = {
    frontend: config.services.filter((service) => service.type === "frontend").length,
    backend: config.services.filter((service) => service.type === "backend").length,
    custom: config.services.filter((service) => service.type === "custom").length,
  };

  return (
    <DashboardShell title={t("services.title")} subtitle={t("services.subtitle")}>
      <Panel className="space-y-4">
        <div>
          <p className="text-secondary text-xs">{t("domainConfig.source")} {config.source}</p>
          {config.runtimeMap && config.runtimeMap !== config.source ? (
            <p className="text-tertiary mt-1 text-xs">{t("services.runtimeMap", { file: config.runtimeMap })}</p>
          ) : null}
        </div>
        {config.error ? <p className="rounded-xl border border-rose-400/40 bg-rose-500/10 px-3 py-2 text-sm text-rose-300 [html.theme-light_&]:text-rose-700">{config.error}</p> : null}
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <Stat label={t("services.total")} value={config.services.length} />
          <Stat label={t("domainConfig.frontend")} value={counts.frontend} />
          <Stat label={t("domainConfig.backend")} value={counts.backend} />
          <Stat label={t("domainConfig.custom")} value={counts.custom} />
        </div>
      </Panel>

      <Panel className="flex flex-col gap-3 lg:flex-row lg:items-center">
        <input
          value={keyword}
          onChange={(event) => setKeyword(event.target.value)}
          placeholder={t("domainConfig.search")}
          className="panel-strong text-primary placeholder:text-tertiary w-full rounded-xl border border-[var(--border)] px-3 py-2 text-sm outline-none focus:border-emerald-400/70 lg:max-w-sm"
        />
        <div className="flex flex-wrap gap-2" role="group" aria-label={t("domainConfig.type")}>
          {filters.map((item) => {
            const selected = type === item;
            return (
              <button
                key={item}
                type="button"
                aria-pressed={selected}
                onClick={() => setType(item)}
                className={[
                  "rounded-full border px-3 py-1.5 text-sm transition",
                  selected
                    ? "border-emerald-400/50 bg-emerald-500/15 text-emerald-300 [html.theme-light_&]:text-emerald-700"
                    : "text-secondary border-[var(--border)] hover:panel-soft hover:text-primary",
                ].join(" ")}
              >
                {t(filterLabel[item])}
              </button>
            );
          })}
        </div>
      </Panel>

      <Panel className="shrink-0 overflow-hidden p-0">
        <div className="flex items-center justify-between px-4 py-3">
          <h2 className="text-lg font-semibold">{t("services.total")}</h2>
          <span className="text-tertiary text-xs">{visible.length}</span>
        </div>
        {visible.length === 0 ? (
          <p className="text-secondary px-4 pb-4 text-sm">{t("services.empty")}</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[640px] border-collapse text-left text-sm">
              <thead>
                <tr className="text-tertiary border-t border-[var(--border)] text-xs">
                  <th className="px-4 py-2 font-medium">{t("domainConfig.service")}</th>
                  <th className="px-4 py-2 font-medium">{t("domainConfig.type")}</th>
                  <th className="px-4 py-2 font-medium">{t("domainConfig.domain")}</th>
                </tr>
              </thead>
              <tbody>
                {visible.map((service) => {
                  const listen = listenAddress(service);
                  return (
                    <tr key={service.name} className="border-t border-[var(--border)] align-top">
                      <td className="px-4 py-2.5">
                        <p className="text-primary font-medium">{service.name}</p>
                        {listen ? <p className="text-tertiary mt-1 font-mono text-xs">{t("services.listen")} {listen}</p> : null}
                      </td>
                      <td className="px-4 py-2.5">
                        <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${typeClass(service.type)}`}>{typeLabel(service.type, t)}</span>
                        {service.preCheck ? <span className="ml-2 inline-flex rounded-full bg-violet-500/15 px-2 py-0.5 text-xs font-medium text-violet-300 [html.theme-light_&]:text-violet-700">{t("services.precheck")}</span> : null}
                      </td>
                      <td className="px-4 py-2.5">
                        <p className="text-secondary break-all font-mono text-xs">{service.domain || t("domainConfig.unbound")}</p>
                        {service.proxyPass.length > 0 ? (
                          <ul className="mt-2 space-y-1">
                            {service.proxyPass.map((rule) => (
                              <li key={`${service.name}-${rule.api}-${rule.service ?? rule.staticRoot ?? "rule"}`} className="text-secondary flex flex-wrap items-baseline gap-x-2 text-xs">
                                <span className="text-primary font-mono">{rule.api || t("domainConfig.proxy")}</span>
                                <span className="break-all">{routeText(rule, t)}</span>
                              </li>
                            ))}
                          </ul>
                        ) : null}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </Panel>
    </DashboardShell>
  );
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className="panel-strong rounded-xl px-3 py-3">
      <p className="text-tertiary text-xs">{label}</p>
      <p className="mt-1 text-2xl font-semibold tabular-nums">{value}</p>
    </div>
  );
}
