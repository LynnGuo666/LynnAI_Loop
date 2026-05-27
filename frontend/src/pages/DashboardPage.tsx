import { useEffect, useState } from "react";
import { listChannels, listAllKeys, getUsageStats, getUsageTimeseries } from "../api/client";
import { StatCard } from "../components/common";
import type { Channel, APIKey, UsageStats, TimeseriesPoint } from "../types";
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts";

export function DashboardPage() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [stats, setStats] = useState<UsageStats | null>(null);
  const [timeseries, setTimeseries] = useState<TimeseriesPoint[]>([]);

  useEffect(() => {
    listChannels().then(setChannels).catch(() => {});
    listAllKeys().then(setKeys).catch(() => {});
    getUsageStats().then(setStats).catch(() => {});
    getUsageTimeseries(7).then(setTimeseries).catch(() => {});
  }, []);

  const activeKeys = keys.filter((k) => k.is_active).length;
  const disabledKeys = keys.filter((k) => !k.is_active).length;

  const formatTokens = (n: number) => {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
    return String(n);
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
        <StatCard label="Channels" value={channels.length} />
        <StatCard label="Total Keys" value={keys.length} />
        <StatCard label="Active Keys" value={activeKeys} color="text-green-400" />
        <StatCard label="Disabled Keys" value={disabledKeys} color="text-red-400" />
        <StatCard label="Today Requests" value={stats?.total_requests ?? "-"} />
        <StatCard label="Total Tokens" value={formatTokens((stats?.total_input_tokens ?? 0) + (stats?.total_output_tokens ?? 0))} />
      </div>
      <div className="rounded-xl border border-[var(--loop-border)] bg-[var(--loop-card)] p-6">
        <h2 className="text-sm font-medium text-[var(--loop-muted)] mb-4">7-Day Usage</h2>
        {timeseries.length > 0 ? (
          <ResponsiveContainer width="100%" height={280}>
            <AreaChart data={timeseries}>
              <defs>
                <linearGradient id="inputGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#6366f1" stopOpacity={0.3} />
                  <stop offset="100%" stopColor="#6366f1" stopOpacity={0} />
                </linearGradient>
                <linearGradient id="outputGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#22d3ee" stopOpacity={0.3} />
                  <stop offset="100%" stopColor="#22d3ee" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis dataKey="date" tick={{ fill: "#6b7280", fontSize: 11 }} tickLine={false} axisLine={false} />
              <YAxis tick={{ fill: "#6b7280", fontSize: 11 }} tickLine={false} axisLine={false} tickFormatter={formatTokens} />
              <Tooltip contentStyle={{ background: "#1e1e2e", border: "1px solid #333", borderRadius: 8, fontSize: 12 }} labelStyle={{ color: "#9ca3af" }} />
              <Area type="monotone" dataKey="input_tokens" stroke="#6366f1" fill="url(#inputGrad)" strokeWidth={2} name="Input" />
              <Area type="monotone" dataKey="output_tokens" stroke="#22d3ee" fill="url(#outputGrad)" strokeWidth={2} name="Output" />
            </AreaChart>
          </ResponsiveContainer>
        ) : (
          <div className="h-64 flex items-center justify-center text-[var(--loop-muted)] text-sm">No data yet</div>
        )}
      </div>
    </div>
  );
}
