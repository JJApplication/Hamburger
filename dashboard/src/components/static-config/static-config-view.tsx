"use client";

import { useMemo, useState } from "react";

import { Panel } from "@/components/common/panel";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import type { StaticConfigModel, StaticSite } from "@/lib/static-config/load";
import { usePreferences, type MessageKey } from "@/lib/preferences/preferences-context";

const filters = ["all", "alias", "backend"] as const;
type SiteFilter = (typeof filters)[number];

const filterLabel: Record<SiteFilter, MessageKey> = {
  all: "staticConfig.filterAll",
  alias: "staticConfig.filterAlias",
  backend: "staticConfig.filterBackend",
};

function siteText(site: StaticSite): string {
  const alias = site.alias.map((item) => `${item.path} ${item.root}`).join(" ");
  const backends = site.backends.map((item) => `${item.api} ${item.service ?? ""} ${item.staticRoot ?? ""}`).join(" ");
  return [site.name, site.root, site.index, site.tryFile, alias, backends].join(" ");
}

export function StaticConfigView({ config }: { config: StaticConfigModel }) {
  const { t } = usePreferences();
  const [keyword, setKeyword] = useState("");
  const [filter, setFilter] = useState<SiteFilter>("all");
  const query = keyword.trim().toLowerCase();
  const listen = [config.host, config.port].filter(Boolean).join(":");

  const visible = useMemo(() => config.sites.filter((site) => {
    if (filter === "alias" && site.alias.length === 0) return false;
    if (filter === "backend" && site.backends.length === 0) return false;
    if (!query) return true;
    return siteText(site).toLowerCase().includes(query);
  }), [config.sites, filter, query]);

  const counts = {
    alias: config.sites.filter((site) => site.alias.length > 0).length,
    backend: config.sites.filter((site) => site.backends.length > 0).length,
  };

  return (
    <DashboardShell title={t("staticConfig.title")} subtitle={t("staticConfig.subtitle")}>
      <Panel className="space-y-4">
        <div>
          <p className="text-secondary text-xs">{t("domainConfig.source")} {config.source}</p>
          {config.configuredFile && config.configuredFile !== config.source ? (
            <p className="text-tertiary mt-1 text-xs">{t("staticConfig.runtimeFile", { file: config.configuredFile })}</p>
          ) : null}
        </div>
        {config.error ? <p className="rounded-xl border border-rose-400/40 bg-rose-500/10 px-3 py-2 text-sm text-rose-300 [html.theme-light_&]:text-rose-700">{config.error}</p> : null}
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <Stat label={t("staticConfig.sites")} value={config.sites.length} />
          <Stat label={t("staticConfig.aliasSites")} value={counts.alias} />
          <Stat label={t("staticConfig.backendSites")} value={counts.backend} />
          <Stat label={t("staticConfig.cache")} value={config.cacheEnabled ? t("staticConfig.on") : t("staticConfig.off")} />
        </div>
        <dl className="grid gap-3 md:grid-cols-2">
          <Fact label={t("staticConfig.listen")} value={listen} />
          <Fact label={t("staticConfig.balancer")} value={config.balancer} />
          <Fact label={t("staticConfig.cacheDir")} value={config.cacheDir} />
          <Fact label={t("staticConfig.expire")} value={config.cacheExpire === "0" ? t("staticConfig.neverExpire") : config.cacheExpire} />
          <Fact
            label={t("staticConfig.memory")}
            value={config.memoryEnabled ? t("staticConfig.memoryDetail", {
              max: config.memoryMaxMb,
              file: config.memoryMaxFileMb,
              entries: config.memoryMaxEntries,
              seconds: config.memoryRevalidateSeconds,
            }) : t("staticConfig.off")}
          />
          <Fact label={t("staticConfig.fastConnect")} value={`${config.fastConnect ? t("staticConfig.on") : t("staticConfig.off")} · ${t("staticConfig.http3")} ${config.http3 ? t("staticConfig.on") : t("staticConfig.off")}`} />
        </dl>
        {config.matchers.length > 0 ? (
          <div>
            <p className="text-tertiary text-xs">{t("staticConfig.matchers")}</p>
            <div className="mt-2 flex flex-wrap gap-1.5">
              {config.matchers.map((matcher) => (
                <span key={matcher} className="panel-strong text-secondary rounded-full px-2 py-0.5 font-mono text-xs">{matcher}</span>
              ))}
            </div>
          </div>
        ) : null}
        {config.headers.length > 0 ? (
          <div>
            <p className="text-tertiary text-xs">{t("staticConfig.headers")}</p>
            <ul className="mt-2 space-y-1">
              {config.headers.map((header) => (
                <li key={header.name} className="text-secondary font-mono text-xs">{header.name}: {header.value}</li>
              ))}
            </ul>
          </div>
        ) : null}
      </Panel>

      <Panel className="flex flex-col gap-3 lg:flex-row lg:items-center">
        <input
          value={keyword}
          onChange={(event) => setKeyword(event.target.value)}
          placeholder={t("staticConfig.search")}
          className="panel-strong text-primary placeholder:text-tertiary w-full rounded-xl border border-[var(--border)] px-3 py-2 text-sm outline-none focus:border-emerald-400/70 lg:max-w-sm"
        />
        <div className="flex flex-wrap gap-2" role="group" aria-label={t("staticConfig.sites")}>
          {filters.map((item) => {
            const selected = filter === item;
            return (
              <button
                key={item}
                type="button"
                aria-pressed={selected}
                onClick={() => setFilter(item)}
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

      {visible.length === 0 ? <Panel><p className="text-secondary text-sm">{t("staticConfig.empty")}</p></Panel> : (
        <div className="grid items-start gap-3 xl:grid-cols-2">
          {visible.map((site) => (
            <article key={site.name} className="panel rounded-2xl border p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <h2 className="truncate text-base font-semibold">{site.name}</h2>
                  <p className="text-secondary mt-1 break-all font-mono text-xs">{site.root}</p>
                </div>
                {site.type ? <span className="text-tertiary shrink-0 text-xs">{site.type}</span> : null}
              </div>
              <p className="text-tertiary mt-3 text-xs">
                {t("staticConfig.index")} {site.index || "—"} · {t("staticConfig.tryFile")} {site.tryFile || "—"}
              </p>
              {site.access || site.compress ? (
                <div className="mt-2 flex flex-wrap gap-1.5">
                  {site.access ? <span className="rounded-full bg-sky-500/15 px-2 py-0.5 text-xs text-sky-300 [html.theme-light_&]:text-sky-700">{t("staticConfig.access")}</span> : null}
                  {site.compress ? <span className="rounded-full bg-sky-500/15 px-2 py-0.5 text-xs text-sky-300 [html.theme-light_&]:text-sky-700">{t("staticConfig.compress")}</span> : null}
                </div>
              ) : null}
              {site.alias.length > 0 ? (
                <ul className="mt-3 space-y-1">
                  {site.alias.map((item) => (
                    <li key={item.path} className="text-secondary break-all text-xs">
                      <span className="text-tertiary">{t("staticConfig.alias")} </span>
                      <span className="text-primary font-mono">{item.path}</span>
                      <span> → {item.root}</span>
                    </li>
                  ))}
                </ul>
              ) : null}
              {site.backends.length > 0 ? (
                <ul className="mt-2 space-y-1">
                  {site.backends.map((item) => (
                    <li key={`${site.name}-${item.api}-${item.service ?? item.staticRoot ?? "route"}`} className="text-secondary break-all text-xs">
                      <span className="text-tertiary">{t("staticConfig.forward")} </span>
                      <span className="text-primary font-mono">{item.api || "/"}</span>
                      <span> → {item.staticRoot ?? item.service ?? ""}</span>
                      {item.useRewrite ? ` · ${t("domainConfig.rewrite")}` : ""}
                      {item.rewrite ? ` ${item.rewrite}` : ""}
                    </li>
                  ))}
                </ul>
              ) : null}
            </article>
          ))}
        </div>
      )}
    </DashboardShell>
  );
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="panel-strong rounded-xl px-3 py-3">
      <p className="text-tertiary text-xs">{label}</p>
      <p className="mt-1 text-2xl font-semibold tabular-nums">{value}</p>
    </div>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="panel-strong rounded-xl px-3 py-2">
      <dt className="text-tertiary text-xs">{label}</dt>
      <dd className="text-primary mt-1 break-all font-mono text-xs">{value || "—"}</dd>
    </div>
  );
}
