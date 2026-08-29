import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { TooltipProps } from "recharts";
import type { NameType, ValueType } from "recharts/types/component/DefaultTooltipContent";
import type { ReactNode } from "react";
import { AnimatedNumber } from "./AnimatedNumber";
import { formatLocalTime } from "../lib/format";

export interface ChartSeries {
  dataKey: string;
  name: string;
  color: string;
  dashed?: boolean;
}

export type ChartDatum = object;

interface BaseChartProps {
  data: ChartDatum[];
  series: ChartSeries[];
  height?: number;
  valueFormatter?: (value: number) => string;
  emptyLabel?: string;
}

const defaultValueFormatter = (value: number) => value.toLocaleString("zh-CN", { maximumFractionDigits: 2 });

function EmptyChart({ height, label }: { height: number; label: string }) {
  return <div className="chart-empty" style={{ minHeight: height }}>{label}</div>;
}

function ChartTooltip({ active, payload, label }: TooltipProps<ValueType, NameType>) {
  if (!active || !payload?.length) return null;
  return (
    <div className="chart-tooltip">
      <div className="chart-tooltip-label">{typeof label === "string" ? formatLocalTime(label) : label}</div>
      {payload.map((item) => (
        <div className="chart-tooltip-row" key={`${String(item.dataKey)}-${String(item.name)}`}>
          <span className="chart-tooltip-dot" style={{ background: item.color }} />
          <span>{item.name ?? item.dataKey}</span>
          <strong>{typeof item.value === "number" ? defaultValueFormatter(item.value) : String(item.value ?? "—")}</strong>
        </div>
      ))}
    </div>
  );
}

function ChartFrame({ children, data, height, emptyLabel }: { children: ReactNode; data: ChartDatum[]; height: number; emptyLabel: string }) {
  if (!data.length) return <EmptyChart height={height} label={emptyLabel} />;
  return <div className="chart-frame" style={{ height }}>{children}</div>;
}

function Axis({ formatter }: { formatter: (value: number) => string }) {
  return (
    <>
      <CartesianGrid stroke="var(--grid)" strokeDasharray="3 6" vertical={false} />
      <XAxis dataKey="timestamp" tickFormatter={formatLocalTime} tickLine={false} axisLine={false} minTickGap={32} />
      <YAxis tickFormatter={formatter} tickLine={false} axisLine={false} width={46} />
    </>
  );
}

export function MonoRoundedLineChart({ data, series, height = 240, valueFormatter = defaultValueFormatter, emptyLabel = "暂无数据" }: BaseChartProps) {
  return (
    <ChartFrame data={data} height={height} emptyLabel={emptyLabel}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 8, right: 12, bottom: 0, left: 0 }}>
          <Axis formatter={valueFormatter} />
          <Tooltip content={<ChartTooltip />} cursor={{ stroke: "var(--muted)" }} />
          {series.length > 1 && <Legend verticalAlign="top" height={28} iconType="circle" />}
          {series.map((item) => (
            <Line key={item.dataKey} type="monotone" dataKey={item.dataKey} name={item.name} stroke={item.color} strokeWidth={2.5} dot={false} activeDot={{ r: 4 }} strokeDasharray={item.dashed ? "5 5" : undefined} connectNulls />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </ChartFrame>
  );
}

export function MonoRoundedAreaChart({ data, series, height = 240, valueFormatter = defaultValueFormatter, emptyLabel = "暂无数据" }: BaseChartProps) {
  return (
    <ChartFrame data={data} height={height} emptyLabel={emptyLabel}>
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 8, right: 12, bottom: 0, left: 0 }}>
          <defs>
            {series.map((item) => (
              <linearGradient key={item.dataKey} id={`fill-${item.dataKey}`} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={item.color} stopOpacity={0.28} />
                <stop offset="100%" stopColor={item.color} stopOpacity={0.02} />
              </linearGradient>
            ))}
          </defs>
          <Axis formatter={valueFormatter} />
          <Tooltip content={<ChartTooltip />} cursor={{ stroke: "var(--muted)" }} />
          {series.length > 1 && <Legend verticalAlign="top" height={28} iconType="circle" />}
          {series.map((item) => (
            <Area key={item.dataKey} type="monotone" dataKey={item.dataKey} name={item.name} stroke={item.color} fill={`url(#fill-${item.dataKey})`} strokeWidth={2} dot={false} connectNulls />
          ))}
        </AreaChart>
      </ResponsiveContainer>
    </ChartFrame>
  );
}

export function MonoRoundedBarChart({ data, series, height = 240, valueFormatter = defaultValueFormatter, emptyLabel = "暂无数据" }: BaseChartProps) {
  return (
    <ChartFrame data={data} height={height} emptyLabel={emptyLabel}>
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data} margin={{ top: 8, right: 12, bottom: 0, left: 0 }}>
          <Axis formatter={valueFormatter} />
          <Tooltip content={<ChartTooltip />} cursor={{ fill: "var(--surface-muted)" }} />
          {series.length > 1 && <Legend verticalAlign="top" height={28} iconType="circle" />}
          {series.map((item) => (
            <Bar key={item.dataKey} dataKey={item.dataKey} name={item.name} fill={item.color} radius={[6, 6, 0, 0]} maxBarSize={28} />
          ))}
        </BarChart>
      </ResponsiveContainer>
    </ChartFrame>
  );
}

export function MonoRoundedKpiCardChart({ label, value, animatedValue, reducedMotion = false, hint, data = [], color = "var(--accent)" }: { label: string; value: string; animatedValue?: number; reducedMotion?: boolean; hint?: string; data?: ChartDatum[]; color?: string }) {
  return (
    <article className="kpi-card">
      <div className="kpi-heading"><span>{label}</span><span className="kpi-dot" style={{ background: color }} /></div>
      <div className="kpi-value">{animatedValue === undefined ? value : <AnimatedNumber value={animatedValue} reducedMotion={reducedMotion} />}</div>
      {hint && <div className="kpi-hint">{hint}</div>}
      {data.length > 1 && (
        <div className="kpi-chart">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data}>
              <defs>
                <linearGradient id={`kpi-${label.replace(/\W/g, "")}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={color} stopOpacity={0.3} />
                  <stop offset="100%" stopColor={color} stopOpacity={0} />
                </linearGradient>
              </defs>
              <Area type="monotone" dataKey="value" stroke={color} fill={`url(#kpi-${label.replace(/\W/g, "")})`} strokeWidth={2} dot={false} />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}
    </article>
  );
}
