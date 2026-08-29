import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Activity, AlertTriangle, ArrowDownUp, Gauge, HardDrive, Moon, Network, RefreshCw, Search, Sun, X } from "lucide-react";
import { fetchGeo, fetchStat, USE_MOCK } from "./api";
import { GlobeOverview } from "./components/GlobeOverview";
import { MonoRoundedAreaChart, MonoRoundedBarChart, MonoRoundedKpiCardChart, MonoRoundedLineChart } from "./components/MonoCharts";
import { Panel, SectionMessage } from "./components/Panel";
import { formatBytes, formatDateTime, formatMilliseconds, formatNumber, formatPercent, formatRate } from "./lib/format";
import { filterDomains, RANGES } from "./lib/stat-utils";
import type { ConnectionSnapshot, DomainSeriesPoint, GeoData, Range, StatResponse } from "./types/stat";

const colors = {
  accent: "#1d9d78",
  frontend: "#4aa9e8",
  backend: "#9b79e8",
  errors: "#e9786d",
  amber: "#dca84c",
  ink: "#1f2a27",
};

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === "AbortError";
}

function usePrefersReducedMotion(): boolean {
  const [reduced, setReduced] = useState(() => typeof window !== "undefined" && window.matchMedia?.("(prefers-reduced-motion: reduce)").matches === true);
  useEffect(() => {
    const query = window.matchMedia?.("(prefers-reduced-motion: reduce)");
    if (!query) return;
    const update = () => setReduced(query.matches);
    query.addEventListener?.("change", update);
    return () => query.removeEventListener?.("change", update);
  }, []);
  return reduced;
}

function useTheme(): ["light" | "dark", () => void] {
  const [theme, setTheme] = useState<"light" | "dark">(() => {
    if (typeof window === "undefined") return "light";
    return window.localStorage.getItem("hamburger-stat-theme") === "dark" ? "dark" : "light";
  });
  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    window.localStorage.setItem("hamburger-stat-theme", theme);
  }, [theme]);
  return [theme, () => setTheme((value) => value === "light" ? "dark" : "light")];
}

