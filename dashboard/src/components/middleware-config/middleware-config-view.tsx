"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import { ConfirmDelete, dangerButtonClass } from "@/components/config-editor/confirm-delete";
import { EnableSwitch } from "@/components/config-editor/enable-switch";
import { Field, SelectField, fieldClass } from "@/components/config-editor/fields";
import { SaveBar, saveErrorText } from "@/components/config-editor/save-bar";
import { Panel } from "@/components/common/panel";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { saveMiddlewareConfig } from "@/lib/config-edit/actions";
import type { MiddlewareConfigModel } from "@/lib/middleware-config/load";
import type { CompressionDraft, ErrorConfigDraft, ErrorPageRow, MiddlewareDraft } from "@/lib/middleware-config/model";
import { useAuth } from "@/lib/auth/auth-context";
import { usePreferences } from "@/lib/preferences/preferences-context";

function LinesField({ label, hint, value, onChange, disabled }: { label: string; hint: string; value: string[]; onChange: (value: string[]) => void; disabled: boolean }) {
  return (
    <label className="block text-sm">
      <span className="text-secondary">{label}</span>
      <textarea
        value={value.join("\n")}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value.split("\n"))}
        rows={Math.min(8, Math.max(3, value.length || 3))}
        className={`${fieldClass} mt-1 font-mono`}
      />
      <span className="text-tertiary mt-1 block text-xs">{hint}</span>
    </label>
  );
}

function Section({ title, enabled, onEnabled, disabled, children }: { title: string; enabled: boolean; onEnabled: (value: boolean) => void; disabled: boolean; children?: React.ReactNode }) {
  const { t } = usePreferences();
  return (
    <Panel className="space-y-3">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-sm font-semibold">{title}</h2>
        <EnableSwitch label={t("middleware.enabled")} checked={enabled} disabled={disabled} onChange={onEnabled} />
      </div>
      {children}
    </Panel>
  );
}

function CompressionFields({ draft, disabled, onChange }: { draft: CompressionDraft; disabled: boolean; onChange: (partial: Partial<CompressionDraft>) => void }) {
  const { t } = usePreferences();
  return (
    <div className="space-y-3">
      <div className="grid gap-3 md:grid-cols-2">
        <Field label={t("middleware.level")} value={draft.level} onChange={(level) => onChange({ level })} />
        <Field label={t("middleware.threshold")} value={draft.threshold} onChange={(threshold) => onChange({ threshold })} />
      </div>
      <LinesField label={t("middleware.types")} hint={t("middleware.onePerLine")} value={draft.types} disabled={disabled} onChange={(types) => onChange({ types })} />
    </div>
  );
}

