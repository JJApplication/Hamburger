"use client";
/* eslint-disable react-hooks/set-state-in-effect */

import { useEffect, useState } from "react";
import { Activity, AlertCircle, Gauge, Globe2, PlugZap, RefreshCw } from "lucide-react";
import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

import { DataState } from "@/components/common/data-state";
import { Panel } from "@/components/common/panel";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { fetchGeo, fetchStat } from "@/lib/api/gateway";
import { useAuth } from "@/lib/auth/auth-context";
import { useGatewayBootstrap } from "@/lib/hooks/use-gateway-bootstrap";
import { usePreferences } from "@/lib/preferences/preferences-context";
import { useGatewaySelectors, useGatewayStore } from "@/store/gateway-store";
import type { StatResponse, StatPoint } from "@/types/gateway";

const ranges = ["1h", "5h", "24h", "7d", "30d"] as const;
type Range = typeof ranges[number];
type Tab = "overview" | "traffic" | "resources" | "geo";
type Capability = (key: string) => boolean;

const number = (value: number | undefined) => new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 1 }).format(value ?? 0);
const bytes = (value: number | undefined) => {
  const size = value ?? 0;
  if (size < 1024) return `${size} B`;
  if (size < 1024 ** 2) return `${(size / 1024).toFixed(1)} KB`;
  if (size < 1024 ** 3) return `${(size / 1024 ** 2).toFixed(1)} MB`;
  return `${(size / 1024 ** 3).toFixed(1)} GB`;
};

export default function Home() {
  useGatewayBootstrap();
  const { t } = usePreferences();
  const { token } = useAuth();
  const overview = useGatewayStore(useGatewaySelectors.overviewSummary);
  const data = useGatewayStore((state) => state.data);
  const refresh = useGatewayStore((state) => state.refresh);
  const isLoading = useGatewayStore((state) => state.isLoading);
  const error = useGatewayStore((state) => state.error);
  const [range, setRange] = useState<Range>("1h");
  const [tab, setTab] = useState<Tab>("overview");
  const [selectedDomain, setSelectedDomain] = useState("");
  const [stat, setStat] = useState<StatResponse | null>(null);
  const [geo, setGeo] = useState<Record<string, number>>({});
  const [statError, setStatError] = useState("");
  const [refreshing, setRefreshing] = useState(false);
  const [statRefreshKey, setStatRefreshKey] = useState(0);

  useEffect(() => {
    if (!token) return;
    const controller = new AbortController();
    setStatError("");
    const load = () => void fetchStat(range, selectedDomain || undefined, token, controller.signal)
      .then(setStat)
      .catch((reason) => {
        if (!controller.signal.aborted) setStatError(reason instanceof Error ? reason.message : "统计数据加载失败");
      });
    load();
    const timer = window.setInterval(load, 30_000);
    return () => { controller.abort(); window.clearInterval(timer); };
  }, [range, selectedDomain, statRefreshKey, token]);

  useEffect(() => {
    if (!token || tab !== "geo") return;
    const controller = new AbortController();
    const load = () => void fetchGeo(token, controller.signal).then(setGeo).catch(() => undefined);
    load();
    const timer = window.setInterval(load, 30_000);
    return () => { controller.abort(); window.clearInterval(timer); };
  }, [tab, token]);

  const current = stat ?? data?.stat;
  const capability: Capability = (key) => current?.meta.capabilities?.[key] === true;
  const requestSeries = current?.series.requests ?? [];
  const trafficSeries = current?.series.traffic ?? [];
  const systemSeries = current?.series.system ?? [];
  const processSeries = current?.series.process ?? [];
  const gcSeries = current?.series.gc ?? [];
  const topGeo = Object.entries(geo).sort((a, b) => b[1] - a[1]).slice(0, 10);

  return (
    <DashboardShell title={t("overview.title")} subtitle="网关健康、流量与资源观测">
      <DataState isLoading={isLoading && !overview} error={error} onRetry={() => token && void refresh(token)} />
      {overview ? (
        <div className="space-y-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex flex-wrap gap-2" role="tablist" aria-label="统计视图">
              {(["overview", "traffic", "resources", "geo"] as Tab[]).map((item) => (
                <button key={item} type="button" role="tab" aria-selected={tab === item} onClick={() => setTab(item)} className={`rounded-full border px-3 py-1.5 text-sm transition ${tab === item ? "border-emerald-400/60 bg-emerald-500/15 text-emerald-300" : "text-secondary border-[var(--border)]"}`}>
                  {item === "overview" ? "概览" : item === "traffic" ? "请求与流量" : item === "resources" ? "资源与 GC" : "全球来源"}
                </button>
              ))}
            </div>
            <div className="flex items-center gap-2">
              <div className="flex rounded-xl border border-[var(--border)] p-1">
                {ranges.map((item) => <button key={item} type="button" onClick={() => setRange(item)} className={`rounded-lg px-2.5 py-1 text-xs ${range === item ? "bg-emerald-500/20 text-emerald-300" : "text-secondary"}`}>{item}</button>)}
              </div>
              <button type="button" className="panel-soft text-secondary rounded-xl border border-[var(--border)] p-2" onClick={() => { if (!token) return; setRefreshing(true); setStatRefreshKey((value) => value + 1); void refresh(token).finally(() => setRefreshing(false)); }} aria-label="刷新">
                <RefreshCw size={16} className={refreshing ? "animate-spin" : ""} />
              </button>
            </div>
          </div>
          {statError ? <p className="rounded-xl border border-rose-400/30 bg-rose-500/10 px-3 py-2 text-sm text-rose-300">{statError}</p> : null}
          {tab === "overview" ? <OverviewTab overview={overview} current={current} requestSeries={requestSeries} data={data} /> : null}
          {tab === "traffic" ? <TrafficTab current={current} requestSeries={requestSeries} trafficSeries={trafficSeries} selectedDomain={selectedDomain} setSelectedDomain={setSelectedDomain} capability={capability} /> : null}
          {tab === "resources" ? <ResourcesTab requestSeries={requestSeries} systemSeries={systemSeries} processSeries={processSeries} gcSeries={gcSeries} capability={capability} /> : null}
          {tab === "geo" ? <GeoTab geo={topGeo} data={data} /> : null}
        </div>
      ) : null}
    </DashboardShell>
  );
}

