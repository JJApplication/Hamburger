"use client";

import { useState } from "react";

import { blankAlias, type SiteDraft } from "@/lib/config-edit/model";
import { usePreferences } from "@/lib/preferences/preferences-context";

import { EditorDialog } from "./editor-dialog";
import { CheckField, Field } from "./fields";
import { RuleList } from "./rule-list";

export function SiteDialog({ draft, onClose, onSubmit }: { draft: SiteDraft; onClose: () => void; onSubmit: (draft: SiteDraft) => void }) {
  const { t } = usePreferences();
  const [form, setForm] = useState(draft);
  const patch = (partial: Partial<SiteDraft>) => setForm({ ...form, ...partial });
  return (
    <EditorDialog title={draft.originalName ? t("editor.edit") : t("editor.addSite")} onClose={onClose}>
      <form className="space-y-4" onSubmit={(event) => { event.preventDefault(); onSubmit(form); }}>
        <div className="grid gap-3 md:grid-cols-2">
          <Field label={t("editor.name")} value={form.name} onChange={(name) => patch({ name })} />
          <Field label={t("editor.siteType")} value={form.type} onChange={(type) => patch({ type })} />
          <Field label={t("editor.root")} value={form.root} mono onChange={(root) => patch({ root })} />
          <Field label={t("staticConfig.index")} value={form.index} mono onChange={(index) => patch({ index })} />
          <Field label={t("staticConfig.tryFile")} value={form.tryFile} mono onChange={(tryFile) => patch({ tryFile })} />
          <CheckField label={t("staticConfig.access")} checked={form.access} onChange={(access) => patch({ access })} />
          <CheckField label={t("staticConfig.compress")} checked={form.compress} onChange={(compress) => patch({ compress })} />
        </div>
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium">{t("staticConfig.alias")}</p>
            <button type="button" onClick={() => patch({ alias: [...form.alias, blankAlias()] })} className="text-sm text-emerald-300 [html.theme-light_&]:text-emerald-700">{t("editor.add")}</button>
          </div>
          {form.alias.map((item, index) => (
            <div key={`${item.path}-${index}`} className="grid gap-2 md:grid-cols-[1fr_1fr_auto]">
              <input value={item.path} onChange={(event) => patch({ alias: form.alias.map((alias, aliasIndex) => aliasIndex === index ? { ...alias, path: event.target.value } : alias) })} placeholder={t("editor.api")} className="panel-strong text-primary rounded-xl border border-[var(--border)] px-3 py-2 font-mono text-sm outline-none focus:border-emerald-400/70" />
              <input value={item.root} onChange={(event) => patch({ alias: form.alias.map((alias, aliasIndex) => aliasIndex === index ? { ...alias, root: event.target.value } : alias) })} placeholder={t("editor.root")} className="panel-strong text-primary rounded-xl border border-[var(--border)] px-3 py-2 font-mono text-sm outline-none focus:border-emerald-400/70" />
              <button type="button" onClick={() => patch({ alias: form.alias.filter((_, aliasIndex) => aliasIndex !== index) })} className="text-secondary px-2 text-xs hover:text-rose-300">{t("editor.remove")}</button>
            </div>
          ))}
        </div>
        <RuleList rules={form.backends} onChange={(backends) => patch({ backends })} />
        <div className="flex justify-end">
          <button type="submit" className="rounded-xl bg-emerald-500 px-4 py-2 text-sm font-semibold text-slate-950">{t("editor.save")}</button>
        </div>
      </form>
    </EditorDialog>
  );
}
