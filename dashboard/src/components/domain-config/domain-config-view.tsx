"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import { ConfirmDelete, dangerButtonClass } from "@/components/config-editor/confirm-delete";
import { fieldClass } from "@/components/config-editor/fields";
import { ServiceSelect } from "@/components/domain-config/service-select";
import { SaveBar, saveErrorText } from "@/components/config-editor/save-bar";
import { ServiceDialog } from "@/components/config-editor/service-dialog";
import { Panel } from "@/components/common/panel";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { saveDomainConfig } from "@/lib/config-edit/actions";
import { bindingToDraft, blankBinding, blankService, serviceToDraft, type BindingDraft, type RuleDraft, type ServiceDraft } from "@/lib/config-edit/model";
import type { DomainConfigModel, ProxyRule } from "@/lib/domain-config/parse";
import { useAuth } from "@/lib/auth/auth-context";
import { usePreferences, type MessageKey } from "@/lib/preferences/preferences-context";

const filters = ["all", "frontend", "backend", "custom"] as const;
type TypeFilter = (typeof filters)[number];

const filterLabel: Record<TypeFilter, MessageKey> = {
  all: "domainConfig.filterAll",
  frontend: "domainConfig.frontend",
  backend: "domainConfig.backend",
  custom: "domainConfig.custom",
};

const actionClass = "rounded-lg border border-[var(--border)] px-2.5 py-1 text-xs text-secondary hover:text-primary";

function typeLabel(type: string, t: (key: MessageKey) => string): string {
  if (type === "frontend" || type === "backend" || type === "custom") return t(filterLabel[type]);
  return type;
}

function typeClass(type: string): string {
  if (type === "frontend") return "bg-emerald-500/15 text-emerald-300 [html.theme-light_&]:text-emerald-700";
  if (type === "backend") return "bg-sky-500/15 text-sky-300 [html.theme-light_&]:text-sky-700";
  return "bg-amber-500/15 text-amber-300 [html.theme-light_&]:text-amber-700";
}

