"use client";

import { DataState } from "@/components/common/data-state";
import { Panel } from "@/components/common/panel";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { useGatewayBootstrap } from "@/lib/hooks/use-gateway-bootstrap";
import { usePreferences } from "@/lib/preferences/preferences-context";
import { useGatewaySelectors, useGatewayStore } from "@/store/gateway-store";

function Item({ label, value }: { label: string; value: string | number | boolean }) {
  return (
    <div className="panel-strong flex items-center justify-between rounded-xl px-3 py-2 text-sm">
      <span className="text-secondary">{label}</span>
      <span className="text-primary font-medium">{String(value)}</span>
    </div>
  );
}

export default function ConfigsPage() {
  useGatewayBootstrap();
  const { t } = usePreferences();

  const configs = useGatewayStore(useGatewaySelectors.configs);
  const isLoading = useGatewayStore((state) => state.isLoading);
  const error = useGatewayStore((state) => state.error);
  const refresh = useGatewayStore((state) => state.refresh);

  return (
    <DashboardShell title={t("configs.title")} subtitle={t("configs.subtitle")}>
      <DataState isLoading={isLoading && !configs} error={error} onRetry={() => void refresh()} />
      {configs ? (
        <section className="grid gap-4 xl:grid-cols-2">
          <Panel className="space-y-3">
            <h2 className="text-lg font-semibold">{t("configs.core")}</h2>
            <Item label={t("configs.mode")} value={configs.core.mode} />
            <Item label={t("configs.listener")} value={configs.core.listener} />
            <Item label={t("configs.logLevel")} value={configs.core.logLevel} />
            <Item label={t("configs.maxConnections")} value={configs.core.maxConnections} />
            <Item label={t("configs.readTimeout")} value={configs.core.readTimeoutMs} />
            <Item label={t("configs.writeTimeout")} value={configs.core.writeTimeoutMs} />
          </Panel>

          <Panel className="space-y-3">
            <h2 className="text-lg font-semibold">{t("configs.frontend")}</h2>
            <Item label={t("configs.upstream")} value={configs.frontendProxy.upstreamScheme} />
            <Item label={t("configs.gzip")} value={configs.frontendProxy.gzip} />
            <Item label={t("configs.websocket")} value={configs.frontendProxy.websocket} />
            <Item label={t("configs.cors")} value={configs.frontendProxy.cors} />
            <Item label={t("configs.errorPage")} value={configs.frontendProxy.errorPage} />
          </Panel>

          <Panel className="space-y-3 xl:col-span-2">
            <h2 className="text-lg font-semibold">{t("configs.backend")}</h2>
            <div className="overflow-x-auto">
              <table className="min-w-full text-left text-sm">
                <thead className="text-secondary">
                  <tr>
                    <th className="px-3 py-2">{t("configs.serviceName")}</th>
                    <th className="px-3 py-2">{t("configs.target")}</th>
                    <th className="px-3 py-2">{t("configs.healthCheck")}</th>
                    <th className="px-3 py-2">{t("configs.status")}</th>
                    <th className="px-3 py-2">{t("configs.weight")}</th>
                  </tr>
                </thead>
                <tbody>
                  {configs.backendServices.map((service) => (
                    <tr key={service.name} className="border-t border-white/5">
                      <td className="px-3 py-2 font-medium">{service.name}</td>
                      <td className="px-3 py-2">{service.target}</td>
                      <td className="px-3 py-2">{service.healthCheckPath}</td>
                      <td className="px-3 py-2">{service.healthStatus}</td>
                      <td className="px-3 py-2">{service.weight}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Panel>
        </section>
      ) : null}
    </DashboardShell>
  );
}
