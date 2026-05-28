import { Fragment, useEffect, useState } from "react";
import { listUsage, getUsageModels, listChannels, listAllKeys } from "../api/client";
import type { UsageLog, Channel, APIKey } from "../types";

export function UsagePage() {
  const [logs, setLogs] = useState<UsageLog[]>([]);
  const [total, setTotal] = useState(0);
  const [models, setModels] = useState<string[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [page, setPage] = useState(1);
  const [expandedId, setExpandedId] = useState<number | null>(null);
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
  const formatMs = (n?: number) => (n && n > 0 ? `${Math.round(n)}ms` : "-");
  const formatSpeed = (n?: number) => (n && n > 0 ? `${n.toFixed(1)} tok/s` : "-");

  const totalPages = Math.ceil(total / 20);

  return (
    <div className="space-y-4 md:space-y-6">
      <div>
        <h1 className="text-xl md:text-2xl font-bold">用量</h1>
        <p className="text-sm text-[var(--loop-muted)] mt-1">这里只统计外部业务请求；手动探测和自动恢复探测不计入用量。</p>
      </div>
      <div className="flex flex-wrap gap-2 md:gap-3">
        <select value={filters.channel_id} onChange={(e) => { setFilters({ ...filters, channel_id: e.target.value }); setPage(1); }}
          className="flex-1 min-w-[120px] px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm">
          <option value="">全部渠道</option>
          {channels.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
        <select value={filters.model} onChange={(e) => { setFilters({ ...filters, model: e.target.value }); setPage(1); }}
          className="flex-1 min-w-[120px] px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm">
          <option value="">全部模型</option>
          {models.map((m) => <option key={m} value={m}>{m}</option>)}
        </select>
        <select value={filters.success} onChange={(e) => { setFilters({ ...filters, success: e.target.value }); setPage(1); }}
          className="flex-1 min-w-[100px] px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm">
          <option value="">全部结果</option>
          <option value="true">成功</option>
          <option value="false">失败</option>
        </select>
        <input type="date" value={filters.start_date} onChange={(e) => { setFilters({ ...filters, start_date: e.target.value }); setPage(1); }}
          className="flex-1 min-w-[130px] px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm" />
        <input type="date" value={filters.end_date} onChange={(e) => { setFilters({ ...filters, end_date: e.target.value }); setPage(1); }}
          className="flex-1 min-w-[130px] px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm" />
      </div>
      <div className="overflow-x-auto rounded-xl border border-[var(--loop-border)]">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-[var(--loop-border)] bg-[var(--loop-card)]">
              <th className="px-3 md:px-4 py-3 text-left text-xs font-medium text-[var(--loop-muted)] uppercase tracking-wider">ID</th>
              <th className="px-3 md:px-4 py-3 text-left text-xs font-medium text-[var(--loop-muted)] uppercase tracking-wider">渠道</th>
              <th className="px-3 md:px-4 py-3 text-left text-xs font-medium text-[var(--loop-muted)] uppercase tracking-wider">Key</th>
              <th className="px-3 md:px-4 py-3 text-left text-xs font-medium text-[var(--loop-muted)] uppercase tracking-wider">模型</th>
              <th className="px-3 md:px-4 py-3 text-left text-xs font-medium text-[var(--loop-muted)] uppercase tracking-wider">延迟</th>
              <th className="px-3 md:px-4 py-3 text-left text-xs font-medium text-[var(--loop-muted)] uppercase tracking-wider">性能</th>
              <th className="px-3 md:px-4 py-3 text-left text-xs font-medium text-[var(--loop-muted)] uppercase tracking-wider">时间</th>
              <th className="px-3 md:px-4 py-3 text-left text-xs font-medium text-[var(--loop-muted)] uppercase tracking-wider"></th>
            </tr>
          </thead>
          <tbody>
            {logs.length === 0 ? (
              <tr>
                <td colSpan={8} className="px-4 py-8 text-center text-[var(--loop-muted)]">暂无用量记录</td>
              </tr>
            ) : (
              logs.map((log) => {
                const expanded = expandedId === log.id;
                return (
                  <Fragment key={log.id}>
                    <tr key={log.id} className="border-b border-[var(--loop-border)] hover:bg-white/[0.02]">
                      <td className="px-3 md:px-4 py-3">{log.id}</td>
                      <td className="px-3 md:px-4 py-3">{channelMap.get(log.channel_id) || `#${log.channel_id}`}</td>
                      <td className="px-3 md:px-4 py-3">{keyMap.get(log.api_key_id) || `#${log.api_key_id}`}</td>
                      <td className="px-3 md:px-4 py-3">{log.model}</td>
                      <td className="px-3 md:px-4 py-3">{log.latency_ms}ms</td>
                      <td className="px-3 md:px-4 py-3">
                        <div>{formatMs(log.first_token_ms)}</div>
                        <div className="text-xs text-[var(--loop-muted)]">{formatSpeed(log.output_tokens_per_sec)}</div>
                      </td>
                      <td className="px-3 md:px-4 py-3 whitespace-nowrap">{new Date(log.created_at).toLocaleString()}</td>
                      <td className="px-3 md:px-4 py-3 text-right">
                        <button
                          onClick={() => setExpandedId(expanded ? null : log.id)}
                          className="rounded-lg border border-[var(--loop-border)] px-2.5 py-1 text-xs hover:bg-white/5"
                        >
                          {expanded ? "收起" : "详情"}
                        </button>
                      </td>
                    </tr>
                    {expanded && (
                      <tr key={`${log.id}-details`} className="border-b border-[var(--loop-border)] bg-white/[0.02]">
                        <td colSpan={8} className="px-3 md:px-4 py-4">
                          <div className="grid gap-3 text-xs sm:grid-cols-2 lg:grid-cols-4">
                            <DetailItem label="输入令牌" value={formatTokens(log.input_tokens)} />
                            <DetailItem label="输出令牌" value={formatTokens(log.output_tokens)} />
                            <DetailItem label="缓存写" value={formatTokens(log.cache_creation_tokens)} />
                            <DetailItem label="缓存读" value={formatTokens(log.cache_read_tokens)} />
                            <DetailItem label="状态" value={log.success ? `${log.status_code}` : `错误 ${log.status_code}`} tone={log.success ? "success" : "danger"} />
                            <DetailItem label="流式" value={log.is_stream ? "是" : "否"} />
                            <DetailItem label="端点" value={log.endpoint || "-"} />
                            <DetailItem label="错误信息" value={log.error_message || "-"} tone={log.error_message ? "danger" : undefined} />
                          </div>
                        </td>
                      </tr>
                    )}
                  </Fragment>
                );
              })
            )}
          </tbody>
        </table>
      </div>
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

function DetailItem({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: "success" | "danger";
}) {
  const toneClass = tone === "success" ? "text-green-400" : tone === "danger" ? "text-red-400" : "text-[var(--loop-text)]";

  return (
    <div className="min-w-0 rounded-lg border border-[var(--loop-border)] bg-[var(--loop-bg)] px-3 py-2">
      <div className="text-[var(--loop-muted)]">{label}</div>
      <div className={`mt-1 break-all font-medium ${toneClass}`}>{value}</div>
    </div>
  );
}