function OverviewTab({ overview, current, requestSeries, data }: { overview: { activeDomains: number; totalMappings: number }; current: StatResponse | undefined; requestSeries: StatPoint[]; data: ReturnType<typeof useGatewayStore.getState>["data"] }) {
  const summary = current?.summary;
  const kpis = [
    { label: "当前连接", value: number((data?.connections?.gateway?.active ?? 0) + (data?.connections?.gateway?.idle ?? 0)), icon: PlugZap },
    { label: "总请求", value: number(summary?.total_requests), icon: Activity },
    { label: "错误率", value: `${((summary?.error_rate ?? 0) * 100).toFixed(2)}%`, icon: AlertCircle },
    { label: "P95 延迟", value: `${(summary?.latency.p95_ms ?? 0).toFixed(1)} ms`, icon: Gauge },
  ];
  return (
    <>
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">{kpis.map(({ label, value, icon: Icon }) => <Panel key={label}><div className="flex items-center justify-between"><p className="text-secondary text-sm">{label}</p><Icon size={16} className="text-emerald-300" /></div><p className="mt-3 text-3xl font-semibold">{value}</p></Panel>)}</section>
      <section className="grid gap-4 xl:grid-cols-[2fr_1fr]"><Panel><div className="flex items-center justify-between"><h2 className="text-lg font-semibold">请求趋势</h2><span className="text-secondary text-xs">每 {current?.meta.bucket_seconds ?? 60} 秒</span></div><Chart data={requestSeries.map((point) => ({ ...point, total: Number(point.total ?? 0), errors: Number(point.errors ?? 0) }))} lines={[{ key: "total", color: "#36d399", name: "总请求" }, { key: "errors", color: "#fb7185", name: "错误" }]} /></Panel><Panel className="space-y-3"><h2 className="text-lg font-semibold">运行摘要</h2><Metric label="活动域名" value={number(overview.activeDomains)} /><Metric label="映射条目" value={number(overview.totalMappings)} /><Metric label="统计窗口" value={current ? `${current.meta.start_time} — ${current.meta.end_time}` : "暂无"} /></Panel></section>
    </>
  );
}

