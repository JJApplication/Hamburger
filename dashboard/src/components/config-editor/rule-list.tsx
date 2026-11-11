"use client";

import { blankRule, type RuleDraft, type RuleMode } from "@/lib/config-edit/model";
import { usePreferences } from "@/lib/preferences/preferences-context";

import { CheckField, Field, SelectField } from "./fields";

export function RuleList({ rules, onChange }: { rules: RuleDraft[]; onChange: (rules: RuleDraft[]) => void }) {
  const { t } = usePreferences();
  const modes: { value: RuleMode; label: string }[] = [
    { value: "service", label: t("editor.targetService") },
    { value: "static", label: t("editor.targetStatic") },
    { value: "host", label: t("editor.targetHost") },
  ];
  const update = (index: number, patch: Partial<RuleDraft>) => {
    onChange(rules.map((rule, ruleIndex) => (ruleIndex === index ? { ...rule, ...patch } : rule)));
  };
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm font-medium">{t("editor.rules")}</p>
        <button type="button" onClick={() => onChange([...rules, blankRule()])} className="text-sm text-emerald-300 [html.theme-light_&]:text-emerald-700">{t("editor.addRule")}</button>
      </div>
      {rules.map((rule, index) => (
        <div key={`${rule.api}-${index}`} className="panel-strong space-y-3 rounded-xl border border-[var(--border)] p-3">
          <div className="grid gap-3 md:grid-cols-2">
            <Field label={t("editor.api")} value={rule.api} mono onChange={(api) => update(index, { api })} />
            <SelectField label={t("editor.target")} value={rule.mode} options={modes} onChange={(mode) => update(index, { mode: mode as RuleMode })} />
            {rule.mode === "service" ? <Field label={t("domainConfig.service")} value={rule.service} onChange={(service) => update(index, { service })} /> : null}
            {rule.mode === "static" ? <Field label={t("editor.targetStatic")} value={rule.staticRoot} mono onChange={(staticRoot) => update(index, { staticRoot })} /> : null}
            {rule.mode === "host" ? (
              <>
                <Field label={t("editor.proxyHost")} value={rule.proxyHost} mono onChange={(proxyHost) => update(index, { proxyHost })} />
                <Field label={t("editor.proxyPort")} value={rule.proxyPort} onChange={(proxyPort) => update(index, { proxyPort })} />
              </>
            ) : null}
            <Field label={t("editor.rewrite")} value={rule.rewrite} mono onChange={(rewrite) => update(index, { rewrite })} />
          </div>
          <div className="flex items-center justify-between gap-3">
            <CheckField label={t("editor.useRewrite")} checked={rule.useRewrite} onChange={(useRewrite) => update(index, { useRewrite })} />
            <button type="button" onClick={() => onChange(rules.filter((_, ruleIndex) => ruleIndex !== index))} className="text-secondary text-xs hover:text-rose-300">{t("editor.remove")}</button>
          </div>
        </div>
      ))}
    </div>
  );
}
