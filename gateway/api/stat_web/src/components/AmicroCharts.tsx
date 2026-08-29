import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { TooltipProps } from "recharts";
import type { NameType, ValueType } from "recharts/types/component/DefaultTooltipContent";
import { makeHourlyActivityCells, type ActivityLevel } from "../lib/amicro-utils";
import { formatLocalTime, formatNumber, formatPercent } from "../lib/format";
import type { Range, StatResponse } from "../types/stat";

const WINDOWS: Array<{ range: Range; label: string }> = [
  { range: "1h", label: "1h" },
  { range: "5h", label: "5h" },
  { range: "24h", label: "24h" },
  { range: "7d", label: "7d" },
  { range: "30d", label: "30d" },
];

const COLORS = {
  frontend: "#4aa9e8",
  backend: "#9277e6",
  errors: "#e9786d",
  blue: "#3b82f6",
};

export type OverviewStats = Partial<Record<Range, StatResponse>>;

interface AmicroBaseProps {
  reducedMotion?: boolean;
}

interface ActivityProps extends AmicroBaseProps {
  stat?: StatResponse;
}

interface StackedBarProps extends AmicroBaseProps {
  stats: OverviewStats;
  currentStat?: StatResponse | null;
}

interface DonutProps extends AmicroBaseProps {
  stat?: StatResponse | null;
}

interface SparklineProps extends AmicroBaseProps {
  stat?: StatResponse;
}

function ChartTooltip({ active, payload, label }: TooltipProps<ValueType, NameType>) {
  if (!active || !payload?.length) return null;
  return (
    <div className="amicro-tooltip">
      <div className="amicro-tooltip-label">{typeof label === "string" ? formatLocalTime(label) : label}</div>
      {payload.map((item) => (
        <div className="amicro-tooltip-row" key={`${String(item.dataKey)}-${String(item.name)}`}>
          <span className="amicro-tooltip-dot" style={{ background: item.color }} />
          <span>{item.name ?? item.dataKey}</span>
          <strong>{typeof item.value === "number" ? formatNumber(item.value) : String(item.value ?? "—")}</strong>
        </div>
      ))}
    </div>
  );
}

function activityCellDate(timestamp: string): string {
  const value = new Date(timestamp);
  return Number.isNaN(value.getTime()) ? "—" : new Intl.DateTimeFormat("zh-CN", { month: "numeric", day: "numeric", hour: "2-digit" }).format(value);
}

function activityCellOpacity(level: ActivityLevel): number {
  return [0.08, 0.28, 0.5, 0.75, 1][level];
}

export function MonoActivityBlue({ stat, reducedMotion = false }: ActivityProps) {
  const cells = makeHourlyActivityCells(stat);
  const rows = Array.from({ length: 7 }, (_, index) => cells.slice(index * 24, (index + 1) * 24));
  const total = cells.reduce((sum, cell) => sum + cell.requests, 0);

  return (
    <article className={`amicro-card amicro-activity-card${reducedMotion ? " is-reduced-motion" : ""}`} aria-label="小时请求热力图">
      <header className="amicro-card-header">
        <div>
          <div className="amicro-card-title-row"><span className="amicro-card-title">ACTIVITY HEATMAP</span><span className="amicro-card-badge blue">Sky Blue Grid</span></div>
          <strong className="amicro-card-total">{formatNumber(total)} <small>7 天请求</small></strong>
        </div>
        <span className="amicro-card-caption">小时级别</span>
      </header>

      <div className="amicro-stage amicro-activity-stage">
        {cells.length ? (
          <>
            <div className="amicro-hour-labels" aria-hidden="true"><span>00</span><span>06</span><span>12</span><span>18</span><span>24</span></div>
            <div className="amicro-activity-grid-shell">
              <div className="amicro-day-labels" aria-hidden="true">{rows.map((row, index) => <span key={row[0]?.timestamp ?? index}>{index === 6 ? "今天" : `${6 - index}天前`}</span>)}</div>
              <div className="amicro-activity-grid">
                {rows.flatMap((row) => row).map((cell) => (
                  <button
                    type="button"
                    className={`amicro-activity-cell level-${cell.level}`}
                    key={cell.timestamp}
                    title={`${activityCellDate(cell.timestamp)} · ${formatNumber(cell.requests)} 次请求`}
                    aria-label={`${activityCellDate(cell.timestamp)}，${formatNumber(cell.requests)} 次请求`}
                    style={{ opacity: activityCellOpacity(cell.level) }}
                  />
                ))}
              </div>
            </div>
            <div className="amicro-activity-hint">每格 1 小时 · 悬停查看请求量</div>
          </>
        ) : <div className="amicro-empty">暂无小时级历史数据</div>}
      </div>

      <footer className="amicro-card-footer"><span>7 天 × 24 小时</span><span>对数热度</span></footer>
    </article>
  );
}