function TrafficTab({ current, requestSeries, trafficSeries, selectedDomain, setSelectedDomain, capability }: { current: StatResponse | undefined; requestSeries: StatPoint[]; trafficSeries: StatPoint[]; selectedDomain: string; setSelectedDomain: (value: string) => void; capability: Capability }) {
  const summary = current?.summary;
  return (
    <section className="grid gap-4 xl:grid-cols-2">
      <Panel><h2 className="text-lg font-semibold">请求趋势与状态码</h2><Chart data={requestSeries.map((point) => ({ ...point, frontend: Number(point.frontend ?? 0), backend: Number(point.backend ?? 0), status_4xx: Number(point.status_4xx ?? 0), status_5xx: Number(point.status_5xx ?? 0) }))} lines={[{ key: "frontend", color: "#60a5fa", name: "前端" }, { key: "backend", color: "#36d399", name: "后端" }, { key: "status_4xx", color: "#fbbf24", name: "4xx" }, { key: "status_5xx", color: "#fb7185", name: "5xx" }]} /></Panel>
      <Panel><h2 className="text-lg font-semibold">程序流量</h2>{capability("program_traffic") ? <Chart data={trafficSeries.map((point) => ({ ...point, request: Number(point.request_bytes ?? 0), response: Number(point.response_bytes ?? 0) }))} lines={[{ key: "request", color: "#60a5fa", name: "请求字节" }, { key: "response", color: "#36d399", name: "响应字节" }]} formatter={bytes} /> : <Unavailable />}</Panel>
      <Panel><h2 className="text-lg font-semibold">状态码分布</h2><div className="mt-4 grid grid-cols-5 gap-2">{(["1xx", "2xx", "3xx", "4xx", "5xx"] as const).map((code) => <Metric key={code} label={code} value={number(summary?.status?.[code])} />)}</div></Panel>
      <Panel><h2 className="text-lg font-semibold">累计统计</h2><div className="mt-4 grid grid-cols-2 gap-2"><Metric label="累计请求" value={number(current?.total)} /><Metric label="API" value={number(current?.api)} /><Metric label="静态" value={number(current?.static)} /><Metric label="失败" value={number(current?.fail)} /><Metric label="今日" value={number(current?.today)} /></div></Panel>
      <Panel className="xl:col-span-2"><div className="flex flex-wrap items-center justify-between gap-2"><h2 className="text-lg font-semibold">域名排行与详情</h2>{selectedDomain ? <button type="button" className="text-secondary text-xs" onClick={() => setSelectedDomain("")}>返回全部</button> : null}</div>{selectedDomain && current?.domain_series?.length ? <Chart data={current.domain_series} lines={[{ key: "requests", color: "#36d399", name: "请求" }, { key: "errors", color: "#fb7185", name: "错误" }]} /> : <div className="mt-4 space-y-2">{(current?.domains ?? []).slice(0, 10).map((item) => <button type="button" key={item.domain} onClick={() => setSelectedDomain(item.domain)} className="panel-soft flex w-full items-center justify-between rounded-xl px-3 py-2 text-left text-sm"><span>{item.domain}</span><span className="text-secondary">{number(item.requests)} 请求 · {bytes(item.request_bytes + item.response_bytes)}</span></button>)}{!current?.domains?.length ? <Unavailable label="暂无域名统计" /> : null}</div>}</Panel>
      <Panel className="xl:col-span-2"><h2 className="text-lg font-semibold">窗口摘要</h2><div className="mt-4 grid gap-3 sm:grid-cols-3"><Metric label="总流量" value={bytes(summary?.total_traffic.total_bytes)} /><Metric label="RPS" value={(summary?.rps ?? 0).toFixed(2)} /><Metric label="4xx / 5xx" value={`${number(summary?.status?.["4xx"])} / ${number(summary?.status?.["5xx"])}`} /></div></Panel>
    </section>
  );
}

function ResourcesTab({ requestSeries, systemSeries, processSeries, gcSeries, capability }: { requestSeries: StatPoint[]; systemSeries: StatPoint[]; processSeries: StatPoint[]; gcSeries: StatPoint[]; capability: Capability }) {
  return (
    <section className="grid gap-4 xl:grid-cols-2">
      <Panel><h2 className="text-lg font-semibold">系统 CPU / 内存</h2>{capability("system_cpu") || capability("system_memory") ? <Chart data={systemSeries.map((point) => ({ ...point, cpu: Number(point.cpu_percent ?? 0), memory: Number(point.memory_percent ?? 0) }))} lines={[{ key: "cpu", color: "#36d399", name: "CPU %" }, { key: "memory", color: "#60a5fa", name: "内存 %" }]} /> : <Unavailable />}</Panel>
      <Panel><h2 className="text-lg font-semibold">进程 CPU / 内存</h2>{capability("process_cpu") || capability("process_memory") ? <Chart data={processSeries.map((point) => ({ ...point, cpu: Number(point.cpu_percent ?? 0), memory: Number(point.memory_percent ?? 0) }))} lines={[{ key: "cpu", color: "#fbbf24", name: "CPU %" }, { key: "memory", color: "#a78bfa", name: "内存 %" }]} /> : <Unavailable />}</Panel>
      <Panel><h2 className="text-lg font-semibold">GC 周期与暂停</h2>{capability("runtime_gc") ? <Chart data={gcSeries.map((point) => ({ ...point, cycles: Number(point.cycles ?? 0), pause: Number(point.pause_total_ms ?? 0) }))} lines={[{ key: "cycles", color: "#36d399", name: "周期" }, { key: "pause", color: "#fb7185", name: "暂停 ms" }]} /> : <Unavailable />}</Panel>
      <Panel><h2 className="text-lg font-semibold">延迟</h2><Chart data={requestSeries.map((point) => ({ ...point, avg: Number(point.avg_latency_ms ?? 0), p95: Number(point.p95_latency_ms ?? 0), max: Number(point.max_latency_ms ?? 0) }))} lines={[{ key: "avg", color: "#60a5fa", name: "平均" }, { key: "p95", color: "#fbbf24", name: "P95" }, { key: "max", color: "#fb7185", name: "最大" }]} formatter={(value) => `${number(value)} ms`} /></Panel>
    </section>
  );
}

