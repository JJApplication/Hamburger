"use client";

import { useState } from "react";

import type { ServiceDraft } from "@/lib/config-edit/model";
import { usePreferences } from "@/lib/preferences/preferences-context";

import { EditorDialog } from "./editor-dialog";
import { CheckField, Field, SelectField } from "./fields";
import { RuleList } from "./rule-list";

export function ServiceDialog({ draft, onClose, onSubmit }: { draft: ServiceDraft; onClose: () => void; onSubmit: (draft: ServiceDraft) => void }) {
  const { t } = usePreferences();
  const [form, setForm] = useState(draft);
  const patch = (partial: Partial<ServiceDraft>) => setForm({ ...form, ...partial });
  return (
    <EditorDialog title={draft.originalName ? t("editor.edit") : t("editor.addService")} onClose={onClose}>
      <form className="space-y-4" onSubmit={(event) => { event.preventDefault(); onSubmit(form); }}>
        <div className="grid gap-3 md:grid-cols-2">
          <Field label={t("editor.name")} value={form.name} onChange={(name) => patch({ name })} />
          <SelectField
            label={t("domainConfig.type")}
            value={form.type}
            options={[
              { value: "frontend", label: t("domainConfig.frontend") },
              { value: "backend", label: t("domainConfig.backend") },
              { value: "custom", label: t("domainConfig.custom") },
            ]}
            onChange={(type) => patch({ type })}
          />
          <Field label={t("editor.domain")} value={form.domain} mono onChange={(domain) => patch({ domain })} />
          <Field label={t("editor.host")} value={form.host} mono onChange={(host) => patch({ host })} />
          <Field label={t("editor.port")} value={form.port} onChange={(port) => patch({ port })} />
          <CheckField label={t("editor.precheck")} checked={form.preCheck} onChange={(preCheck) => patch({ preCheck })} />
        </div>
        <RuleList rules={form.rules} onChange={(rules) => patch({ rules })} />
        <div className="flex justify-end">
          <button type="submit" className="rounded-xl bg-emerald-500 px-4 py-2 text-sm font-semibold text-slate-950">{t("editor.save")}</button>
        </div>
      </form>
    </EditorDialog>
  );
}
