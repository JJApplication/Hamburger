"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import { ConfirmDelete, dangerButtonClass } from "@/components/config-editor/confirm-delete";
import { fieldClass } from "@/components/config-editor/fields";
import { SaveBar, saveErrorText } from "@/components/config-editor/save-bar";
import { SiteDialog } from "@/components/config-editor/site-dialog";
import { Panel } from "@/components/common/panel";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { saveStaticConfig } from "@/lib/config-edit/actions";
import { blankSite, siteToDraft, staticToDraft, type SiteDraft, type StaticDraft } from "@/lib/config-edit/model";
import type { StaticConfigModel } from "@/lib/static-config/load";
import { useAuth } from "@/lib/auth/auth-context";
import { usePreferences, type MessageKey } from "@/lib/preferences/preferences-context";

const filters = ["all", "alias", "backend"] as const;
type SiteFilter = (typeof filters)[number];

const filterLabel: Record<SiteFilter, MessageKey> = {
  all: "staticConfig.filterAll",
  alias: "staticConfig.filterAlias",
  backend: "staticConfig.filterBackend",
};

const actionClass = "rounded-lg border border-[var(--border)] px-2.5 py-1 text-xs text-secondary hover:text-primary";

function siteText(site: SiteDraft): string {
  const alias = site.alias.map((item) => `${item.path} ${item.root}`).join(" ");
  const backends = site.backends.map((item) => `${item.api} ${item.service} ${item.staticRoot}`).join(" ");
  return [site.name, site.root, site.index, site.tryFile, alias, backends].join(" ");
}