interface StackedDatum {
  label: string;
  frontend: number;
  backend: number;
  errors: number;
}

function stackedData(stats: OverviewStats): StackedDatum[] {
  return WINDOWS.map(({ range, label }) => {
    const summary = stats[range]?.summary;
    return {
      label,
      frontend: summary?.frontend_requests ?? 0,
      backend: summary?.backend_requests ?? 0,
      errors: summary?.error_requests ?? 0,
    };
  });
}

export function MonoRoundedStackedBar({ stats, currentStat, reducedMotion = false }: StackedBarProps) {
  const data = stackedData(stats);
  const hasData = data.some((item) => item.frontend > 0 || item.backend > 0 || item.errors > 0);
  const currentTotal = currentStat?.summary.total_requests ?? 0;

  return (
    <article className={`amicro-card amicro-stacked-card${reducedMotion ? " is-reduced-motion" : ""}`} aria-label="不同时间窗口的请求组成">
      <header className="amicro-card-header">
        <div>
          <div className="amicro-card-title-row"><span className="amicro-card-title">STACKED REQUESTS</span><span className="amicro-card-badge purple">3 Layers</span></div>
          <strong className="amicro-card-total">{formatNumber(currentTotal)} <small>当前窗口</small></strong>
        </div>
        <span className="amicro-card-caption">请求组成</span>
      </header>
      <div className="amicro-stage amicro-chart-stage">
        {hasData ? (
          <ResponsiveContainer width="100%" height={176}>
            <BarChart data={data} margin={{ top: 10, right: 8, left: -20, bottom: 0 }}>
              <CartesianGrid stroke="var(--grid)" strokeDasharray="2 4" vertical={false} />
              <XAxis dataKey="label" tickLine={false} axisLine={false} tick={{ fontSize: 10, fill: "var(--muted)" }} />
              <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 9, fill: "var(--muted)" }} tickFormatter={formatNumber} />
              <Tooltip content={<ChartTooltip />} cursor={{ fill: "var(--surface-muted)" }} />
              <Bar dataKey="frontend" name="前端" stackId="requests" fill={COLORS.frontend} radius={[0, 0, 6, 6]} barSize={24} isAnimationActive={!reducedMotion} animationDuration={700} />
              <Bar dataKey="backend" name="后端" stackId="requests" fill={COLORS.backend} barSize={24} isAnimationActive={!reducedMotion} animationDuration={700} />
              <Bar dataKey="errors" name="错误" stackId="requests" fill={COLORS.errors} radius={[6, 6, 0, 0]} minPointSize={5} barSize={24} isAnimationActive={!reducedMotion} animationDuration={700} />
            </BarChart>
          </ResponsiveContainer>
        ) : <div className="amicro-empty">暂无窗口请求数据</div>}
      </div>
      <footer className="amicro-card-footer amicro-legend-footer"><span><i style={{ background: COLORS.frontend }} />前端</span><span><i style={{ background: COLORS.backend }} />后端</span><span><i style={{ background: COLORS.errors }} />错误</span></footer>
    </article>
  );
}

interface DonutDatum {
  name: string;
  value: number;
  color: string;
}

function donutData(stat?: StatResponse | null): DonutDatum[] {
  const summary = stat?.summary;
  return [
    { name: "前端请求", value: summary?.frontend_requests ?? 0, color: COLORS.frontend },
    { name: "后端请求", value: summary?.backend_requests ?? 0, color: COLORS.backend },
    { name: "错误请求", value: summary?.error_requests ?? 0, color: COLORS.errors },
  ];
}

