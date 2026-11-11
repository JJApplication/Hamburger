"use client";

import { useEffect } from "react";

import { usePreferences } from "@/lib/preferences/preferences-context";

export const dangerButtonClass = "shrink-0 whitespace-nowrap rounded-lg border border-rose-400/55 bg-rose-500/10 px-2.5 py-1 text-xs font-medium text-rose-300 transition hover:bg-rose-500/20 [html.theme-light_&]:text-rose-700";

export function ConfirmDelete({ name, onConfirm, onClose }: { name: string; onConfirm: () => void; onClose: () => void }) {
  const { t } = usePreferences();

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/55 p-4" onMouseDown={onClose}>
      <div className="panel w-full max-w-md rounded-2xl border p-5" role="alertdialog" aria-modal="true" aria-labelledby="confirm-delete-title" onMouseDown={(event) => event.stopPropagation()}>
        <h2 id="confirm-delete-title" className="text-lg font-semibold">{t("editor.delete")}</h2>
        <p className="text-secondary mt-2 text-sm">{t("editor.confirmDelete", { name })}</p>
        <div className="mt-5 flex justify-end gap-2">
          <button type="button" autoFocus onClick={onClose} className="rounded-xl border border-[var(--border)] px-3 py-2 text-sm">{t("editor.cancel")}</button>
          <button type="button" onClick={onConfirm} className="rounded-xl bg-rose-600 px-3 py-2 text-sm font-semibold text-white transition hover:bg-rose-500">{t("editor.delete")}</button>
        </div>
      </div>
    </div>
  );
}
