import { useEffect, useState } from "react";
import {
  getUsageStats,
  getUsageTimeseries,
  getUsageModelStats,
  getUsageChannelStats,
} from "../api/client";
import { StatCard } from "../components/common";
import type {
  UsageStats,
  TimeseriesPoint,
  ModelStats,
  ChannelStats,
} from "../types";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
  Cell,
  PieChart,
  Pie,
} from "recharts";
import {
  Card,
  CardBody,
  Button,
  Chip,
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
} from "@heroui/react";

const RANGES = [
  { label: "7 天", days: 7 },
  { label: "14 天", days: 14 },
  { label: "30 天", days: 30 },
  { label: "全部", days: 0 },
] as const;

const COLORS = [
  "#6366f1", "#22d3ee", "#f59e0b", "#10b981", "#f43f5e",
  "#8b5cf6", "#06b6d4", "#eab308", "#14b8a6", "#e11d48",
];

function formatTokens(n: number) {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
}

export function StatisticsPage() {
  const [rangeIdx, setRangeIdx] = useState(0);
  const [stats, setStats] = useState<UsageStats | null>(null);
  const [timeseries, setTimeseries] = useState<TimeseriesPoint[]>([]);
  const [modelStats, setModelStats] = useState<ModelStats[]>([]);
  const [channelStats, setChannelStats] = useState<ChannelStats[]>([]);

  useEffect(() => {
    const range = RANGES[rangeIdx];
    const now = new Date();
    let startDate: string | undefined;
    if (range.days > 0) {
      const d = new Date(now);
      d.setDate(d.getDate() - range.days);
      startDate = d.toISOString().slice(0, 10);
    }
    const endDate = now.toISOString().slice(0, 10);

    getUsageStats(startDate, endDate).then(setStats).catch(() => setStats(null));
    getUsageTimeseries(range.days || 365).then(setTimeseries).catch(() => setTimeseries([]));
    getUsageModelStats(startDate, endDate).then(setModelStats).catch(() => setModelStats([]));
    getUsageChannelStats(startDate, endDate).then(setChannelStats).catch(() => setChannelStats([]));
  }, [rangeIdx]);

  const hasUsageData = timeseries.some(
    (p) => p.requests > 0 || p.input_tokens > 0 || p.output_tokens > 0
  );

  const successRate =
    stats && stats.total_requests > 0
      ? ((stats.success_count / stats.total_requests) * 100).toFixed(1) + "%"
      : "-";

  const pieData =
    stats && stats.total_requests > 0
      ? [
          { name: "成功", value: stats.success_count },
          { name: "失败", value: stats.failure_count },
        ]
      : [];

  return (
    <div className="space-y-4 md:space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-xl md:text-2xl font-bold">统计分析</h1>
        <div className="flex gap-1.5">
          {RANGES.map((r, i) => (
            <Button
              key={r.days}
              size="sm"
              variant={i === rangeIdx ? "solid" : "bordered"}
              color={i === rangeIdx ? "primary" : "default"}
              onPress={() => setRangeIdx(i)}
            >
              {r.label}
            </Button>
          ))}
        </div>
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
        <StatCard label="总请求" value={stats?.total_requests ?? "-"} />
        <StatCard
          label="输入令牌"
          value={formatTokens(stats?.total_input_tokens ?? 0)}
        />
        <StatCard
          label="输出令牌"
          value={formatTokens(stats?.total_output_tokens ?? 0)}
        />
        <StatCard
          label="成功率"
          value={successRate}
          color={stats && stats.total_requests > 0 ? "text-success" : undefined}
        />
        <StatCard
          label="首字耗时"
          value={
            stats?.avg_first_token_ms && stats.avg_first_token_ms > 0
              ? `${Math.round(stats.avg_first_token_ms)}ms`
              : "-"
          }
        />
        <StatCard
          label="输出速度"
          value={
            stats?.avg_output_tokens_per_sec &&
            stats.avg_output_tokens_per_sec > 0
              ? `${stats.avg_output_tokens_per_sec.toFixed(1)} tok/s`
              : "-"
          }
        />
      </div>

      {/* Token trend chart */}
      <Card>
        <CardBody className="p-6">
          <h2 className="text-sm font-medium text-default-500 mb-4">
            令牌趋势
          </h2>
          {hasUsageData ? (
            <ResponsiveContainer width="100%" height={260}>
              <AreaChart data={timeseries}>
                <defs>
                  <linearGradient id="statInputGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#6366f1" stopOpacity={0.3} />
                    <stop offset="100%" stopColor="#6366f1" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="statOutputGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#22d3ee" stopOpacity={0.3} />
                    <stop offset="100%" stopColor="#22d3ee" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <XAxis
                  dataKey="date"
                  tick={{ fill: "#6b7280", fontSize: 11 }}
                  tickLine={false}
                  axisLine={false}
                />
                <YAxis
                  tick={{ fill: "#6b7280", fontSize: 11 }}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={formatTokens}
                />
                <Tooltip
                  contentStyle={{
                    background: "var(--heroui-background)",
                    border: "1px solid var(--heroui-default-200)",
                    borderRadius: 8,
                    fontSize: 12,
                  }}
                  labelStyle={{ color: "var(--heroui-default-500)" }}
                />
                <Area
                  type="monotone"
                  dataKey="input_tokens"
                  stroke="#6366f1"
                  fill="url(#statInputGrad)"
                  strokeWidth={2}
                  name="输入"
                />
                <Area
                  type="monotone"
                  dataKey="output_tokens"
                  stroke="#22d3ee"
                  fill="url(#statOutputGrad)"
                  strokeWidth={2}
                  name="输出"
                />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <EmptyState text="暂无令牌数据" />
          )}
        </CardBody>
      </Card>

      {/* Model distribution + Success pie */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card>
          <CardBody className="p-6">
            <h2 className="text-sm font-medium text-default-500 mb-4">
              模型分布
            </h2>
            {modelStats.length > 0 ? (
              <ResponsiveContainer width="100%" height={Math.max(200, modelStats.length * 36)}>
                <BarChart
                  data={modelStats.slice(0, 10)}
                  layout="vertical"
                  margin={{ left: 80 }}
                >
                  <XAxis
                    type="number"
                    tick={{ fill: "#6b7280", fontSize: 11 }}
                    tickLine={false}
                    axisLine={false}
                  />
                  <YAxis
                    type="category"
                    dataKey="model"
                    tick={{ fill: "#6b7280", fontSize: 11 }}
                    tickLine={false}
                    axisLine={false}
                    width={80}
                  />
                  <Tooltip
                    contentStyle={{
                      background: "var(--heroui-background)",
                      border: "1px solid var(--heroui-default-200)",
                      borderRadius: 8,
                      fontSize: 12,
                    }}
                  />
                  <Bar dataKey="requests" name="请求数" radius={[0, 4, 4, 0]}>
                    {modelStats.slice(0, 10).map((_, i) => (
                      <Cell key={i} fill={COLORS[i % COLORS.length]} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <EmptyState text="暂无模型数据" />
            )}
          </CardBody>
        </Card>

        <Card>
          <CardBody className="p-6">
            <h2 className="text-sm font-medium text-default-500 mb-4">
              成功率
            </h2>
            {pieData.length > 0 ? (
              <div className="flex items-center justify-center gap-6">
                <ResponsiveContainer width={180} height={180}>
                  <PieChart>
                    <Pie
                      data={pieData}
                      cx="50%"
                      cy="50%"
                      innerRadius={50}
                      outerRadius={80}
                      dataKey="value"
                      strokeWidth={0}
                    >
                      <Cell fill="#10b981" />
                      <Cell fill="#f43f5e" />
                    </Pie>
                    <Tooltip
                      contentStyle={{
                        background: "var(--heroui-background)",
                        border: "1px solid var(--heroui-default-200)",
                        borderRadius: 8,
                        fontSize: 12,
                      }}
                    />
                  </PieChart>
                </ResponsiveContainer>
                <div className="space-y-3 text-sm">
                  <div className="flex items-center gap-2">
                    <span className="w-3 h-3 rounded-full bg-success inline-block" />
                    <span className="text-default-500">成功</span>
                    <span className="font-medium">{stats?.success_count ?? 0}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-3 h-3 rounded-full bg-danger inline-block" />
                    <span className="text-default-500">失败</span>
                    <span className="font-medium">{stats?.failure_count ?? 0}</span>
                  </div>
                </div>
              </div>
            ) : (
              <EmptyState text="暂无请求数据" />
            )}
          </CardBody>
        </Card>
      </div>

      {/* Channel performance table */}
      <Card>
        <CardBody className="p-6">
          <h2 className="text-sm font-medium text-default-500 mb-4">
            渠道性能
          </h2>
          {channelStats.length > 0 ? (
            <Table aria-label="Channel performance table">
              <TableHeader>
                <TableColumn>渠道</TableColumn>
                <TableColumn>请求数</TableColumn>
                <TableColumn>成功率</TableColumn>
                <TableColumn>输入令牌</TableColumn>
                <TableColumn>输出令牌</TableColumn>
                <TableColumn>平均延迟</TableColumn>
              </TableHeader>
              <TableBody emptyContent="暂无数据">
                {channelStats.map((cs) => (
                  <TableRow key={cs.channel_id}>
                    <TableCell>{cs.channel_name}</TableCell>
                    <TableCell>{cs.requests}</TableCell>
                    <TableCell>
                      <Chip
                        size="sm"
                        variant="flat"
                        color={cs.success_rate >= 90 ? "success" : cs.success_rate >= 50 ? "warning" : "danger"}
                      >
                        {cs.success_rate.toFixed(1)}%
                      </Chip>
                    </TableCell>
                    <TableCell>{formatTokens(cs.input_tokens)}</TableCell>
                    <TableCell>{formatTokens(cs.output_tokens)}</TableCell>
                    <TableCell>
                      {cs.avg_latency_ms > 0
                        ? `${Math.round(cs.avg_latency_ms)}ms`
                        : "-"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <EmptyState text="暂无渠道数据" />
          )}
        </CardBody>
      </Card>
    </div>
  );
}

function EmptyState({ text }: { text: string }) {
  return (
    <div className="h-48 flex items-center justify-center">
      <span className="text-sm text-default-500">{text}</span>
    </div>
  );
}