export default function App() {
  const [range, setRange] = useState<Range>("1h");
  const [stat, setStat] = useState<StatResponse | null>(null);
  const [geo, setGeo] = useState<GeoData>({});
  const [selectedDomain, setSelectedDomain] = useState("");
  const [domainQuery, setDomainQuery] = useState("");
  const [detailSeries, setDetailSeries] = useState<DomainSeriesPoint[]>([]);
  const [resourceMode, setResourceMode] = useState<"average" | "peak">("average");
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [geoLoading, setGeoLoading] = useState(true);
  const [error, setError] = useState("");
  const [geoError, setGeoError] = useState("");
  const [lastUpdated, setLastUpdated] = useState("");
  const [theme, toggleTheme] = useTheme();
  const reducedMotion = usePrefersReducedMotion();
  const statController = useRef<AbortController | null>(null);
  const geoController = useRef<AbortController | null>(null);
  const requestId = useRef(0);

  const loadStat = useCallback(async (nextRange: Range, nextDomain: string, silent = false) => {
    statController.current?.abort();
    const controller = new AbortController();
    statController.current = controller;
    const currentRequest = ++requestId.current;
    if (silent) setRefreshing(true); else setLoading(true);
    setError("");
    try {
      const main = await fetchStat(nextRange, undefined, controller.signal);
      let detail: DomainSeriesPoint[] = [];
      if (nextDomain) {
        try {
          const selected = await fetchStat(nextRange, nextDomain, controller.signal);
          detail = selected.domain_series ?? [];
        } catch (detailError) {
          if (isAbortError(detailError)) throw detailError;
          setError("域名详情暂时不可用，但总览数据仍已更新");
        }
      }
      if (currentRequest !== requestId.current) return;
      setStat(main);
      setDetailSeries(detail);
      setLastUpdated(new Date().toISOString());
    } catch (loadError) {
      if (isAbortError(loadError) || currentRequest !== requestId.current) return;
      setError(loadError instanceof Error ? loadError.message : "统计数据加载失败");
    } finally {
      if (currentRequest === requestId.current) {
        setLoading(false);
        setRefreshing(false);
      }
    }
  }, []);

  const loadGeo = useCallback(async () => {
    geoController.current?.abort();
    const controller = new AbortController();
    geoController.current = controller;
    setGeoLoading(true);
    setGeoError("");
    try {
      setGeo(await fetchGeo(controller.signal));
    } catch (loadError) {
      if (isAbortError(loadError)) return;
      setGeoError(loadError instanceof Error ? loadError.message : "GEO 数据加载失败");
    } finally {
      if (!controller.signal.aborted) setGeoLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadStat(range, selectedDomain);
    const timer = window.setInterval(() => void loadStat(range, selectedDomain, true), 30_000);
    return () => window.clearInterval(timer);
  }, [loadStat, range, selectedDomain]);

  useEffect(() => {
    void loadGeo();
    const timer = window.setInterval(() => void loadGeo(), 30_000);
    return () => window.clearInterval(timer);
  }, [loadGeo]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (target?.tagName === "INPUT" || target?.tagName === "TEXTAREA") return;
      if (event.key.toLowerCase() === "r") {
        event.preventDefault();
        void loadStat(range, selectedDomain);
      }
      if (event.key === "Escape" && selectedDomain) setSelectedDomain("");
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [loadStat, range, selectedDomain]);

  const filteredDomains = useMemo(() => filterDomains(stat?.domains ?? [], domainQuery), [domainQuery, stat?.domains]);
  const summary = stat?.summary;
  const requestSeries = stat?.series.requests ?? [];
  const systemSeries = stat?.series.system ?? [];
  const processSeries = stat?.series.process ?? [];
  const trafficSeries = stat?.series.traffic ?? [];
  const hasData = Boolean(summary?.total_requests);
  const missingCapabilities = useMemo(() => {
    if (!stat) return [];
    const names: Record<string, string> = {
      system_cpu: "系统 CPU", system_memory: "系统内存", system_network: "系统网络",
      system_disk_io: "系统磁盘 IO", process_cpu: "进程 CPU", process_memory: "进程内存", process_disk_io: "进程磁盘 IO",
    };
    return Object.entries(names).filter(([key]) => stat.meta.capabilities[key] === false).map(([, name]) => name);
  }, [stat]);

  const statusData = summary ? [{ timestamp: "当前窗口", "1xx": summary.status["1xx"], "2xx": summary.status["2xx"], "3xx": summary.status["3xx"], "4xx": summary.status["4xx"], "5xx": summary.status["5xx"] }] : [];
  const systemResourceData = systemSeries.map((point) => ({
    ...point,
    cpu: resourceMode === "average" ? point.cpu_percent : point.cpu_peak_percent,
    memory: resourceMode === "average" ? point.memory_percent : point.memory_peak_percent,
  }));
  const processResourceData = processSeries.map((point) => ({
    ...point,
    cpu: resourceMode === "average" ? point.cpu_percent : point.cpu_peak_percent,
    memory: resourceMode === "average" ? point.memory_percent : point.memory_peak_percent,
  }));

  return (
    <div className="stat-app">
      <header className="topbar">
        <div className="brand-lockup"><div className="brand-mark">H</div><div><strong>Hamburger Stat</strong><span>独立网关监控</span></div></div>
        <div className="topbar-actions"><span className={`live-pill ${USE_MOCK ? "mock-pill" : ""}`}><i />{USE_MOCK ? "模拟数据" : "实时历史"}</span><button className="icon-button" type="button" onClick={toggleTheme} aria-label={theme === "light" ? "切换深色主题" : "切换浅色主题"}>{theme === "light" ? <Moon size={17} /> : <Sun size={17} />}</button><button className="refresh-button" type="button" onClick={() => { void loadStat(range, selectedDomain); void loadGeo(); }} disabled={loading || refreshing}><RefreshCw size={15} className={refreshing ? "spin" : ""} />刷新</button></div>
      </header>

      <main className="page-shell">
        <section className="hero-row">
          <div><div className="eyebrow">GATEWAY OBSERVABILITY</div><h1>网关可观测性总览</h1><p>请求、资源、流量和全球来源，按同一条时间线持续更新。</p></div>
          <div className="range-controls" aria-label="统计时间窗口">{RANGES.map((item) => <button type="button" key={item} className={range === item ? "range-button active" : "range-button"} aria-pressed={range === item} onClick={() => setRange(item)}>{item}</button>)}</div>
        </section>

        {error && <div className="notice error-notice" role="alert"><AlertTriangle size={17} /><span>{error}</span><button type="button" onClick={() => setError("")} aria-label="关闭错误提示"><X size={15} /></button></div>}
        {loading && !stat ? <SectionMessage kind="loading" title="正在加载统计数据" detail="首次读取可能需要等待 SQLite 历史存储响应。" /> : stat ? <>
          <section className="legacy-stat-panel" aria-label="累计 Stat 数据库统计">
            <div className="legacy-stat-heading"><div><div className="eyebrow">LEGACY STAT DATABASE</div><h2>累计统计</h2></div><span className="panel-caption">原 Stat 数据库 · 不受当前时间窗口影响</span></div>
            <div className="legacy-stat-grid">
              <LegacyStatCard label="总请求数" value={stat.total} color={colors.accent} />
              <LegacyStatCard label="失败请求数" value={stat.fail} color={colors.errors} />
              <LegacyStatCard label="前端请求数" value={stat.static} color={colors.frontend} />
              <LegacyStatCard label="后端请求数" value={stat.api} color={colors.backend} />
            </div>
          </section>

          <section className="kpi-grid" aria-label="当前窗口关键指标">
            <MonoRoundedKpiCardChart label="总请求" value={formatNumber(summary?.total_requests ?? 0)} hint={`${formatNumber(summary?.frontend_requests ?? 0)} 前端 · ${formatNumber(summary?.backend_requests ?? 0)} 后端`} data={requestSeries.map((point) => ({ value: point.total }))} color={colors.accent} />
            <MonoRoundedKpiCardChart label="RPS" value={formatRate(summary?.rps ?? 0)} hint={`窗口 ${stat.meta.range}`} data={requestSeries.map((point) => ({ value: point.rps }))} color={colors.frontend} />
            <MonoRoundedKpiCardChart label="错误率" value={formatPercent(summary?.error_rate ?? 0)} hint={`${formatNumber(summary?.error_requests ?? 0)} 个 4xx / 5xx`} data={requestSeries.map((point) => ({ value: point.errors }))} color={colors.errors} />
            <MonoRoundedKpiCardChart label="P95 延迟" value={formatMilliseconds(summary?.latency.p95_ms ?? 0)} hint={`平均 ${formatMilliseconds(summary?.latency.avg_ms ?? 0)} · 最大 ${formatMilliseconds(summary?.latency.max_ms ?? 0)}`} data={requestSeries.map((point) => ({ value: point.p95_latency_ms }))} color={colors.amber} />
            <MonoRoundedKpiCardChart label="总流量" value={formatBytes(summary?.total_traffic.total_bytes ?? 0)} hint={`${formatBytes(summary?.total_traffic.request_bytes ?? 0)} 收 · ${formatBytes(summary?.total_traffic.response_bytes ?? 0)} 发`} data={trafficSeries.map((point) => ({ value: point.request_bytes + point.response_bytes }))} color={colors.backend} />
          </section>

          <Panel title="全球请求来源" eyebrow="GEO · 累计总览" action={<span className="panel-live">{geoLoading ? "同步中" : "每 30 秒刷新"}</span>}>
            {geoError ? <SectionMessage kind="error" title="GEO 数据加载失败" detail={geoError} /> : <GlobeOverview geo={geo} reducedMotion={reducedMotion} />}
          </Panel>

          {!hasData && <div className="notice empty-notice"><Activity size={17} /><span>当前时间窗口还没有历史请求数据；结构化图表会在第一批请求刷盘后自动出现。</span></div>}
          {missingCapabilities.length > 0 && <div className="notice partial-notice"><AlertTriangle size={17} /><span>部分系统指标不可用：{missingCapabilities.join("、")}。这些字段会以空值显示，不会被当作零。</span></div>}

          <section className="two-column-grid">
            <Panel title="请求趋势" eyebrow="REQUESTS" action={<span className="panel-caption">每桶 {formatNumber(stat.meta.bucket_seconds)} 秒</span>}>
              <MonoRoundedLineChart data={requestSeries} series={[{ dataKey: "frontend", name: "前端", color: colors.frontend }, { dataKey: "backend", name: "后端", color: colors.backend }, { dataKey: "errors", name: "错误", color: colors.errors }]} valueFormatter={formatNumber} />
            </Panel>
            <Panel title="状态码分布" eyebrow="HTTP STATUS">
              <MonoRoundedBarChart data={statusData} series={[{ dataKey: "1xx", name: "1xx", color: "#8bc8a8" }, { dataKey: "2xx", name: "2xx", color: colors.accent }, { dataKey: "3xx", name: "3xx", color: colors.frontend }, { dataKey: "4xx", name: "4xx", color: colors.amber }, { dataKey: "5xx", name: "5xx", color: colors.errors }]} valueFormatter={formatNumber} />
            </Panel>
          </section>

          <Panel title="延迟走势" eyebrow="LATENCY" action={<span className="panel-caption">近似直方图 P95</span>}>
            <MonoRoundedLineChart data={requestSeries} series={[{ dataKey: "avg_latency_ms", name: "平均", color: colors.frontend }, { dataKey: "p95_latency_ms", name: "P95", color: colors.amber }, { dataKey: "max_latency_ms", name: "最大", color: colors.errors }]} valueFormatter={(value) => formatMilliseconds(value)} />
          </Panel>

          <div className="panel-section-heading"><div><div className="eyebrow">SYSTEM & PROCESS</div><h2>资源曲线</h2></div><div className="segmented-control" role="group" aria-label="资源统计模式"><button type="button" className={resourceMode === "average" ? "active" : ""} onClick={() => setResourceMode("average")}>平均</button><button type="button" className={resourceMode === "peak" ? "active" : ""} onClick={() => setResourceMode("peak")}>峰值</button></div></div>
          <section className="two-column-grid">
            <Panel title="系统 CPU / 内存" eyebrow="SYSTEM RESOURCES" action={<Gauge size={18} /> }>
              <MonoRoundedLineChart data={systemResourceData} series={[{ dataKey: "cpu", name: "CPU %", color: colors.accent }, { dataKey: "memory", name: "内存 %", color: colors.frontend }]} valueFormatter={(value) => `${value.toFixed(0)}%`} />
            </Panel>
            <Panel title="进程 CPU / 内存 / IO" eyebrow="PROCESS RESOURCES" action={<Activity size={18} /> }>
              <MonoRoundedLineChart data={processResourceData} series={[{ dataKey: "cpu", name: "CPU %", color: colors.backend }, { dataKey: "memory", name: "内存 %", color: colors.amber }]} valueFormatter={(value) => `${value.toFixed(0)}%`} />
              <div className="sub-chart-label">进程磁盘读写</div>
              <MonoRoundedAreaChart data={processSeries} height={190} series={[{ dataKey: "disk_read_bytes", name: "磁盘读", color: colors.frontend }, { dataKey: "disk_write_bytes", name: "磁盘写", color: colors.errors }]} valueFormatter={formatBytes} />
            </Panel>
          </section>

          <section className="two-column-grid">
            <Panel title="系统网络 / 磁盘 IO" eyebrow="HOST IO" action={<div className="panel-icon-pair"><Network size={16} /><HardDrive size={16} /></div>}>
              <MonoRoundedAreaChart data={systemSeries} series={[{ dataKey: "network_rx_bytes", name: "网络收", color: colors.frontend }, { dataKey: "network_tx_bytes", name: "网络发", color: colors.accent }, { dataKey: "disk_read_bytes", name: "磁盘读", color: colors.amber }, { dataKey: "disk_write_bytes", name: "磁盘写", color: colors.errors }]} valueFormatter={formatBytes} />
            </Panel>
            <Panel title="程序前后端流量" eyebrow="PROGRAM PAYLOAD" action={<ArrowDownUp size={18} /> }>
              <MonoRoundedAreaChart data={trafficSeries} series={[{ dataKey: "frontend_request_bytes", name: "前端收", color: colors.frontend }, { dataKey: "frontend_response_bytes", name: "前端发", color: colors.accent }, { dataKey: "backend_request_bytes", name: "后端收", color: colors.backend }, { dataKey: "backend_response_bytes", name: "后端发", color: colors.amber }]} valueFormatter={formatBytes} />
            </Panel>
          </section>

          <Panel title="域名排行与详情" eyebrow="DOMAINS" action={<div className="domain-search"><Search size={15} /><input value={domainQuery} onChange={(event) => setDomainQuery(event.target.value)} placeholder="搜索域名" aria-label="搜索域名" />{domainQuery && <button type="button" onClick={() => setDomainQuery("")} aria-label="清除域名搜索"><X size={14} /></button>}</div>}>
            <div className="domain-layout">
              <div className="domain-table-wrap">
                {filteredDomains.length ? <table className="domain-table"><thead><tr><th>域名</th><th>请求</th><th>错误</th><th>流量</th></tr></thead><tbody>{filteredDomains.map((domain) => <tr key={domain.domain} className={selectedDomain === domain.domain ? "selected" : ""}><td><button type="button" className="domain-button" onClick={() => setSelectedDomain(domain.domain)}>{domain.domain}</button></td><td>{formatNumber(domain.requests)}</td><td className={domain.errors ? "danger-text" : "muted-text"}>{formatNumber(domain.errors)}</td><td>{formatBytes(domain.request_bytes + domain.response_bytes)}</td></tr>)}</tbody></table> : <SectionMessage title={domainQuery ? "没有匹配的域名" : "暂无域名数据"} detail={domainQuery ? "尝试清空搜索条件。" : "请求进入历史桶后会显示在这里。"} />}
              </div>
              <div className="domain-detail">
                {selectedDomain ? <><div className="detail-heading"><div><span>SELECTED DOMAIN</span><strong>{selectedDomain}</strong></div><button type="button" className="text-button" onClick={() => setSelectedDomain("")}>返回全部</button></div><MonoRoundedAreaChart data={detailSeries} series={[{ dataKey: "requests", name: "请求", color: colors.accent }, { dataKey: "errors", name: "错误", color: colors.errors }]} valueFormatter={formatNumber} emptyLabel="此窗口没有该域名的数据" /></> : <div className="domain-detail-empty"><Search size={22} /><strong>选择一个域名查看序列</strong><span>详情查询会携带 <code>domain=</code> 参数。</span></div>}
              </div>
            </div>
          </Panel>

          <Panel title="当前连接" eyebrow="CONNECTIONS" action={<span className="panel-caption">进程内快照</span>}>
            <div className="connection-grid"><ConnectionCard title="Gateway" value={stat.connections.gateway} /><ConnectionCard title="Front" value={stat.connections.front} /></div>
          </Panel>

          <footer className="page-footer"><span>数据窗口 {formatDateTime(stat.meta.start_time)} — {formatDateTime(stat.meta.end_time)} · UTC 存储，本地时间显示</span><span>{lastUpdated ? `最近更新 ${formatDateTime(lastUpdated)}` : "等待更新"} · 按 R 刷新</span></footer>
        </> : <SectionMessage kind="error" title="无法显示统计数据" detail="请检查 VITE_API_BASE_URL、API 服务状态和浏览器网络权限。" />}
      </main>
    </div>
  );
}

function LegacyStatCard({ label, value, color }: { label: string; value: number; color: string }) {
  return <article className="legacy-stat-card"><div className="legacy-stat-card-heading"><span>{label}</span><span className="kpi-dot" style={{ background: color }} /></div><strong>{formatNumber(value)}</strong></article>;
}

function ConnectionCard({ title, value }: { title: string; value: ConnectionSnapshot }) {
  return <div className="connection-card"><div className="connection-title"><span>{title}</span><i /></div><div className="connection-main"><strong>{formatNumber(value.active + value.idle)}</strong><span>活动 / 空闲</span></div><div className="connection-stats"><span>新建 <b>{formatNumber(value.new)}</b></span><span>升级 <b>{formatNumber(value.hijacked)}</b></span><span>关闭 <b>{formatNumber(value.closed)}</b></span></div></div>;
}