export function MiddlewareConfigView({ config }: { config: MiddlewareConfigModel }) {
  const { t } = usePreferences();
  const { token } = useAuth();
  const router = useRouter();
  const [draft, setDraft] = useState(config.draft);
  const [errorConfig, setErrorConfig] = useState(config.errorConfig);
  const [seen, setSeen] = useState(config.version);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState("");
  const [confirm, setConfirm] = useState<{ name: string; run: () => void } | null>(null);
  if (seen !== config.version) {
    setSeen(config.version);
    setDraft(config.draft);
    setErrorConfig(config.errorConfig);
    setNotice("");
    setConfirm(null);
  }
  const locked = Boolean(config.error);
  const dirty = JSON.stringify(draft) !== JSON.stringify(config.draft) || JSON.stringify(errorConfig) !== JSON.stringify(config.errorConfig);

  const patch = (partial: Partial<MiddlewareDraft>) => setDraft((current) => ({ ...current, ...partial }));
  const patchError = (partial: Partial<ErrorConfigDraft>) => setErrorConfig((current) => ({ ...current, ...partial }));
  const patchPage = (key: string, partial: Partial<ErrorPageRow>) => setErrorConfig((current) => ({
    ...current,
    pages: current.pages.map((page) => (page.key === key ? { ...page, ...partial } : page)),
  }));

  const save = async () => {
    setSaving(true);
    setNotice("");
    try {
      const result = await saveMiddlewareConfig(token ?? "", config.version, draft, errorConfig);
      if (!result.ok) {
        setNotice(saveErrorText(t, result.error));
        return;
      }
      router.refresh();
    } finally {
      setSaving(false);
    }
  };

  const modes = [
    { value: "text", label: t("middleware.modeText") },
    { value: "json", label: t("middleware.modeJson") },
    { value: "html", label: t("middleware.modeHtml") },
    { value: "internal_html", label: t("middleware.modeInternal") },
  ];

  return (
    <DashboardShell title={t("middleware.title")} subtitle={t("middleware.subtitle")}>
      <div className="space-y-4">
        <p className="text-secondary text-xs">{config.source}</p>
        {config.error ? <p className="rounded-xl border border-rose-400/40 bg-rose-500/10 px-3 py-2 text-sm text-rose-300 [html.theme-light_&]:text-rose-700">{config.error}</p> : null}
        <div className="grid gap-4 xl:grid-cols-2">
          <Section title={t("middleware.cors")} enabled={draft.cors.enabled} disabled={locked} onEnabled={(enabled) => patch({ cors: { ...draft.cors, enabled } })}>
            <LinesField label={t("middleware.methods")} hint={t("middleware.onePerLine")} value={draft.cors.methods} disabled={locked} onChange={(methods) => patch({ cors: { ...draft.cors, methods } })} />
            <LinesField label={t("middleware.origins")} hint={t("middleware.onePerLine")} value={draft.cors.origins} disabled={locked} onChange={(origins) => patch({ cors: { ...draft.cors, origins } })} />
            <LinesField label={t("middleware.headers")} hint={t("middleware.onePerLine")} value={draft.cors.headers} disabled={locked} onChange={(headers) => patch({ cors: { ...draft.cors, headers } })} />
          </Section>
          <Section title={t("middleware.trace")} enabled={draft.trace.enabled} disabled={locked} onEnabled={(enabled) => patch({ trace: { ...draft.trace, enabled } })}>
            <Field label={t("middleware.traceId")} value={draft.trace.traceId} mono onChange={(traceId) => patch({ trace: { ...draft.trace, traceId } })} />
          </Section>
          <Section title={t("middleware.gzip")} enabled={draft.gzip.enabled} disabled={locked} onEnabled={(enabled) => patch({ gzip: { ...draft.gzip, enabled } })}>
            <CompressionFields draft={draft.gzip} disabled={locked} onChange={(partial) => patch({ gzip: { ...draft.gzip, ...partial } })} />
          </Section>
          <Section title={t("middleware.zstd")} enabled={draft.zstd.enabled} disabled={locked} onEnabled={(enabled) => patch({ zstd: { ...draft.zstd, enabled } })}>
            <CompressionFields draft={draft.zstd} disabled={locked} onChange={(partial) => patch({ zstd: { ...draft.zstd, ...partial } })} />
          </Section>
        </div>
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <Section title={t("middleware.secureHeader")} enabled={draft.secureHeader} disabled={locked} onEnabled={(secureHeader) => patch({ secureHeader })} />
          <Section title={t("middleware.noCache")} enabled={draft.noCache} disabled={locked} onEnabled={(noCache) => patch({ noCache })} />
          <Section title={t("middleware.sanitizer")} enabled={draft.sanitizer} disabled={locked} onEnabled={(sanitizer) => patch({ sanitizer })} />
          <Section title={t("middleware.domainCheck")} enabled={draft.domainCheck} disabled={locked} onEnabled={(domainCheck) => patch({ domainCheck })} />
        </div>
        <div className="grid gap-4 xl:grid-cols-2">
          <Section title={t("middleware.imageProtect")} enabled={draft.imageProtect.enabled} disabled={locked} onEnabled={(enabled) => patch({ imageProtect: { ...draft.imageProtect, enabled } })}>
            <LinesField label={t("middleware.imageTypes")} hint={t("middleware.onePerLine")} value={draft.imageProtect.imageTypes} disabled={locked} onChange={(imageTypes) => patch({ imageProtect: { ...draft.imageProtect, imageTypes } })} />
            <LinesField label={t("middleware.allowReferer")} hint={t("middleware.onePerLine")} value={draft.imageProtect.allowReferers} disabled={locked} onChange={(allowReferers) => patch({ imageProtect: { ...draft.imageProtect, allowReferers } })} />
          </Section>
          <Panel className="space-y-3">
            <div className="flex items-center justify-between gap-3">
              <h2 className="text-sm font-semibold">{t("middleware.errorPage")}</h2>
              <EnableSwitch label={t("middleware.pageCache")} checked={errorConfig.cache} disabled={locked} onChange={(cache) => patchError({ cache })} />
            </div>
            <SelectField label={t("middleware.errorMode")} value={errorConfig.mode} options={modes} onChange={(mode) => patchError({ mode })} />
            <div className="space-y-2">
              <div className="grid grid-cols-[7rem_1fr_auto] gap-2 text-xs text-secondary">
                <span>{t("middleware.statusCode")}</span>
                <span>{t("middleware.pagePath")}</span>
                <span />
              </div>
              {errorConfig.pages.map((page) => (
                <div key={page.key} className="grid grid-cols-[7rem_1fr_auto] gap-2">
                  <input value={page.code} disabled={locked} onChange={(event) => patchPage(page.key, { code: event.target.value })} className={`${fieldClass} font-mono`} />
                  <input value={page.path} disabled={locked} onChange={(event) => patchPage(page.key, { path: event.target.value })} className={`${fieldClass} font-mono`} />
                  <button type="button" disabled={locked} onClick={() => setConfirm({ name: page.code.trim() || page.path.trim() || t("middleware.errorPage"), run: () => setErrorConfig((current) => ({ ...current, pages: current.pages.filter((item) => item.key !== page.key) })) })} className={`${dangerButtonClass} self-center disabled:opacity-40`}>{t("editor.remove")}</button>
                </div>
              ))}
            </div>
            <button type="button" disabled={locked} onClick={() => patchError({ pages: [...errorConfig.pages, { key: `page-${Date.now()}`, code: "", path: "" }] })} className="text-sm text-emerald-300 disabled:opacity-40 [html.theme-light_&]:text-emerald-700">{t("middleware.addPage")}</button>
          </Panel>
        </div>
        <SaveBar dirty={dirty && !locked} saving={saving} notice={notice} onSave={() => void save()} onDiscard={() => { setDraft(config.draft); setErrorConfig(config.errorConfig); setNotice(""); setConfirm(null); }} />
        {confirm ? <ConfirmDelete name={confirm.name} onClose={() => setConfirm(null)} onConfirm={() => { confirm.run(); setConfirm(null); }} /> : null}
      </div>
    </DashboardShell>
  );
}
