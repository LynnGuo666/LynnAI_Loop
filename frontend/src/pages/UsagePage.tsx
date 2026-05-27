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

  const formatTokens = (n: number) => {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
    return String(n);
  };

  const columns = [
    { key: "id", label: "ID" },
    {
      key: "channel_id",
      label: "Channel",
      render: (l: UsageLog) => channelMap.get(l.channel_id) || `#${l.channel_id}`,
    },
    { key: "model", label: "Model" },
    {
      key: "input_tokens",
      label: "Input",
      render: (l: UsageLog) => formatTokens(l.input_tokens),
    },
    {
      key: "output_tokens",
      label: "Output",
      render: (l: UsageLog) => formatTokens(l.output_tokens),
    },
    { key: "latency_ms", label: "Latency", render: (l: UsageLog) => `${l.latency_ms}ms` },
    {
      key: "success",
      label: "Status",
      render: (l: UsageLog) => (
        <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${l.success ? "bg-green-500/20 text-green-400" : "bg-red-500/20 text-red-400"}`}>
          {l.success ? l.status_code : `ERR ${l.status_code}`}
        </span>
      ),
    },
    {
      key: "is_stream",
      label: "Stream",
      render: (l: UsageLog) => (l.is_stream ? "Yes" : "No"),
    },
    { key: "created_at", label: "Time", render: (l: UsageLog) => new Date(l.created_at).toLocaleString() },
  ];

  const totalPages = Math.ceil(total / 20);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Usage</h1>
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
        <StatCard label="Total Requests" value={stats?.total_requests ?? 0} />
        <StatCard label="Input Tokens" value={formatTokens(stats?.total_input_tokens ?? 0)} />
        <StatCard label="Output Tokens" value={formatTokens(stats?.total_output_tokens ?? 0)} />
        <StatCard label="Cache Tokens" value={formatTokens(stats?.total_cache_tokens ?? 0)} />
        <StatCard label="Success" value={stats?.success_count ?? 0} color="text-green-400" />
        <StatCard label="Failure" value={stats?.failure_count ?? 0} color="text-red-400" />
      </div>
      <div className="flex flex-wrap gap-3">
        <select value={filters.channel_id} onChange={(e) => { setFilters({ ...filters, channel_id: e.target.value }); setPage(1); }}
          className="px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm">
          <option value="">All Channels</option>
          {channels.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
        <select value={filters.model} onChange={(e) => { setFilters({ ...filters, model: e.target.value }); setPage(1); }}
          className="px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm">
          <option value="">All Models</option>
          {models.map((m) => <option key={m} value={m}>{m}</option>)}
        </select>
        <select value={filters.success} onChange={(e) => { setFilters({ ...filters, success: e.target.value }); setPage(1); }}
          className="px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm">
          <option value="">All Results</option>
          <option value="true">Success</option>
          <option value="false">Failure</option>
        </select>
        <input type="date" value={filters.start_date} onChange={(e) => { setFilters({ ...filters, start_date: e.target.value }); setPage(1); }}
          className="px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm" />
        <input type="date" value={filters.end_date} onChange={(e) => { setFilters({ ...filters, end_date: e.target.value }); setPage(1); }}
          className="px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm" />
      </div>
      <DataTable columns={columns} data={logs} empty="No usage records" />
      {totalPages > 1 && (
        <div className="flex justify-center gap-2">
          <button onClick={() => setPage(Math.max(1, page - 1))} disabled={page <= 1}
            className="px-3 py-1.5 rounded-lg border border-[var(--loop-border)] text-sm disabled:opacity-30 hover:bg-white/5">Prev</button>
          <span className="px-3 py-1.5 text-sm text-[var(--loop-muted)]">Page {page} / {totalPages}</span>
          <button onClick={() => setPage(Math.min(totalPages, page + 1))} disabled={page >= totalPages}
            className="px-3 py-1.5 rounded-lg border border-[var(--loop-border)] text-sm disabled:opacity-30 hover:bg-white/5">Next</button>
        </div>
      )}
    </div>
  );
}
