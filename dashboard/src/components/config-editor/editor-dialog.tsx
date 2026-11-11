"use client";

import { usePreferences } from "@/lib/preferences/preferences-context";

export function EditorDialog({ title, children, onClose }: { title: string; children: React.ReactNode; onClose: () => void }) {
  const { t } = usePreferences();
  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/55 p-4 sm:items-center" onMouseDown={onClose}>
      <div className="panel my-auto max-h-[calc(100dvh-2rem)] w-full max-w-2xl overflow-y-auto rounded-2xl border p-5" role="dialog" aria-modal="true" aria-label={title} onMouseDown={(event) => event.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between gap-3">
          <h2 className="text-lg font-semibold">{title}</h2>
          <button type="button" onClick={onClose} className="text-secondary rounded-lg px-2 py-1 text-sm hover:text-primary">{t("editor.close")}</button>
        </div>
        {children}
      </div>
    </div>
  );
}
