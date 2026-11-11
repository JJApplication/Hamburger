"use client";

import { LoaderCircle, Play, Radar, Server, Square } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { StatusPill } from "@/components/common/status-pill";
import { usePreferences, type MessageKey } from "@/lib/preferences/preferences-context";
import type { DomainConnection, DomainHealthStatus } from "@/types/gateway";
import { probeDomain, setDomainState } from "@/lib/api/gateway";
import { useAuth } from "@/lib/auth/auth-context";
import { useGatewayStore } from "@/store/gateway-store";

const PROBE_COOLDOWN_MS = 1000;

const probeStatusKey: Record<DomainHealthStatus, MessageKey> = {
  online: "domain.status.online",
  warning: "domain.status.warning",
  offline: "domain.status.offline",
  unknown: "domain.status.unknown",
};

interface DomainCardProps {
  domain: DomainConnection;
}

export function DomainCard({ domain }: DomainCardProps) {
  const { t } = usePreferences();
  const { token } = useAuth();
  const refreshDomains = useGatewayStore((state) => state.refreshDomains);
  const [pending, setPending] = useState(false);
  const [probing, setProbing] = useState(false);
  const [probeCooling, setProbeCooling] = useState(false);
  const [message, setMessage] = useState("");
  const probeLock = useRef(false);
  const probeTimer = useRef<number | null>(null);
  const canStop = domain.status !== "offline";
  const toggle = async () => {
    if (!token) return;
    setPending(true); setMessage("");
    try { await setDomainState(domain.domain, canStop ? "stop" : "start", token); setMessage(canStop ? "已发送停止请求" : "已发送启动请求"); }
    catch (error) { setMessage(error instanceof Error ? error.message : "操作失败"); }
    finally { setPending(false); }
  };
  const probe = async () => {
    if (!token || probeLock.current) return;
    probeLock.current = true;
    setProbing(true);
    setMessage("");
    try {
      const result = await probeDomain(domain.domain, token);
      await refreshDomains(token);
      setMessage(`${t("domain.probeDone")}：${t(probeStatusKey[result.status])}`);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "探测失败");
    } finally {
      setProbing(false);
      setProbeCooling(true);
      probeTimer.current = window.setTimeout(() => {
        probeLock.current = false;
        setProbeCooling(false);
      }, PROBE_COOLDOWN_MS);
    }
  };

  useEffect(() => () => {
    if (probeTimer.current != null) window.clearTimeout(probeTimer.current);
  }, []);

  return (
    <article className="panel rounded-2xl border p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-lg font-semibold">{domain.domain}</p>
          <p className="text-secondary mt-1 text-xs">
            最近探测：{domain.lastHeartbeat || "尚未探测"}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <StatusPill status={domain.status} />
          <button
            type="button"
            onClick={() => void probe()}
            disabled={probing || probeCooling}
            className="flex items-center gap-1.5 rounded-full border border-sky-400/40 bg-sky-500/10 px-2.5 py-0.5 text-xs text-sky-300 transition hover:bg-sky-500/20 disabled:cursor-not-allowed disabled:opacity-50 [html.theme-light_&]:text-sky-700"
          >
            {probing ? <LoaderCircle size={12} className="animate-spin" /> : <Radar size={12} />}
            {probing ? t("domain.probing") : t("domain.probe")}
          </button>
        </div>
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
      <div className="mt-4 flex items-center justify-between gap-3 border-t border-[var(--border)] pt-3">
        <span className="text-secondary text-xs">连接数按域名关联统计，不能直接相加为网关总数</span>
        <button type="button" onClick={() => void toggle()} disabled={pending} className={`flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs transition ${canStop ? "border-rose-400/40 text-rose-300 hover:bg-rose-500/10" : "border-emerald-400/40 text-emerald-300 hover:bg-emerald-500/10"}`}>
          {pending ? <LoaderCircle size={14} className="animate-spin" /> : canStop ? <Square size={13} /> : <Play size={13} />}{pending ? "处理中" : canStop ? "停止服务" : "启动服务"}
        </button>
      </div>
      {message ? <p className="text-secondary mt-2 text-right text-xs">{message}</p> : null}
    </article>
  );
}
