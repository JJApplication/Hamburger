"use client";

import { clsx } from "clsx";

import { usePreferences } from "@/lib/preferences/preferences-context";
import type { DomainHealthStatus } from "@/types/gateway";

interface StatusPillProps {
  status: DomainHealthStatus;
}

const statusStyles: Record<DomainHealthStatus, string> = {
  online: "bg-emerald-500/15 text-emerald-300 border-emerald-400/30",
  warning: "bg-amber-500/15 text-amber-300 border-amber-400/30",
  offline: "bg-rose-500/15 text-rose-300 border-rose-400/30",
};

export function StatusPill({ status }: StatusPillProps) {
  const { t } = usePreferences();

  const statusLabel: Record<DomainHealthStatus, string> = {
    online: t("domain.status.online"),
    warning: t("domain.status.warning"),
    offline: t("domain.status.offline"),
  };

  return (
    <span className={clsx("inline-flex items-center rounded-full border px-2 py-0.5 text-xs", statusStyles[status])}>
      {statusLabel[status]}
    </span>
  );
}