export function MonoRoundedDonut({ stat, reducedMotion = false }: DonutProps) {
  const data = donutData(stat);
  const chartTotal = data.reduce((sum, item) => sum + item.value, 0);
  const requestTotal = stat?.summary.total_requests ?? 0;
  const hasData = chartTotal > 0;

  return (
    <article className={`amicro-card amicro-donut-card${reducedMotion ? " is-reduced-motion" : ""}`} aria-label="当前窗口前端后端错误请求占比">
      <header className="amicro-card-header">
        <div>
          <div className="amicro-card-title-row"><span className="amicro-card-title">REQUEST MIX</span><span className="amicro-card-badge coral">Soft Arc Caps</span></div>
          <strong className="amicro-card-total">{formatNumber(requestTotal)} <small>请求总量</small></strong>
        </div>
        <span className="amicro-card-caption">当前窗口</span>
      </header>
      <div className="amicro-stage amicro-donut-stage">
        {hasData ? (
          <>
            <ResponsiveContainer width="100%" height={178}>
              <PieChart>
                <Tooltip content={<ChartTooltip />} />
                <Pie data={data} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={48} outerRadius={70} paddingAngle={5} cornerRadius={8} stroke="var(--surface-muted)" strokeWidth={2} isAnimationActive={!reducedMotion} animationDuration={850}>
                  {data.map((item) => <Cell key={item.name} fill={item.color} />)}
                </Pie>
              </PieChart>
            </ResponsiveContainer>
            <div className="amicro-donut-center"><strong>{formatNumber(requestTotal)}</strong><span>请求</span></div>
          </>
        ) : <div className="amicro-empty">暂无请求数据</div>}
      </div>
      <div className="amicro-donut-legend">
        {data.map((item) => <div key={item.name}><span><i style={{ background: item.color }} />{item.name}</span><strong>{formatNumber(item.value)}</strong><small>{chartTotal ? formatPercent(item.value / chartTotal, 1) : "0%"}</small></div>)}
      </div>
      <footer className="amicro-card-footer"><span>前端 / 后端 / 错误</span><span>错误为交叉指标</span></footer>
    </article>
  );
}

interface SparkRow {
  key: "cpu" | "memory";
  label: string;
  color: string;
  current: number | null;
  data: Array<{ timestamp: string; value: number | null }>;
}

function latestValue(values: Array<number | null>): number | null {
  for (let index = values.length - 1; index >= 0; index -= 1) {
    const value = values[index];
    if (typeof value === "number" && Number.isFinite(value)) return value;
  }
  return null;
}

function sparkRows(stat?: StatResponse): SparkRow[] {
  const points = stat?.series.system ?? [];
  return [
    {
      key: "cpu",
      label: "系统 CPU",
      color: COLORS.frontend,
      current: latestValue(points.map((point) => point.cpu_percent)),
      data: points.map((point) => ({ timestamp: point.timestamp, value: point.cpu_percent })),
    },
    {
      key: "memory",
      label: "系统内存",
      color: "#43a982",
      current: latestValue(points.map((point) => point.memory_percent)),
      data: points.map((point) => ({ timestamp: point.timestamp, value: point.memory_percent })),
    },
  ];
}

export function MonoRoundedSparkline({ stat, reducedMotion = false }: SparklineProps) {
  const rows = sparkRows(stat);
  const hasData = rows.some((row) => row.data.some((point) => typeof point.value === "number" && Number.isFinite(point.value)));

  return (
    <article className={`amicro-card amicro-sparkline-card${reducedMotion ? " is-reduced-motion" : ""}`} aria-label="最近 24 小时系统 CPU 和内存趋势">
      <header className="amicro-card-header">
        <div>
          <div className="amicro-card-title-row"><span className="amicro-card-title">SYSTEM SPARKLINES</span><span className="amicro-card-badge green">Telemetry</span></div>
          <strong className="amicro-card-total">24h <small>CPU / 内存</small></strong>
        </div>
        <span className="amicro-card-caption">最近 24 小时</span>
      </header>
      <div className="amicro-stage amicro-sparkline-stage">
        {hasData ? rows.map((row) => (
          <div className="amicro-spark-row" key={row.key}>
            <div className="amicro-spark-label"><strong>{row.label}</strong><span>{row.current === null ? "—" : `${formatNumber(row.current, 1)}%`}</span></div>
            <div className="amicro-spark-chart">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={row.data} margin={{ top: 5, right: 2, bottom: 5, left: 2 }}>
                  <YAxis hide domain={[0, 100]} />
                  <XAxis hide dataKey="timestamp" />
                  <Line type="monotone" dataKey="value" stroke={row.color} strokeWidth={2.4} strokeLinecap="round" dot={false} connectNulls isAnimationActive={!reducedMotion} animationDuration={700} />
                </LineChart>
              </ResponsiveContainer>
            </div>
          </div>
        )) : <div className="amicro-empty">暂无可用的系统资源数据</div>}
      </div>
      <footer className="amicro-card-footer amicro-legend-footer"><span><i style={{ background: COLORS.frontend }} />CPU</span><span><i style={{ background: "#43a982" }} />内存</span><span>每 15 分钟</span></footer>
    </article>
  );
}