function TypeBadge({ type, label }: { type: string; label: string }) {
  return <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${typeClass(type)}`}>{label}</span>;
}

function listenAddress(service: ServiceDraft): string {
  if (service.host && service.port) return `${service.host}:${service.port}`;
  return service.host || service.port;
}

function asRule(rule: RuleDraft): ProxyRule {
  const port = Number(rule.proxyPort);
  return {
    api: rule.api,
    service: rule.mode === "service" ? rule.service || undefined : undefined,
    staticRoot: rule.mode === "static" ? rule.staticRoot || undefined : undefined,
    proxyHost: rule.mode === "host" ? rule.proxyHost || undefined : undefined,
    proxyPort: rule.proxyPort.trim() && Number.isFinite(port) ? port : undefined,
    useRewrite: rule.useRewrite,
    rewrite: rule.rewrite || undefined,
  };
}

export function DomainConfigView({ config }: { config: DomainConfigModel }) {
  const { t } = usePreferences();
  const { token } = useAuth();
  const router = useRouter();
  const [keyword, setKeyword] = useState("");
  const [type, setType] = useState<TypeFilter>("all");
  const [bindings, setBindings] = useState(() => config.bindings.map(bindingToDraft));
  const [services, setServices] = useState(() => config.services.map(serviceToDraft));
  const [seen, setSeen] = useState(config.version);
  const [editing, setEditing] = useState<ServiceDraft | null>(null);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState("");
  const [confirm, setConfirm] = useState<{ name: string; run: () => void } | null>(null);
  if (seen !== config.version) {
    setSeen(config.version);
    setBindings(config.bindings.map(bindingToDraft));
    setServices(config.services.map(serviceToDraft));
    setNotice("");
  }
  const servicesByName = useMemo(() => new Map(services.map((service) => [service.name, service])), [services]);
  const query = keyword.trim().toLowerCase();
  const locked = Boolean(config.error);
  const baseline = JSON.stringify({ bindings: config.bindings.map(bindingToDraft), services: config.services.map(serviceToDraft) });
  const dirty = JSON.stringify({ bindings, services }) !== baseline;

  const visibleBindings = useMemo(() => bindings.filter((binding) => {
    const service = servicesByName.get(binding.service);
    if (type !== "all" && (service?.type ?? "") !== type) return false;
    if (!query) return true;
    return binding.domain.toLowerCase().includes(query) || binding.service.toLowerCase().includes(query);
  }), [bindings, query, servicesByName, type]);

  const visibleServices = useMemo(() => services.filter((service) => {
    if (type !== "all" && service.type !== type) return false;
    if (!query) return true;
    const domains = bindings.filter((binding) => binding.service === service.name).map((binding) => binding.domain);
    const proxyText = service.rules.map((rule) => [rule.api, rule.service, rule.staticRoot, rule.proxyHost].filter(Boolean).join(" ")).join(" ");
    return [service.name, service.domain, listenAddress(service), proxyText, ...domains].join(" ").toLowerCase().includes(query);
  }), [bindings, query, services, type]);

  const counts = {
    frontend: services.filter((service) => service.type === "frontend").length,
    backend: services.filter((service) => service.type === "backend").length,
    custom: services.filter((service) => service.type === "custom").length,
  };

  const updateBinding = (key: string, patch: Partial<BindingDraft>) => {
    setBindings((current) => current.map((item) => item.key === key ? { ...item, ...patch } : item));
  };

  const askDelete = (name: string, run: () => void) => setConfirm({ name: name || "—", run });

  const removeService = (service: ServiceDraft) => {
    askDelete(service.name || service.originalName, () => setServices((current) => current.filter((item) => item.key !== service.key)));
  };

  const save = async () => {
    setSaving(true);
    setNotice("");
    try {
      const result = await saveDomainConfig(token ?? "", config.version, bindings, services);
      if (!result.ok) {
        setNotice(saveErrorText(t, result.error));
        return;
      }
      router.refresh();
    } finally {
      setSaving(false);
    }
  };

  const reset = () => {
    setBindings(config.bindings.map(bindingToDraft));
    setServices(config.services.map(serviceToDraft));
    setNotice("");
  };

  return (
    <DashboardShell title={t("domainConfig.title")} subtitle={t("domainConfig.subtitle")}>
      <Panel className="space-y-4">
        <div>
          <p className="text-secondary text-xs">{t("domainConfig.source")} {config.source}</p>
          {config.runtimeMap && config.runtimeMap !== config.source ? (
            <p className="text-tertiary mt-1 text-xs">{t("domainConfig.runtimeMap", { file: config.runtimeMap })}</p>
          ) : null}
        </div>
        {config.error ? <p className="rounded-xl border border-rose-400/40 bg-rose-500/10 px-3 py-2 text-sm text-rose-300 [html.theme-light_&]:text-rose-700">{config.error}</p> : null}
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <Stat label={t("domainConfig.mappings")} value={bindings.length} />
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

      <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1.15fr)_minmax(0,0.85fr)]">
        <Panel className="overflow-hidden p-0">
          <div className="flex items-center justify-between gap-3 px-4 py-3">
            <h2 className="text-lg font-semibold">{t("domainConfig.mappings")}</h2>
            <div className="flex items-center gap-3">
              <span className="text-tertiary text-xs">{visibleBindings.length}</span>
              {locked ? null : <button type="button" onClick={() => setBindings((current) => [...current, blankBinding(services[0]?.name ?? "")])} className="rounded-xl bg-emerald-500 px-3 py-1.5 text-sm font-semibold text-slate-950">{t("editor.addMapping")}</button>}
            </div>
          </div>
          {visibleBindings.length === 0 ? (
            <p className="text-secondary px-4 pb-4 text-sm">{t("domainConfig.empty")}</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[560px] border-collapse text-left text-sm">
                <thead>
                  <tr className="text-tertiary border-t border-[var(--border)] text-xs">
                    <th className="px-4 py-2 font-medium">{t("domainConfig.domain")}</th>
                    <th className="px-4 py-2 font-medium">{t("domainConfig.service")}</th>
                    <th className="px-4 py-2 font-medium">{t("domainConfig.type")}</th>
                    <th className="px-4 py-2 font-medium">{t("editor.actions")}</th>
                  </tr>
                </thead>
                <tbody>
                  {visibleBindings.map((binding) => {
                    const service = servicesByName.get(binding.service);
                    const serviceType = service?.type ?? "custom";
                    const choices = services
                      .filter((item) => item.name)
                      .map((item) => ({ name: item.name, type: item.type }));
                    if (binding.service && !choices.some((item) => item.name === binding.service)) {
                      choices.unshift({ name: binding.service, type: serviceType });
                    }
                    return (
                      <tr key={binding.key} className="border-t border-[var(--border)]">
                        <td className="px-4 py-2.5">
                          {locked ? <span className="text-primary font-medium">{binding.domain}</span> : (
                            <input value={binding.domain} onChange={(event) => updateBinding(binding.key, { domain: event.target.value })} className={`${fieldClass} font-mono`} />
                          )}
                        </td>
                        <td className="px-4 py-2.5">
                          {locked ? <span className="text-secondary">{binding.service}</span> : (
                            <ServiceSelect
                              value={binding.service}
                              options={choices}
                              placeholder={t("domainConfig.service")}
                              searchPlaceholder={t("editor.findService")}
                              emptyText={t("services.empty")}
                              typeLabel={(type) => typeLabel(type, t)}
                              onChange={(service) => updateBinding(binding.key, { service })}
                            />
                          )}
                        </td>
                        <td className="px-4 py-2.5">
                          <TypeBadge type={serviceType} label={service ? typeLabel(serviceType, t) : t("domainConfig.undefined")} />
                        </td>
                        <td className="px-4 py-2.5">
                          {locked ? null : <button type="button" onClick={() => askDelete(binding.domain, () => setBindings((current) => current.filter((item) => item.key !== binding.key)))} className={dangerButtonClass}>{t("editor.delete")}</button>}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </Panel>

        <Panel className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-lg font-semibold">{t("domainConfig.services")}</h2>
            <div className="flex items-center gap-3">
              <span className="text-tertiary text-xs">{visibleServices.length}</span>
              {locked ? null : <button type="button" onClick={() => setEditing(blankService())} className="rounded-xl bg-emerald-500 px-3 py-1.5 text-sm font-semibold text-slate-950">{t("editor.addService")}</button>}
            </div>
          </div>
          {visibleServices.length === 0 ? <p className="text-secondary text-sm">{t("domainConfig.empty")}</p> : null}
          <div className="space-y-2">
            {visibleServices.map((service) => {
              const domains = bindings.filter((binding) => binding.service === service.name).map((binding) => binding.domain);
              const listen = listenAddress(service);
              return (
                <article key={service.key} className="panel-strong rounded-xl border border-[var(--border)] px-3 py-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <h3 className="truncate text-sm font-semibold">{service.name || t("domainConfig.unbound")}</h3>
                      <p className="text-tertiary mt-1 break-all text-xs">{domains.length > 0 ? domains.join("、") : service.domain || t("domainConfig.unbound")}</p>
                    </div>
                    <TypeBadge type={service.type} label={typeLabel(service.type, t)} />
                  </div>
                  {listen ? <p className="text-secondary mt-2 font-mono text-xs">{listen}</p> : null}
                  {service.rules.length > 0 ? (
                    <ul className="mt-2 space-y-1">
                      {service.rules.map((rule, index) => {
                        const item = asRule(rule);
                        return (
                          <li key={`${service.key}-${rule.api}-${index}`} className="text-secondary flex flex-wrap items-baseline gap-x-2 text-xs">
                            <span className="text-primary font-mono">{rule.api || t("domainConfig.proxy")}</span>
                            <span className="break-all">
                              {item.staticRoot
                                ? `${t("domainConfig.static")} ${item.staticRoot}`
                                : item.service
                                  ? `${t("domainConfig.forward")} ${item.service}`
                                  : item.proxyHost
                                    ? `${t("domainConfig.forward")} ${item.proxyHost}${item.proxyPort ? `:${item.proxyPort}` : ""}`
                                    : t("domainConfig.noProxy")}
                              {item.useRewrite ? ` · ${t("domainConfig.rewrite")}` : ""}
                              {item.rewrite ? ` ${item.rewrite}` : ""}
                            </span>
                          </li>
                        );
                      })}
                    </ul>
                  ) : null}
                  {locked ? null : (
                    <div className="mt-3 flex gap-2">
                      <button type="button" onClick={() => setEditing(service)} className={actionClass}>{t("editor.edit")}</button>
                      <button type="button" onClick={() => removeService(service)} className={dangerButtonClass}>{t("editor.delete")}</button>
                    </div>
                  )}
                </article>
              );
            })}
          </div>
        </Panel>
      </div>
      <SaveBar dirty={dirty && !locked} saving={saving} notice={notice || (config.runtimeMap && config.runtimeMap !== config.source ? t("domainConfig.runtimeMap", { file: config.runtimeMap }) : "")} onDiscard={reset} onSave={() => void save()} />
      {editing ? (
        <ServiceDialog
          draft={editing}
          onClose={() => setEditing(null)}
          onSubmit={(next) => {
            setServices((current) => current.some((item) => item.key === next.key) ? current.map((item) => item.key === next.key ? next : item) : [...current, next]);
            setEditing(null);
          }}
        />
      ) : null}
      {confirm ? <ConfirmDelete name={confirm.name} onClose={() => setConfirm(null)} onConfirm={() => { confirm.run(); setConfirm(null); }} /> : null}
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
