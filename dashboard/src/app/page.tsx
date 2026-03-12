"use client";

import { AlertCircle, Link2, Network, PlugZap } from "lucide-react";

import { ConnectionTrendChart } from "@/components/charts/connection-trend-chart";
import { DataState } from "@/components/common/data-state";
import { Panel } from "@/components/common/panel";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { useGatewayBootstrap } from "@/lib/hooks/use-gateway-bootstrap";
import { usePreferences } from "@/lib/preferences/preferences-context";
import { useGatewaySelectors, useGatewayStore } from "@/store/gateway-store";

const kpiItems = [
  { key: "totalConnections", icon: PlugZap },
  { key: "activeDomains", icon: Network },
  { key: "warningDomains", icon: AlertCircle },
  { key: "totalMappings", icon: Link2 },
] as const;

export default function Home() {
  useGatewayBootstrap();
  const { t } = usePreferences();

  const overview = useGatewayStore(useGatewaySelectors.overviewSummary);
  const refresh = useGatewayStore((state) => state.refresh);
  const isLoading = useGatewayStore((state) => state.isLoading);
  const error = useGatewayStore((state) => state.error);

  return (
    <DashboardShell title={t("overview.title")} subtitle={t("overview.subtitle")}>
      <DataState isLoading={isLoading && !overview} error={error} onRetry={() => void refresh()} />
      {overview ? (
        <div className="space-y-5">
          <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            {kpiItems.map((item) => {
              const Icon = item.icon;
              const labelMap = {
                totalConnections: t("overview.totalConnections"),
                activeDomains: t("overview.activeDomains"),
                warningDomains: t("overview.warningDomains"),
                totalMappings: t("overview.totalMappings"),
              } as const;
              return (
                <Panel key={item.key}>
                  <div className="flex items-center justify-between">
                    <p className="text-secondary text-sm">{labelMap[item.key]}</p>
                    <Icon size={15} className="text-emerald-300" />
                  </div>
                  <p className="mt-3 text-3xl font-semibold">{overview[item.key]}</p>
                </Panel>
              );
            })}
          </section>

          <section className="grid gap-4 xl:grid-cols-[2fr_1fr]">
            <Panel>
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-semibold">{t("overview.trend")}</h2>
                <p className="text-secondary text-xs">{t("overview.lastHour")}</p>
              </div>
              <div className="mt-4">
                <ConnectionTrendChart data={overview.trend} />
              </div>
            </Panel>

            <Panel className="space-y-3">
              <h2 className="text-lg font-semibold">{t("overview.runtimeSummary")}</h2>
              <div className="panel-soft rounded-xl p-3">
                <p className="text-secondary text-xs">{t("overview.criticalAlerts")}</p>
                <p className="mt-1 text-xl font-semibold text-amber-300">{overview.criticalAlerts}</p>
              </div>
              <div className="panel-soft rounded-xl p-3">
                <p className="text-secondary text-xs">{t("overview.latestChange")}</p>
                <p className="text-primary mt-1 text-sm">{overview.latestChange}</p>
              </div>
              <button
                type="button"
                onClick={() => void refresh()}
                className="text-primary w-full rounded-xl border border-[var(--border)] py-2 text-sm transition hover:opacity-85"
              >
                {t("overview.refresh")}
              </button>
            </Panel>
          </section>
        </div>
      ) : null}
    </DashboardShell>
  );
}
