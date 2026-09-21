"use client";

import { LoaderCircle, Play, Server, Square } from "lucide-react";
import { useState } from "react";

import { StatusPill } from "@/components/common/status-pill";
import { usePreferences } from "@/lib/preferences/preferences-context";
import type { DomainConnection } from "@/types/gateway";
import { setDomainState } from "@/lib/api/gateway";
import { useAuth } from "@/lib/auth/auth-context";

interface DomainCardProps {
  domain: DomainConnection;
}

export function DomainCard({ domain }: DomainCardProps) {
  const { t } = usePreferences();
  const { token } = useAuth();
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");
  const canStop = domain.status !== "offline";
  const toggle = async () => {
    if (!token) return;
    setPending(true); setMessage("");
    try { await setDomainState(domain.domain, canStop ? "stop" : "start", token); setMessage(canStop ? "已发送停止请求" : "已发送启动请求"); }
    catch (error) { setMessage(error instanceof Error ? error.message : "操作失败"); }
    finally { setPending(false); }
  };

  return (
    <article className="panel rounded-2xl border p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-lg font-semibold">{domain.domain}</p>
          <p className="text-secondary mt-1 text-xs">
            最近探测：{domain.lastHeartbeat || "尚未探测"}
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
