"use client";

import { DataState } from "@/components/common/data-state";
import { Panel } from "@/components/common/panel";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { useEffect, useState } from "react";
import { Save, Send, LoaderCircle } from "lucide-react";
import { applyManagementConfig, fetchOperation, saveManagementConfig } from "@/lib/api/gateway";
import { useGatewayBootstrap } from "@/lib/hooks/use-gateway-bootstrap";
import { usePreferences } from "@/lib/preferences/preferences-context";
import { useGatewaySelectors, useGatewayStore } from "@/store/gateway-store";
import { useAuth } from "@/lib/auth/auth-context";

function Item({ label, value }: { label: string; value: string | number | boolean }) {
  return (
    <div className="panel-strong flex items-center justify-between rounded-xl px-3 py-2 text-sm">
      <span className="text-secondary">{label}</span>
      <span className="text-primary font-medium">{String(value)}</span>
    </div>
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function flattenChanges(before: unknown, after: unknown, prefix = "", result: Record<string, unknown> = {}) {
  if (isRecord(before) && isRecord(after)) {
    const keys = new Set([...Object.keys(before), ...Object.keys(after)]);
    keys.forEach((key) => flattenChanges(before[key], after[key], prefix ? `${prefix}.${key}` : key, result));
    return result;
  }
  if (JSON.stringify(before) !== JSON.stringify(after) && prefix) result[prefix] = after;
  return result;
}

export default function ConfigsPage() {
  useGatewayBootstrap();
  const { t } = usePreferences();
  const { token } = useAuth();
  const [proxyMode, setProxyMode] = useState("");
  const [maxConnections, setMaxConnections] = useState(0);
  const [gzip, setGzip] = useState(false);
  const [cache, setCache] = useState(false);
  const [websocket, setWebsocket] = useState(false);
  const [advancedText, setAdvancedText] = useState("{}");
  const [saveMessage, setSaveMessage] = useState("");
  const [saving, setSaving] = useState(false);
  const configs = useGatewayStore(useGatewaySelectors.configs);
  const isLoading = useGatewayStore((state) => state.isLoading);
  const error = useGatewayStore((state) => state.error);
  const refresh = useGatewayStore((state) => state.refresh);

  useEffect(() => {
    if (!configs) return;
    setProxyMode(configs.core.mode);
    setMaxConnections(configs.core.maxConnections);
    setGzip(configs.frontendProxy.gzip);
    setCache(configs.frontendProxy.cache);
    setWebsocket(configs.frontendProxy.websocket);
    setAdvancedText(JSON.stringify(configs.raw?.values ?? {}, null, 2));
  }, [configs]);

  const save = async () => {
    const raw = configs?.raw;
    if (!token || !raw) return;
    setSaving(true); setSaveMessage("");
    try {
      let advanced: unknown;
      try { advanced = JSON.parse(advancedText); } catch { throw new Error("高级配置不是有效 JSON"); }
      if (!isRecord(advanced)) throw new Error("高级配置必须是 JSON 对象");
      const values = flattenChanges(raw.values, advanced);
      Object.assign(values, { "proxy.proxy_mode": proxyMode, "proxy.max_conns_per_host": maxConnections, "middleware.gzip.enabled": gzip, "features.proxy_cache.enabled": cache, "features.websocket.enabled": websocket });
      await saveManagementConfig({ version: raw.version, values }, token);
      await refresh(token);
      setSaveMessage(`已保存 ${Object.keys(values).length} 项，等待应用`);
    } catch (error) { setSaveMessage(error instanceof Error ? error.message : "保存失败"); }
    finally { setSaving(false); }
  };
  const apply = async () => {
    if (!token) return;
    setSaving(true); setSaveMessage("");
    try {
      const result = await applyManagementConfig(["gateway"], token);
      for (let attempt = 0; attempt < 20; attempt += 1) {
        const status = await fetchOperation(result.operation_id, token);
        if (status.status === "complete") { setSaveMessage("配置已应用"); void refresh(token); break; }
        if (status.status === "failed") throw new Error(status.error ?? "应用失败");
        await new Promise((resolve) => window.setTimeout(resolve, 250));
      }
    } catch (error) { setSaveMessage(error instanceof Error ? error.message : "应用失败"); }
    finally { setSaving(false); }
  };

  return (
    <DashboardShell title={t("configs.title")} subtitle={t("configs.subtitle")}>
      <DataState isLoading={isLoading && !configs} error={error} onRetry={() => token && void refresh(token)} />
      {configs ? (
        <section className="grid gap-4 xl:grid-cols-2">
          <Panel className="space-y-4 xl:col-span-2">
            <div className="flex flex-wrap items-start justify-between gap-3"><div><h2 className="text-lg font-semibold">可编辑运行配置</h2><p className="text-secondary mt-1 text-xs">{configs.raw?.source_file ?? "未读取配置源"} · 版本 {configs.raw?.version?.slice(0, 12) ?? "-"}</p></div><span className={`rounded-full border px-2 py-1 text-xs ${configs.raw?.pending ? "border-amber-400/40 text-amber-300" : "border-emerald-400/40 text-emerald-300"}`}>{configs.raw?.pending ? "待应用" : "已生效"}</span></div>
            <div className="grid gap-3 md:grid-cols-2"><label className="text-secondary text-sm">代理模式<input value={proxyMode} onChange={(event) => setProxyMode(event.target.value)} className="panel-strong text-primary mt-1 w-full rounded-xl border border-[var(--border)] px-3 py-2 outline-none" /></label><label className="text-secondary text-sm">每主机最大连接<input type="number" min={0} value={maxConnections} onChange={(event) => setMaxConnections(Number(event.target.value))} className="panel-strong text-primary mt-1 w-full rounded-xl border border-[var(--border)] px-3 py-2 outline-none" /></label><label className="panel-soft flex items-center justify-between rounded-xl px-3 py-2 text-sm"><span>Gzip 压缩</span><input type="checkbox" checked={gzip} onChange={(event) => setGzip(event.target.checked)} /></label><label className="panel-soft flex items-center justify-between rounded-xl px-3 py-2 text-sm"><span>代理缓存</span><input type="checkbox" checked={cache} onChange={(event) => setCache(event.target.checked)} /></label><label className="panel-soft flex items-center justify-between rounded-xl px-3 py-2 text-sm"><span>WebSocket</span><input type="checkbox" checked={websocket} onChange={(event) => setWebsocket(event.target.checked)} /></label></div>
            <label className="text-secondary block text-sm">高级结构化配置（JSON）<textarea value={advancedText} onChange={(event) => setAdvancedText(event.target.value)} spellCheck={false} className="panel-strong text-primary mt-1 min-h-48 w-full rounded-xl border border-[var(--border)] px-3 py-2 font-mono text-xs leading-5 outline-none focus:border-emerald-400" /></label>
            <div className="flex flex-wrap items-center justify-end gap-2"><span className="text-secondary mr-auto text-xs">保存会写回原配置文件；应用会重载网关。</span>{saveMessage ? <span className="text-secondary text-xs">{saveMessage}</span> : null}<button type="button" disabled={saving || !configs.raw} onClick={() => void save()} className="flex items-center gap-1.5 rounded-xl border border-[var(--border)] px-3 py-2 text-sm"><Save size={15} />保存</button><button type="button" disabled={saving || !configs.raw?.pending} onClick={() => void apply()} className="flex items-center gap-1.5 rounded-xl bg-emerald-500 px-3 py-2 text-sm font-semibold text-slate-950 disabled:opacity-40">{saving ? <LoaderCircle size={15} className="animate-spin" /> : <Send size={15} />}应用</button></div>
          </Panel>
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
