"use client";
/* eslint-disable react-hooks/set-state-in-effect */

import { useEffect, useState } from "react";
import { DataState } from "@/components/common/data-state";
import { Panel } from "@/components/common/panel";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { useGatewayBootstrap } from "@/lib/hooks/use-gateway-bootstrap";
import { usePreferences } from "@/lib/preferences/preferences-context";
import { useGatewaySelectors, useGatewayStore } from "@/store/gateway-store";
import { useAuth } from "@/lib/auth/auth-context";
import { saveManagementConfig } from "@/lib/api/gateway";

const riskColorMap = {
  low: "text-emerald-300 bg-emerald-500/15 border-emerald-300/30",
  medium: "text-amber-300 bg-amber-500/15 border-amber-300/30",
  high: "text-rose-300 bg-rose-500/15 border-rose-300/30",
} as const;

type AnyTLSForm = { host: string; port: number; user: string; cert_file: string; key_file: string; dial_timeout: number; password: string };

export default function ExperimentsPage() {
  useGatewayBootstrap();
  const { t } = usePreferences();
  const { token } = useAuth();

  const experiments = useGatewayStore(useGatewaySelectors.experiments);
  const isLoading = useGatewayStore((state) => state.isLoading);
  const error = useGatewayStore((state) => state.error);
  const refresh = useGatewayStore((state) => state.refresh);
  const configs = useGatewayStore(useGatewaySelectors.configs);
  const [message, setMessage] = useState("");
  const [anytlsForm, setAnytlsForm] = useState<AnyTLSForm>({ host: "", port: 0, user: "", cert_file: "", key_file: "", dial_timeout: 0, password: "" });
  const [trojanPath, setTrojanPath] = useState("");

  useEffect(() => {
    const values = configs?.raw?.values ?? {};
    const exp = (values.exp_config ?? {}) as Record<string, unknown>;
    const anytls = (exp.any_tls_server ?? {}) as Record<string, unknown>;
    setAnytlsForm({ host: String(anytls.host ?? ""), port: Number(anytls.port ?? 0), user: String(anytls.user ?? ""), cert_file: String(anytls.cert_file ?? ""), key_file: String(anytls.key_file ?? ""), dial_timeout: Number(anytls.dial_timeout ?? 0), password: "" });
    setTrojanPath(String(exp.trojan_server ?? ""));
  }, [configs?.raw]);

  const toggle = async (key: "anytls" | "trojan", enabled: boolean) => {
    if (!token || !configs?.raw) return;
    if (key === "trojan" && !enabled) {
      const expValues = configs.raw.values.exp_config as { trojan_server?: unknown } | undefined;
      if (!String(expValues?.trojan_server ?? "").trim()) {
        setMessage("Trojan 尚未配置独立配置文件，暂不能启用");
        return;
      }
    }
    const field = key === "anytls" ? "exp_config.any_tls_server.enabled" : "exp_config.trojan_enabled";
    if (key === "trojan" && !enabled) { setMessage("Trojan 将保留配置路径，应用后停止服务"); }
    try {
      await saveManagementConfig({ version: configs.raw.version, values: { [field]: !enabled } }, token);
      await refresh(token);
      setMessage("已保存，点击配置页应用后生效");
    } catch (error) { setMessage(error instanceof Error ? error.message : "保存失败"); }
  };

  const saveParameters = async (key: "anytls" | "trojan") => {
    if (!token || !configs?.raw) return;
    const values: Record<string, unknown> = key === "anytls"
      ? { "exp_config.any_tls_server.host": anytlsForm.host, "exp_config.any_tls_server.port": anytlsForm.port, "exp_config.any_tls_server.user": anytlsForm.user, "exp_config.any_tls_server.cert_file": anytlsForm.cert_file, "exp_config.any_tls_server.key_file": anytlsForm.key_file, "exp_config.any_tls_server.dial_timeout": anytlsForm.dial_timeout }
      : { "exp_config.trojan_server": trojanPath };
    if (key === "anytls" && anytlsForm.password) values["exp_config.any_tls_server.password"] = anytlsForm.password;
    try { await saveManagementConfig({ version: configs.raw.version, values }, token); await refresh(token); setMessage(`${key === "anytls" ? "AnyTLS" : "Trojan"} 参数已保存，点击配置页应用`); }
    catch (error) { setMessage(error instanceof Error ? error.message : "保存失败"); }
  };

  return (
    <DashboardShell title={t("experiments.title")} subtitle={t("experiments.subtitle")}>
      <DataState isLoading={isLoading && experiments.length === 0} error={error} onRetry={() => token && void refresh(token)} />
      {message ? <p className="text-secondary rounded-xl border border-[var(--border)] px-3 py-2 text-sm">{message}</p> : null}

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

            {feature.key === "anytls" ? <div className="grid gap-2 sm:grid-cols-2"><label className="text-secondary text-xs">监听地址<input value={anytlsForm.host} onChange={(event) => setAnytlsForm((value) => ({ ...value, host: event.target.value }))} className="panel-soft text-primary mt-1 w-full rounded-lg border border-[var(--border)] px-2 py-1.5" /></label><label className="text-secondary text-xs">监听端口<input type="number" value={anytlsForm.port} onChange={(event) => setAnytlsForm((value) => ({ ...value, port: Number(event.target.value) }))} className="panel-soft text-primary mt-1 w-full rounded-lg border border-[var(--border)] px-2 py-1.5" /></label><label className="text-secondary text-xs">用户<input value={anytlsForm.user} onChange={(event) => setAnytlsForm((value) => ({ ...value, user: event.target.value }))} className="panel-soft text-primary mt-1 w-full rounded-lg border border-[var(--border)] px-2 py-1.5" /></label><label className="text-secondary text-xs">新密码（留空保持）<input type="password" value={anytlsForm.password} onChange={(event) => setAnytlsForm((value) => ({ ...value, password: event.target.value }))} className="panel-soft text-primary mt-1 w-full rounded-lg border border-[var(--border)] px-2 py-1.5" /></label><label className="text-secondary text-xs">证书文件<input value={anytlsForm.cert_file} onChange={(event) => setAnytlsForm((value) => ({ ...value, cert_file: event.target.value }))} className="panel-soft text-primary mt-1 w-full rounded-lg border border-[var(--border)] px-2 py-1.5" /></label><label className="text-secondary text-xs">密钥文件<input value={anytlsForm.key_file} onChange={(event) => setAnytlsForm((value) => ({ ...value, key_file: event.target.value }))} className="panel-soft text-primary mt-1 w-full rounded-lg border border-[var(--border)] px-2 py-1.5" /></label><label className="text-secondary text-xs">拨号超时（秒）<input type="number" value={anytlsForm.dial_timeout} onChange={(event) => setAnytlsForm((value) => ({ ...value, dial_timeout: Number(event.target.value) }))} className="panel-soft text-primary mt-1 w-full rounded-lg border border-[var(--border)] px-2 py-1.5" /></label></div> : <label className="text-secondary block text-xs">Trojan 独立配置文件<input value={trojanPath} onChange={(event) => setTrojanPath(event.target.value)} className="panel-soft text-primary mt-1 w-full rounded-lg border border-[var(--border)] px-2 py-1.5" /></label>}
            <button type="button" onClick={() => void saveParameters(feature.key)} className="w-full rounded-xl border border-[var(--border)] px-3 py-2 text-sm transition hover:border-emerald-400/50">保存 {feature.key === "anytls" ? "AnyTLS" : "Trojan"} 参数</button>

            <button
              type="button"
              onClick={() => void toggle(feature.key, feature.enabled)}
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