function GeoTab({ geo, data }: { geo: Array<[string, number]>; data: ReturnType<typeof useGatewayStore.getState>["data"] }) {
  return <section className="grid gap-4 xl:grid-cols-[1fr_1.5fr]"><Panel><div className="flex items-center gap-2"><Globe2 size={18} className="text-emerald-300" /><h2 className="text-lg font-semibold">全球请求来源</h2></div><p className="text-secondary mt-1 text-sm">累计 GEO 数据，不随时间窗口切换。</p><div className="mt-4 space-y-2">{geo.length ? geo.map(([country, value], index) => <div key={country} className="panel-soft flex items-center justify-between rounded-xl px-3 py-2 text-sm"><span><span className="text-tertiary mr-2">{index + 1}</span>{country}</span><strong>{number(value)}</strong></div>) : <Unavailable label="暂无 GEO 数据" />}</div></Panel><Panel><h2 className="text-lg font-semibold">连接快照</h2><div className="mt-4 grid gap-3 sm:grid-cols-2"><ConnectionCard title="Gateway" value={data?.connections?.gateway} /><ConnectionCard title="Front" value={data?.connections?.front} /></div></Panel></section>;
}

function Metric({ label, value }: { label: string; value: string }) { return <div className="panel-soft rounded-xl p-3"><p className="text-secondary text-xs">{label}</p><p className="mt-1 text-xl font-semibold">{value}</p></div>; }
function Unavailable({ label = "不可用" }: { label?: string }) { return <p className="text-secondary flex h-64 items-center justify-center text-sm">{label}</p>; }
function ConnectionCard({ title, value }: { title: string; value: { active: number; idle: number; new: number; closed: number; hijacked: number } | undefined }) { return <div className="panel-soft rounded-xl p-3"><div className="flex items-center justify-between"><span className="text-secondary text-sm">{title}</span><PlugZap size={15} className="text-emerald-300" /></div><p className="mt-2 text-2xl font-semibold">{number((value?.active ?? 0) + (value?.idle ?? 0))}</p><div className="text-secondary mt-2 grid grid-cols-3 gap-2 text-xs"><span>活动 {number(value?.active)}</span><span>空闲 {number(value?.idle)}</span><span>关闭 {number(value?.closed)}</span></div></div>; }
function Chart({ data, lines, formatter = number }: { data: Array<Record<string, unknown>>; lines: Array<{ key: string; color: string; name: string }>; formatter?: (value: number) => string }) { return <div className="mt-4 h-64"><ResponsiveContainer width="100%" height="100%"><LineChart data={data} margin={{ top: 8, right: 12, left: 4, bottom: 0 }}><CartesianGrid stroke="var(--border)" strokeDasharray="3 3" /><XAxis dataKey="timestamp" tickFormatter={(value) => { const date = new Date(String(value)); return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }); }} tick={{ fill: "var(--text-secondary)", fontSize: 11 }} tickLine={false} axisLine={false} /><YAxis tick={{ fill: "var(--text-secondary)", fontSize: 11 }} tickLine={false} axisLine={false} /><Tooltip contentStyle={{ background: "var(--panel)", borderColor: "var(--border)", borderRadius: 12 }} formatter={(value) => formatter(Number(value))} />{lines.map((line) => <Line key={line.key} type="monotone" dataKey={line.key} name={line.name} stroke={line.color} strokeWidth={2.5} dot={false} connectNulls />)}</LineChart></ResponsiveContainer></div>; }