export function StaticConfigView({ config }: { config: StaticConfigModel }) {
  const { t } = usePreferences();
  const { token } = useAuth();
  const router = useRouter();
  const [keyword, setKeyword] = useState("");
  const [filter, setFilter] = useState<SiteFilter>("all");
  const [matcher, setMatcher] = useState("");
  const [draft, setDraft] = useState(() => staticToDraft(config));
  const [seen, setSeen] = useState(config.version);
  const [editing, setEditing] = useState<SiteDraft | null>(null);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState("");
  const [confirm, setConfirm] = useState<{ name: string; run: () => void } | null>(null);
  if (seen !== config.version) {
    setSeen(config.version);
    setDraft(staticToDraft(config));
    setNotice("");
  }
  const query = keyword.trim().toLowerCase();
  const locked = Boolean(config.error);
  const dirty = JSON.stringify(draft) !== JSON.stringify(staticToDraft(config));
  const listen = [draft.host, draft.port].filter(Boolean).join(":");

  const visible = useMemo(() => draft.sites.filter((site) => {
    if (filter === "alias" && site.alias.length === 0) return false;
    if (filter === "backend" && site.backends.length === 0) return false;
    if (!query) return true;
    return siteText(site).toLowerCase().includes(query);
  }), [draft.sites, filter, query]);

  const counts = {
    alias: draft.sites.filter((site) => site.alias.length > 0).length,
    backend: draft.sites.filter((site) => site.backends.length > 0).length,
  };

  const patch = (partial: Partial<StaticDraft>) => setDraft((current) => ({ ...current, ...partial }));

  const addMatcher = () => {
    const next = matcher.trim();
    if (!next) return;
    patch({ matchers: [...draft.matchers, next] });
    setMatcher("");
  };

  const removeSite = (site: SiteDraft) => {
    setConfirm({
      name: site.name || site.originalName || "—",
      run: () => patch({ sites: draft.sites.filter((item) => item.key !== site.key) }),
    });
  };

  const save = async () => {
    setSaving(true);
    setNotice("");
    try {
      const result = await saveStaticConfig(token ?? "", config.version, draft);
      if (!result.ok) {
        setNotice(saveErrorText(t, result.error));
        return;
      }
      router.refresh();
    } finally {
      setSaving(false);
    }
  };

  return (
    <DashboardShell title={t("staticConfig.title")} subtitle={t("staticConfig.subtitle")}>
      <Panel className="space-y-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="text-secondary text-xs">{t("domainConfig.source")} {config.source}</p>
            {config.configuredFile && config.configuredFile !== config.source ? (
              <p className="text-tertiary mt-1 text-xs">{t("staticConfig.runtimeFile", { file: config.configuredFile })}</p>
            ) : null}
          </div>
        </div>
        {config.error ? <p className="rounded-xl border border-rose-400/40 bg-rose-500/10 px-3 py-2 text-sm text-rose-300 [html.theme-light_&]:text-rose-700">{config.error}</p> : null}
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <Stat label={t("staticConfig.sites")} value={draft.sites.length} />
          <Stat label={t("staticConfig.aliasSites")} value={counts.alias} />
          <Stat label={t("staticConfig.backendSites")} value={counts.backend} />
          <Stat label={t("staticConfig.cache")} value={draft.cacheEnabled ? t("staticConfig.on") : t("staticConfig.off")} />
        </div>
        <div className="grid gap-3 md:grid-cols-2">
          <label className="text-secondary block text-sm">{t("staticConfig.listen")}
            <span className="mt-1 grid grid-cols-[1fr_8rem] gap-2">
              <input disabled={locked} value={draft.host} onChange={(event) => patch({ host: event.target.value })} className={`${fieldClass} font-mono`} />
              <input disabled={locked} value={draft.port} onChange={(event) => patch({ port: event.target.value })} className={fieldClass} />
            </span>
          </label>
          <label className="text-secondary block text-sm">{t("staticConfig.balancer")}
            <input disabled={locked} value={draft.balancer} onChange={(event) => patch({ balancer: event.target.value })} className={`${fieldClass} mt-1 font-mono`} />
          </label>
          <label className="text-secondary block text-sm">{t("staticConfig.cacheDir")}
            <input disabled={locked} value={draft.cacheDir} onChange={(event) => patch({ cacheDir: event.target.value })} className={`${fieldClass} mt-1 font-mono`} />
          </label>
          <label className="text-secondary block text-sm">{t("staticConfig.expire")}
            <input disabled={locked} value={draft.cacheExpire} onChange={(event) => patch({ cacheExpire: event.target.value })} className={`${fieldClass} mt-1`} />
          </label>
          <label className="text-secondary block text-sm">{t("staticConfig.memory")}
            <span className="mt-1 grid grid-cols-2 gap-2">
              <input disabled={locked} value={draft.memoryMaxMb} onChange={(event) => patch({ memoryMaxMb: event.target.value })} placeholder="MB" className={fieldClass} />
              <input disabled={locked} value={draft.memoryMaxFileMb} onChange={(event) => patch({ memoryMaxFileMb: event.target.value })} placeholder="MB" className={fieldClass} />
              <input disabled={locked} value={draft.memoryMaxEntries} onChange={(event) => patch({ memoryMaxEntries: event.target.value })} className={fieldClass} />
              <input disabled={locked} value={draft.memoryRevalidateSeconds} onChange={(event) => patch({ memoryRevalidateSeconds: event.target.value })} className={fieldClass} />
            </span>
          </label>
          <div className="grid content-start gap-2">
            <Toggle label={t("staticConfig.cache")} checked={draft.cacheEnabled} disabled={locked} onChange={(cacheEnabled) => patch({ cacheEnabled })} />
            <Toggle label={t("staticConfig.memory")} checked={draft.memoryEnabled} disabled={locked} onChange={(memoryEnabled) => patch({ memoryEnabled })} />
            <Toggle label={t("staticConfig.fastConnect")} checked={draft.fastConnect} disabled={locked} onChange={(fastConnect) => patch({ fastConnect })} />
            <Toggle label={t("staticConfig.http3")} checked={draft.http3} disabled={locked} onChange={(http3) => patch({ http3 })} />
          </div>
        </div>
        <p className="text-tertiary text-xs">{listen || "—"} · {draft.cacheExpire === "0" ? t("staticConfig.neverExpire") : draft.cacheExpire || "—"} · {t("staticConfig.memoryDetail", { max: draft.memoryMaxMb || "—", file: draft.memoryMaxFileMb || "—", entries: draft.memoryMaxEntries || "—", seconds: draft.memoryRevalidateSeconds || "—" })}</p>
        <div>
          <p className="text-tertiary text-xs">{t("staticConfig.matchers")}</p>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {draft.matchers.map((item, index) => (
              <span key={`${item}-${index}`} className="panel-strong text-secondary inline-flex items-center gap-1 rounded-full px-2 py-0.5 font-mono text-xs">
                {item}
                {locked ? null : <button type="button" onClick={() => patch({ matchers: draft.matchers.filter((_, itemIndex) => itemIndex !== index) })} className="text-tertiary hover:text-rose-300" aria-label={t("editor.remove")}>×</button>}
              </span>
            ))}
          </div>
          {locked ? null : (
            <div className="mt-2 flex gap-2">
              <input value={matcher} onChange={(event) => setMatcher(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") { event.preventDefault(); addMatcher(); } }} placeholder={t("editor.addMatcher")} className={`${fieldClass} max-w-xs font-mono`} />
              <button type="button" onClick={addMatcher} className={actionClass}>{t("editor.add")}</button>
            </div>
          )}
        </div>
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <p className="text-tertiary text-xs">{t("staticConfig.headers")}</p>
            {locked ? null : <button type="button" onClick={() => patch({ headers: [...draft.headers, { key: `header-${draft.headers.length}-${Date.now()}`, originalName: "", name: "", value: "" }] })} className="text-sm text-emerald-300 [html.theme-light_&]:text-emerald-700">{t("editor.addHeader")}</button>}
          </div>
          {draft.headers.map((header, index) => (
            <div key={header.key} className="grid gap-2 md:grid-cols-[1fr_1fr_auto]">
              <input disabled={locked} value={header.name} onChange={(event) => patch({ headers: draft.headers.map((item, itemIndex) => itemIndex === index ? { ...item, name: event.target.value } : item) })} placeholder={t("editor.headerName")} className={`${fieldClass} font-mono`} />
              <input disabled={locked} value={header.value} onChange={(event) => patch({ headers: draft.headers.map((item, itemIndex) => itemIndex === index ? { ...item, value: event.target.value } : item) })} placeholder={t("editor.headerValue")} className={`${fieldClass} font-mono`} />
              {locked ? <span /> : <button type="button" onClick={() => patch({ headers: draft.headers.filter((item) => item.key !== header.key) })} className="text-secondary px-2 text-xs hover:text-rose-300">{t("editor.remove")}</button>}
            </div>
          ))}
        </div>
      </Panel>

      <Panel className="flex flex-col gap-3 lg:flex-row lg:items-center">
        <input
          value={keyword}
          onChange={(event) => setKeyword(event.target.value)}
          placeholder={t("staticConfig.search")}
          className="panel-strong text-primary placeholder:text-tertiary w-full rounded-xl border border-[var(--border)] px-3 py-2 text-sm outline-none focus:border-emerald-400/70 lg:max-w-sm"
        />
        <div className="flex flex-1 flex-wrap items-center gap-2" role="group" aria-label={t("staticConfig.sites")}>
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
          {locked ? null : <button type="button" onClick={() => setEditing(blankSite())} className="ml-auto rounded-xl bg-emerald-500 px-3 py-1.5 text-sm font-semibold text-slate-950">{t("editor.addSite")}</button>}
        </div>
      </Panel>

      {visible.length === 0 ? <Panel><p className="text-secondary text-sm">{t("staticConfig.empty")}</p></Panel> : (
        <div className="grid items-start gap-3 xl:grid-cols-2">
          {visible.map((site) => (
            <article key={site.key} className="panel rounded-2xl border p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <h2 className="truncate text-base font-semibold">{site.name || t("domainConfig.unbound")}</h2>
                  <p className="text-secondary mt-1 break-all font-mono text-xs">{site.root || "—"}</p>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  {site.type ? <span className="text-tertiary text-xs">{site.type}</span> : null}
                  {locked ? null : (
                    <>
                      <button type="button" onClick={() => setEditing(site)} className={actionClass}>{t("editor.edit")}</button>
                      <button type="button" onClick={() => removeSite(site)} className={dangerButtonClass}>{t("editor.delete")}</button>
                    </>
                  )}
                </div>
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
                  {site.backends.map((item, index) => (
                    <li key={`${site.key}-${item.api}-${index}`} className="text-secondary break-all text-xs">
                      <span className="text-tertiary">{t("staticConfig.forward")} </span>
                      <span className="text-primary font-mono">{item.api || "/"}</span>
                      <span> → {item.mode === "static" ? item.staticRoot : item.mode === "host" ? `${item.proxyHost}${item.proxyPort ? `:${item.proxyPort}` : ""}` : item.service}</span>
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
      <SaveBar
        dirty={dirty && !locked}
        saving={saving}
        notice={notice || (config.configuredFile && config.configuredFile !== config.source ? t("staticConfig.runtimeFile", { file: config.configuredFile }) : "")}
        onDiscard={() => { setDraft(staticToDraft(config)); setNotice(""); }}
        onSave={() => void save()}
      />
      {editing ? (
        <SiteDialog
          draft={editing}
          onClose={() => setEditing(null)}
          onSubmit={(next) => {
            setDraft((current) => ({
              ...current,
              sites: current.sites.some((item) => item.key === next.key) ? current.sites.map((item) => item.key === next.key ? next : item) : [...current.sites, next],
            }));
            setEditing(null);
          }}
        />
      ) : null}
      {confirm ? <ConfirmDelete name={confirm.name} onClose={() => setConfirm(null)} onConfirm={() => { confirm.run(); setConfirm(null); }} /> : null}
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

function Toggle({ label, checked, disabled, onChange }: { label: string; checked: boolean; disabled: boolean; onChange: (checked: boolean) => void }) {
  return (
    <label className="panel-soft flex items-center justify-between rounded-xl px-3 py-2 text-sm">
      <span>{label}</span>
      <input type="checkbox" checked={checked} disabled={disabled} onChange={(event) => onChange(event.target.checked)} />
    </label>
  );
}
