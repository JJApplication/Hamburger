"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import { EnableSwitch } from "@/components/config-editor/enable-switch";
import { Field, fieldClass } from "@/components/config-editor/fields";
import { SaveBar, saveErrorText } from "@/components/config-editor/save-bar";
import { Panel } from "@/components/common/panel";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { saveSecurityConfig } from "@/lib/config-edit/actions";
import type { SecurityConfigModel } from "@/lib/security-config/load";
import type { SecurityDraft } from "@/lib/security-config/model";
import { useAuth } from "@/lib/auth/auth-context";
import { usePreferences, type MessageKey } from "@/lib/preferences/preferences-context";

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

function SwitchRow({ title, hint, checked, disabled, onChange }: { title: string; hint?: string; checked: boolean; disabled: boolean; onChange: (value: boolean) => void }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-xl border border-[var(--border)] px-3 py-2">
      <div className="min-w-0">
        <p className="text-sm">{title}</p>
        {hint ? <p className="text-tertiary mt-0.5 text-xs">{hint}</p> : null}
      </div>
      <EnableSwitch label={title} hideLabel checked={checked} disabled={disabled} onChange={onChange} />
    </div>
  );
}

export function SecurityConfigView({ config }: { config: SecurityConfigModel }) {
  const { t } = usePreferences();
  const { token } = useAuth();
  const router = useRouter();
  const [draft, setDraft] = useState(config.draft);
  const [seen, setSeen] = useState(config.version);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState("");
  if (seen !== config.version) {
    setSeen(config.version);
    setDraft(config.draft);
    setNotice("");
  }
  const locked = Boolean(config.error);
  const dirty = JSON.stringify(draft) !== JSON.stringify(config.draft);
  const patch = (partial: Partial<SecurityDraft>) => setDraft((current) => ({ ...current, ...partial }));

  const save = async () => {
    setSaving(true);
    setNotice("");
    try {
      const result = await saveSecurityConfig(token ?? "", config.version, draft);
      if (!result.ok) {
        setNotice(saveErrorText(t, result.error));
        return;
      }
      router.refresh();
    } finally {
      setSaving(false);
    }
  };

  const row = (key: MessageKey, field: keyof SecurityDraft, hint?: MessageKey) => (
    <SwitchRow
      title={t(key)}
      hint={hint ? t(hint) : undefined}
      checked={Boolean(draft[field])}
      disabled={locked}
      onChange={(value) => patch({ [field]: value })}
    />
  );

  return (
    <DashboardShell title={t("security.title")} subtitle={t("security.subtitle")}>
      <div className="space-y-4">
        <p className="text-secondary text-xs">{config.source}</p>
        {config.error ? <p className="rounded-xl border border-rose-400/40 bg-rose-500/10 px-3 py-2 text-sm text-rose-300 [html.theme-light_&]:text-rose-700">{config.error}</p> : null}
        <Panel className="space-y-3">
          <h2 className="text-sm font-semibold">{t("security.access")}</h2>
          {row("security.strictMode", "strictMode", "security.strictHint")}
          <div className="grid gap-3 md:grid-cols-2">
            <Field label={t("security.rateLimit")} value={draft.rateLimit} onChange={(rateLimit) => patch({ rateLimit })} />
            <Field label={t("security.maxQuerySize")} value={draft.maxQuerySize} onChange={(maxQuerySize) => patch({ maxQuerySize })} />
          </div>
          <p className="text-tertiary text-xs">{t("security.maxQueryHint")}</p>
          <div className="grid gap-3 xl:grid-cols-2">
            <LinesField label={t("security.allowIps")} hint={t("security.onePerLine")} value={draft.allowIps} disabled={locked} onChange={(allowIps) => patch({ allowIps })} />
            <LinesField label={t("security.denyIps")} hint={t("security.onePerLine")} value={draft.denyIps} disabled={locked} onChange={(denyIps) => patch({ denyIps })} />
          </div>
        </Panel>
        <Panel className="space-y-3">
          <LinesField label={t("security.whiteList")} hint={t("security.whiteListHint")} value={draft.whiteList} disabled={locked} onChange={(whiteList) => patch({ whiteList })} />
        </Panel>
        <Panel className="space-y-3">
          <h2 className="text-sm font-semibold">{t("security.headers")}</h2>
          <div className="grid gap-3 md:grid-cols-2">
            {row("security.hsts", "hsts", "security.hstsHint")}
            {row("security.hstsSubdomain", "hstsSubdomain")}
            {row("security.hstsPreload", "hstsPreload")}
            {row("security.xss", "xss")}
            {row("security.iframe", "iframe", "security.iframeHint")}
            {row("security.sameSite", "sameSite")}
          </div>
        </Panel>
        <SaveBar dirty={dirty && !locked} saving={saving} notice={notice} onSave={() => void save()} onDiscard={() => { setDraft(config.draft); setNotice(""); }} />
      </div>
    </DashboardShell>
  );
}
