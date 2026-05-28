import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { listChannels, listAllKeys, getUsageStats, getUsageTimeseries } from "../api/client";
import { StatCard } from "../components/common";
import type { Channel, APIKey, UsageStats, TimeseriesPoint } from "../types";
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts";
import { Card, CardBody, Button } from "@heroui/react";

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
  const hasUsageData = timeseries.some(
    (point) => point.requests > 0 || point.input_tokens > 0 || point.output_tokens > 0
  );

  const formatTokens = (n: number) => {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
    return String(n);
  };

  return (
    <div className="space-y-4 md:space-y-6">
      <h1 className="text-xl md:text-2xl font-bold">仪表盘</h1>
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
        <StatCard label="渠道数" value={channels.length} />
        <StatCard label="密钥总数" value={keys.length} />
        <StatCard label="可用密钥" value={activeKeys} color="text-success" />
        <StatCard label="停用密钥" value={disabledKeys} color="text-danger" />
        <StatCard label="今日请求" value={stats?.total_requests ?? "-"} />
        <StatCard label="令牌总量" value={formatTokens((stats?.total_input_tokens ?? 0) + (stats?.total_output_tokens ?? 0))} />
      </div>
      <Card>
        <CardBody className="p-6">
          <h2 className="text-sm font-medium text-default-500 mb-4">近 7 天用量</h2>
          {hasUsageData ? (
            <ResponsiveContainer width="100%" height={220}>
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
                <Tooltip
                  contentStyle={{
                    background: "var(--heroui-background)",
                    border: "1px solid var(--heroui-default-200)",
                    borderRadius: 8,
                    fontSize: 12,
                  }}
                  labelStyle={{ color: "var(--heroui-default-500)" }}
                />
                <Area type="monotone" dataKey="input_tokens" stroke="#6366f1" fill="url(#inputGrad)" strokeWidth={2} name="输入" />
                <Area type="monotone" dataKey="output_tokens" stroke="#22d3ee" fill="url(#outputGrad)" strokeWidth={2} name="输出" />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-64 flex flex-col items-center justify-center gap-4 text-center">
              <div>
                <div className="text-sm font-medium text-foreground">还没有业务请求记录</div>
                <div className="mt-1 text-xs text-default-500">产生用量后，这里会展示近 7 天的令牌趋势。</div>
              </div>
              <div className="flex flex-wrap justify-center gap-2">
                <Button as={Link} to="/usage" variant="bordered" size="sm">
                  查看用量明细
                </Button>
                <Button as={Link} to="/channels" color="primary" size="sm">
                  查看渠道
                </Button>
              </div>
            </div>
          )}
        </CardBody>
      </Card>
    </div>
  );
}
