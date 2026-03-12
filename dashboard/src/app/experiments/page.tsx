"use client";

import { DataState } from "@/components/common/data-state";
import { Panel } from "@/components/common/panel";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { useGatewayBootstrap } from "@/lib/hooks/use-gateway-bootstrap";
import { usePreferences } from "@/lib/preferences/preferences-context";
import { useGatewaySelectors, useGatewayStore } from "@/store/gateway-store";

const riskColorMap = {
  low: "text-emerald-300 bg-emerald-500/15 border-emerald-300/30",
  medium: "text-amber-300 bg-amber-500/15 border-amber-300/30",
  high: "text-rose-300 bg-rose-500/15 border-rose-300/30",
} as const;

export default function ExperimentsPage() {
  useGatewayBootstrap();
  const { t } = usePreferences();

  const experiments = useGatewayStore(useGatewaySelectors.experiments);
  const isLoading = useGatewayStore((state) => state.isLoading);
  const error = useGatewayStore((state) => state.error);
  const refresh = useGatewayStore((state) => state.refresh);
  const toggleExperiment = useGatewayStore((state) => state.toggleExperiment);

  return (
    <DashboardShell title={t("experiments.title")} subtitle={t("experiments.subtitle")}>
      <DataState isLoading={isLoading && experiments.length === 0} error={error} onRetry={() => void refresh()} />

      <section className="grid gap-4 xl:grid-cols-2">
        {experiments.map((feature) => (
          <Panel key={feature.key} className="space-y-4">
            <div className="flex items-start justify-between gap-3">
              <div>
                <h2 className="text-lg font-semibold">{feature.name}</h2>
                <p className="text-secondary mt-1 text-sm">{feature.description}</p>
              </div>
              <span className={["rounded-full border px-2 py-0.5 text-xs", riskColorMap[feature.riskLevel]].join(" ")}>
                {t("experiments.risk")}：{feature.riskLevel}
              </span>
            </div>

            <div className="panel-strong space-y-2 rounded-xl p-3">
              {Object.entries(feature.params).map(([key, value]) => (
                <div key={`${feature.key}-${key}`} className="flex items-center justify-between text-sm">
                  <span className="text-secondary">{key}</span>
                  <span className="text-primary font-medium">{String(value)}</span>
                </div>
              ))}
            </div>

            <button
              type="button"
              onClick={() => toggleExperiment(feature.key)}
              className={[
                "w-full rounded-xl border py-2 text-sm transition",
                feature.enabled
                  ? "border-emerald-400/60 bg-emerald-500/10 text-emerald-200 hover:bg-emerald-500/20"
                  : "border-slate-500 text-slate-200 hover:border-slate-300",
              ].join(" ")}
            >
              {feature.enabled ? t("experiments.enabled") : t("experiments.disabled")}
            </button>
          </Panel>
        ))}
      </section>
    </DashboardShell>
  );
}
