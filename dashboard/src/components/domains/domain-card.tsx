"use client";

import { Server } from "lucide-react";

import { StatusPill } from "@/components/common/status-pill";
import { usePreferences } from "@/lib/preferences/preferences-context";
import type { DomainConnection } from "@/types/gateway";

interface DomainCardProps {
  domain: DomainConnection;
}

export function DomainCard({ domain }: DomainCardProps) {
  const { t } = usePreferences();

  return (
    <article className="panel rounded-2xl border p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-lg font-semibold">{domain.domain}</p>
          <p className="text-secondary mt-1 text-xs">
            {t("domain.lastHeartbeat")}：{domain.lastHeartbeat}
          </p>
        </div>
        <StatusPill status={domain.status} />
      </div>

      <div className="mt-4 grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
        <div className="panel-soft rounded-xl p-3">
          <p className="text-secondary">{t("domain.currentConnections")}</p>
          <p className="mt-1 text-xl font-semibold">{domain.currentConnections}</p>
        </div>
        <div className="panel-soft rounded-xl p-3">
          <p className="text-secondary">{t("domain.peakConnections")}</p>
          <p className="mt-1 text-xl font-semibold">{domain.peakConnections}</p>
        </div>
        <div className="panel-soft col-span-2 rounded-xl p-3 sm:col-span-1">
          <p className="text-secondary">{t("domain.mappingCount")}</p>
          <p className="mt-1 text-xl font-semibold">{domain.mappings.length}</p>
        </div>
      </div>

      <div className="mt-4 space-y-2">
        {domain.mappings.map((mapping, index) => (
          <div key={`${domain.domain}-${mapping.frontendPort}-${index}`} className="panel-strong flex flex-wrap items-center gap-2 rounded-xl px-3 py-2 text-sm">
            <Server size={14} className="text-secondary" />
            <span className="rounded bg-cyan-500/15 px-1.5 py-0.5 text-cyan-300">{mapping.protocol.toUpperCase()}</span>
            <span className="text-secondary">
              {domain.domain}:{mapping.frontendPort}
            </span>
            <span className="text-tertiary">→</span>
            <span className="text-primary font-medium">{mapping.backendTarget}</span>
          </div>
        ))}
      </div>
    </article>
  );
}
