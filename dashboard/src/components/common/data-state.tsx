"use client";

import { AlertTriangle, LoaderCircle } from "lucide-react";

import { Panel } from "@/components/common/panel";
import { usePreferences } from "@/lib/preferences/preferences-context";

interface DataStateProps {
  isLoading: boolean;
  error: string | null;
  onRetry: () => void;
}

export function DataState({ isLoading, error, onRetry }: DataStateProps) {
  const { t } = usePreferences();

  if (isLoading) {
    return (
      <Panel className="text-secondary flex items-center justify-center gap-2 py-12">
        <LoaderCircle size={18} className="animate-spin" />
        {t("common.loading")}
      </Panel>
    );
  }

  if (!error) {
    return null;
  }

  return (
    <Panel className="flex flex-col items-center justify-center gap-3 py-12 text-center">
      <AlertTriangle size={20} className="text-amber-300" />
      <p className="text-primary text-sm">{error}</p>
      <button
        type="button"
        onClick={onRetry}
        className="text-primary rounded-lg border border-[var(--border)] px-3 py-1.5 text-sm transition hover:opacity-85"
      >
        {t("common.retry")}
      </button>
    </Panel>
  );
}
