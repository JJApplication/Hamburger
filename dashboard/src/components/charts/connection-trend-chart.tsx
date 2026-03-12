"use client";

import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

import { usePreferences } from "@/lib/preferences/preferences-context";
import type { ConnectionTrendPoint } from "@/types/gateway";

interface ConnectionTrendChartProps {
  data: ConnectionTrendPoint[];
}

export function ConnectionTrendChart({ data }: ConnectionTrendChartProps) {
  const { t, theme } = usePreferences();

  return (
    <div className="h-[260px] w-full">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 8, right: 12, left: -20, bottom: 0 }}>
          <XAxis dataKey="timestamp" stroke="var(--text-secondary)" tickLine={false} axisLine={false} />
          <YAxis stroke="var(--text-secondary)" tickLine={false} axisLine={false} width={44} />
          <Tooltip
            contentStyle={{
              backgroundColor: theme === "dark" ? "#111827" : "#ffffff",
              border: "1px solid var(--border)",
              borderRadius: 12,
              color: "var(--text-primary)",
            }}
          />
          <Line
            type="monotone"
            dataKey="totalConnections"
            stroke="#34d399"
            strokeWidth={2.4}
            dot={false}
            name={t("overview.totalConnections")}
          />
          <Line type="monotone" dataKey="activeDomains" stroke="#38bdf8" strokeWidth={2} dot={false} name={t("overview.activeDomains")} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
