"use client";

import { useEffect } from "react";
import { X } from "lucide-react";

import { usePreferences } from "@/lib/preferences/preferences-context";
import packageJson from "../../../package.json";

export function AboutDialog({ onClose }: { onClose: () => void }) {
  const { t } = usePreferences();

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/65 p-4 backdrop-blur-sm" role="dialog" aria-modal="true" aria-labelledby="about-title" onMouseDown={onClose}>
      <div className="panel w-full max-w-md rounded-2xl border p-5 shadow-2xl" onMouseDown={(event) => event.stopPropagation()}>
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="text-secondary text-xs uppercase tracking-[0.2em]">Hamburger</p>
            <h2 id="about-title" className="mt-1 text-xl font-semibold">{t("about.title")}</h2>
          </div>
          <button type="button" onClick={onClose} className="text-secondary rounded-full p-2 hover:opacity-75" aria-label="关闭"><X size={18} /></button>
        </div>
        <p className="text-secondary mt-4 text-sm leading-6">{t("about.summary")}</p>
        <dl className="mt-4 space-y-2 text-sm">
          <div className="flex items-center justify-between gap-3 rounded-xl border border-[var(--border)] px-3 py-2">
            <dt className="text-secondary">{t("about.panel")}</dt>
            <dd className="font-mono">{packageJson.version}</dd>
          </div>
          <div className="flex items-center justify-between gap-3 rounded-xl border border-[var(--border)] px-3 py-2">
            <dt className="text-secondary">{t("about.site")}</dt>
            <dd><a href="http://hamburger.renj.io" target="_blank" rel="noreferrer" className="text-emerald-300 hover:underline [html.theme-light_&]:text-emerald-700">hamburger.renj.io</a></dd>
          </div>
        </dl>
      </div>
    </div>
  );
}
