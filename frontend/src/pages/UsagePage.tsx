import { useEffect, useState } from "react";
import { listUsage, getUsageStats, getUsageModels, listChannels, listAllKeys } from "../api/client";
import { DataTable, StatCard } from "../components/common";
import type { UsageLog, UsageStats, Channel, APIKey } from "../types";

export function UsagePage() {
  const [logs, setLogs] = useState<UsageLog[]>([]);
  const [total, setTotal] = useState(0);
  const [stats, setStats] = useState<UsageStats | null>(null);
  const [models, setModels] = useState<string[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState({
    channel_id: "",
    api_key_id: "",
    model: "",
    success: "",
    start_date: "",
    end_date: "",
  });

  const load = () => {
    const f: Record<string, unknown> = { page, page_size: 20 };
    if (filters.channel_id) f.channel_id = Number(filters.channel_id);
    if (filters.api_key_id) f.api_key_id = Number(filters.api_key_id);
    if (filters.model) f.model = filters.model;
    if (filters.success === "true") f.success = true;
    if (filters.success === "false") f.success = false;
    if (filters.start_date) f.start_date = filters.start_date;
    if (filters.end_date) f.end_date = filters.end_date;
    listUsage(f as any).then((r) => { setLogs(r.data || []); setTotal(r.total); }).catch(() => {});
    getUsageStats(filters.start_date || undefined, filters.end_date || undefined).then(setStats).catch(() => {});
  };

  useEffect(() => {
    load();
    getUsageModels().then(setModels).catch(() => {});
    listChannels().then(setChannels).catch(() => {});
    listAllKeys().then(setKeys).catch(() => {});
  }, [page, filters]);

  const channelMap = new Map(channels.map((c) => [c.id, c.name]));
  const keyMap = new Map(keys.map((k) => [k.id, k.alias || `${k.key_value.slice(0, 8)}...`]));

  const formatTokens = (n: number) => {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
    return String(n);
  };

  const columns = [
    { key: "id", label: "ID" },
    {
      key: "channel_id",
      label: "渠道",
      render: (l: UsageLog) => channelMap.get(l.channel_id) || `#${l.channel_id}`,
    },
    {
      key: "api_key_id",
      label: "Key",
      render: (l: UsageLog) => keyMap.get(l.api_key_id) || `#${l.api_key_id}`,
    },
    { key: "model", label: "模型" },
    {
      key: "input_tokens",
      label: "输入",
      render: (l: UsageLog) => formatTokens(l.input_tokens),
    },
    {
      key: "output_tokens",
      label: "输出",
      render: (l: UsageLog) => formatTokens(l.output_tokens),
    },
    {
      key: "cache_creation_tokens",
      label: "缓存写",
      render: (l: UsageLog) => formatTokens(l.cache_creation_tokens),
    },
    {
      key: "cache_read_tokens",
      label: "缓存读",
      render: (l: UsageLog) => formatTokens(l.cache_read_tokens),
    },
    { key: "latency_ms", label: "延迟", render: (l: UsageLog) => `${l.latency_ms}ms` },
    {
      key: "success",
      label: "状态",
      render: (l: UsageLog) => (
        <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${l.success ? "bg-green-500/20 text-green-400" : "bg-red-500/20 text-red-400"}`}>
          {l.success ? l.status_code : `错误 ${l.status_code}`}
        </span>
      ),
    },
    {
      key: "is_stream",
      label: "流式",
      render: (l: UsageLog) => (l.is_stream ? "是" : "否"),
    },
    { key: "created_at", label: "时间", render: (l: UsageLog) => new Date(l.created_at).toLocaleString() },
  ];

  const totalPages = Math.ceil(total / 20);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">用量</h1>
        <p className="text-sm text-[var(--loop-muted)] mt-1">这里只统计外部业务请求；手动探测和自动恢复探测不计入用量。</p>
      </div>
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
        <StatCard label="请求总数" value={stats?.total_requests ?? 0} />
        <StatCard label="输入令牌" value={formatTokens(stats?.total_input_tokens ?? 0)} />
        <StatCard label="输出令牌" value={formatTokens(stats?.total_output_tokens ?? 0)} />
        <StatCard label="缓存令牌" value={formatTokens(stats?.total_cache_tokens ?? 0)} />
        <StatCard label="成功" value={stats?.success_count ?? 0} color="text-green-400" />
        <StatCard label="失败" value={stats?.failure_count ?? 0} color="text-red-400" />
      </div>
      <div className="flex flex-wrap gap-3">
        <select value={filters.channel_id} onChange={(e) => { setFilters({ ...filters, channel_id: e.target.value }); setPage(1); }}
          className="px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm">
          <option value="">全部渠道</option>
          {channels.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
        <select value={filters.model} onChange={(e) => { setFilters({ ...filters, model: e.target.value }); setPage(1); }}
          className="px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm">
          <option value="">全部模型</option>
          {models.map((m) => <option key={m} value={m}>{m}</option>)}
        </select>
        <select value={filters.success} onChange={(e) => { setFilters({ ...filters, success: e.target.value }); setPage(1); }}
          className="px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm">
          <option value="">全部结果</option>
          <option value="true">成功</option>
          <option value="false">失败</option>
        </select>
        <input type="date" value={filters.start_date} onChange={(e) => { setFilters({ ...filters, start_date: e.target.value }); setPage(1); }}
          className="px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm" />
        <input type="date" value={filters.end_date} onChange={(e) => { setFilters({ ...filters, end_date: e.target.value }); setPage(1); }}
          className="px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm" />
      </div>
      <DataTable columns={columns} data={logs} empty="暂无用量记录" />
      {totalPages > 1 && (
        <div className="flex justify-center gap-2">
          <button onClick={() => setPage(Math.max(1, page - 1))} disabled={page <= 1}
            className="px-3 py-1.5 rounded-lg border border-[var(--loop-border)] text-sm disabled:opacity-30 hover:bg-white/5">上一页</button>
          <span className="px-3 py-1.5 text-sm text-[var(--loop-muted)]">第 {page} / {totalPages} 页</span>
          <button onClick={() => setPage(Math.min(totalPages, page + 1))} disabled={page >= totalPages}
            className="px-3 py-1.5 rounded-lg border border-[var(--loop-border)] text-sm disabled:opacity-30 hover:bg-white/5">下一页</button>
        </div>
      )}
    </div>
  );
}
