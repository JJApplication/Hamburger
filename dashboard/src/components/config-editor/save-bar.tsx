"use client";

import { LoaderCircle } from "lucide-react";

import type { SaveFailure } from "@/lib/config-edit/model";
import { usePreferences, type MessageKey } from "@/lib/preferences/preferences-context";

export function saveErrorText(t: (key: MessageKey, values?: Record<string, string | number>) => string, error: SaveFailure): string {
  switch (error.code) {
    case "conflict": return t("editor.conflict");
    case "auth": return t("editor.auth");
    case "parse": return t("editor.parse");
    case "empty-name": return t("editor.emptyName");
    case "duplicate-service": return t("editor.duplicateService", { name: error.name });
    case "duplicate-domain": return t("editor.duplicateDomain", { name: error.name });
    case "empty-domain": return t("editor.emptyDomain");
    case "empty-binding": return t("editor.emptyBinding");
    case "invalid-port": return t("editor.invalidPort", { name: error.name });
    case "empty-site": return t("editor.emptySite");
    case "duplicate-site": return t("editor.duplicateSite", { name: error.name });
    case "invalid-rule": return t("editor.invalidRule", { name: error.name });
    case "failed": return t("editor.failed", { message: error.message });
  }
}

export function SaveBar({ dirty, saving, notice, onSave, onDiscard }: { dirty: boolean; saving: boolean; notice: string; onSave: () => void; onDiscard: () => void }) {
  const { t } = usePreferences();
  if (!dirty) return null;
  return (
    <div className="sticky bottom-3 z-20">
      <div className="panel flex flex-wrap items-center gap-3 rounded-2xl border px-4 py-3 shadow-lg">
        <p className="text-secondary mr-auto text-xs">{[t("editor.reloadHint"), notice].filter(Boolean).join(" ")}</p>
        <button type="button" disabled={saving} onClick={onDiscard} className="rounded-xl border border-[var(--border)] px-3 py-2 text-sm disabled:opacity-40">{t("editor.discard")}</button>
        <button type="button" disabled={saving} onClick={onSave} className="flex items-center gap-1.5 rounded-xl bg-emerald-500 px-3 py-2 text-sm font-semibold text-slate-950 disabled:opacity-40">
          {saving ? <LoaderCircle size={15} className="animate-spin" /> : null}
          {saving ? t("editor.saving") : t("editor.save")}
        </button>
      </div>
    </div>
  );
}
