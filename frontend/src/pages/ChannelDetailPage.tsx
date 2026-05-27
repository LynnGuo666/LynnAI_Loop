import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { getChannel, listKeysByChannel, createKey, deleteKey, enableKey, probeKey, getUsageTimeseries } from "../api/client";
import { StatCard, DataTable, ConfirmDialog } from "../components/common";
import type { Channel, APIKey, TimeseriesPoint, KeyProbe } from "../types";
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts";

export function ChannelDetailPage() {
  const { id } = useParams<{ id: string }>();
  const channelId = Number(id);
  const [channel, setChannel] = useState<Channel | null>(null);
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [timeseries, setTimeseries] = useState<TimeseriesPoint[]>([]);
  const [showAddKey, setShowAddKey] = useState(false);
  const [delKeyId, setDelKeyId] = useState<number | null>(null);
  const [probeResult, setProbeResult] = useState<KeyProbe | null>(null);

  const load = () => {
    getChannel(channelId).then(setChannel).catch(() => {});
    listKeysByChannel(channelId).then(setKeys).catch(() => {});
    getUsageTimeseries(7).then(setTimeseries).catch(() => {});
  };
  useEffect(() => { load(); }, [channelId]);

  const activeKeys = keys.filter((k) => k.is_active).length;

  const maskKey = (k: string) => {
    if (k.length <= 10) return "****";
    return k.slice(0, 7) + "..." + k.slice(-4);
  };

  const handleAddKey = async (keyValue: string, alias: string) => {
    await createKey(channelId, { key_value: keyValue, alias });
    setShowAddKey(false);
    load();
  };

  const handleDeleteKey = async () => {
    if (delKeyId) await deleteKey(delKeyId);
    setDelKeyId(null);
    load();
  };

  const handleEnable = async (keyId: number) => {
    await enableKey(keyId);
    load();
  };

  const handleProbe = async (keyId: number) => {
    const result = await probeKey(keyId);
    setProbeResult(result);
    load();
  };

  const columns = [
    { key: "id", label: "ID" },
    { key: "alias", label: "别名" },
    {
      key: "key_value",
      label: "密钥",
      render: (k: APIKey) => <span className="font-mono text-xs text-[var(--loop-muted)]">{maskKey(k.key_value)}</span>,
    },
    {
      key: "is_active",
      label: "状态",
      render: (k: APIKey) => (
        <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${k.is_active ? "bg-green-500/20 text-green-400" : "bg-red-500/20 text-red-400"}`}>
          {k.is_active ? "启用" : "停用"}
        </span>
      ),
    },
    { key: "consecutive_failures", label: "连续失败" },
    { key: "total_successes", label: "成功次数" },
    {
      key: "actions",
      label: "",
      render: (k: APIKey) => (
        <div className="flex gap-2 text-xs">
          {!k.is_active && (
            <button onClick={() => handleEnable(k.id)} className="text-green-400 hover:text-green-300">启用</button>
          )}
          <button onClick={() => handleProbe(k.id)} className="text-blue-400 hover:text-blue-300">探测</button>
          <button onClick={() => setDelKeyId(k.id)} className="text-red-400 hover:text-red-300">删除</button>
        </div>
      ),
    },
  ];

  const formatTokens = (n: number) => {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
    return String(n);
  };

  if (!channel) return <div className="text-[var(--loop-muted)]">加载中...</div>;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{channel.name}</h1>
        <p className="text-sm text-[var(--loop-muted)] mt-1">{channel.base_url}</p>
        {channel.description && <p className="text-sm text-[var(--loop-muted)]">{channel.description}</p>}
      </div>
      <div className="grid grid-cols-3 gap-4">
        <StatCard label="密钥总数" value={keys.length} />
        <StatCard label="可用密钥" value={activeKeys} color="text-green-400" />
        <StatCard label="停用密钥" value={keys.length - activeKeys} color="text-red-400" />
      </div>
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">API 密钥</h2>
        <button onClick={() => setShowAddKey(true)} className="px-4 py-2 rounded-xl bg-[var(--loop-primary)] text-white text-sm font-medium hover:opacity-90 transition">
          + 添加密钥
        </button>
      </div>
      <DataTable columns={columns} data={keys} empty="暂未添加密钥" />
      <div className="rounded-xl border border-[var(--loop-border)] bg-[var(--loop-card)] p-6">
        <h2 className="text-sm font-medium text-[var(--loop-muted)] mb-4">近 7 天用量</h2>
        {timeseries.length > 0 ? (
          <ResponsiveContainer width="100%" height={240}>
            <AreaChart data={timeseries}>
              <XAxis dataKey="date" tick={{ fill: "#6b7280", fontSize: 11 }} tickLine={false} axisLine={false} />
              <YAxis tick={{ fill: "#6b7280", fontSize: 11 }} tickLine={false} axisLine={false} tickFormatter={formatTokens} />
              <Tooltip contentStyle={{ background: "#1e1e2e", border: "1px solid #333", borderRadius: 8, fontSize: 12 }} />
              <Area type="monotone" dataKey="input_tokens" stroke="#6366f1" fill="#6366f120" strokeWidth={2} name="输入" />
              <Area type="monotone" dataKey="output_tokens" stroke="#22d3ee" fill="#22d3ee20" strokeWidth={2} name="输出" />
            </AreaChart>
          </ResponsiveContainer>
        ) : (
          <div className="h-48 flex items-center justify-center text-[var(--loop-muted)] text-sm">暂无数据</div>
        )}
      </div>
      {showAddKey && <AddKeyModal onClose={() => setShowAddKey(false)} onAdd={handleAddKey} />}
      <ConfirmDialog open={delKeyId !== null} title="删除密钥" message="这会永久移除该 API 密钥。" onConfirm={handleDeleteKey} onCancel={() => setDelKeyId(null)} danger />
      {probeResult && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setProbeResult(null)}>
          <div className="bg-[var(--loop-card)] border border-[var(--loop-border)] rounded-2xl p-6 w-full max-w-sm" onClick={(e) => e.stopPropagation()}>
            <h3 className="text-lg font-semibold mb-3">探测结果</h3>
            <div className="space-y-2 text-sm">
              <div>是否成功：<span className={probeResult.success ? "text-green-400" : "text-red-400"}>{probeResult.success ? "是" : "否"}</span></div>
              <div>状态码：{probeResult.status_code}</div>
              <div>延迟：{probeResult.latency_ms}ms</div>
              {probeResult.error_msg && <div className="text-red-400">错误：{probeResult.error_msg}</div>}
            </div>
            <button onClick={() => setProbeResult(null)} className="mt-4 px-4 py-2 text-sm rounded-lg border border-[var(--loop-border)] hover:bg-white/5">关闭</button>
          </div>
        </div>
      )}
    </div>
  );
}

function AddKeyModal({ onClose, onAdd }: { onClose: () => void; onAdd: (key: string, alias: string) => void }) {
  const [key, setKey] = useState("");
  const [alias, setAlias] = useState("");
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={onClose}>
      <div className="bg-[var(--loop-card)] border border-[var(--loop-border)] rounded-2xl p-6 w-full max-w-md space-y-4" onClick={(e) => e.stopPropagation()}>
        <h2 className="text-lg font-semibold">添加 API 密钥</h2>
        <input value={alias} onChange={(e) => setAlias(e.target.value)} placeholder="别名（可选）" className="w-full px-4 py-2.5 rounded-xl bg-[var(--loop-bg)] border border-[var(--loop-border)] text-[var(--loop-text)] placeholder:text-[var(--loop-muted)] focus:outline-none focus:border-[var(--loop-primary)]" />
        <input value={key} onChange={(e) => setKey(e.target.value)} placeholder="sk-ant-..." className="w-full px-4 py-2.5 rounded-xl bg-[var(--loop-bg)] border border-[var(--loop-border)] text-[var(--loop-text)] placeholder:text-[var(--loop-muted)] focus:outline-none focus:border-[var(--loop-primary)] font-mono text-sm" />
        <div className="flex justify-end gap-3">
          <button onClick={onClose} className="px-4 py-2 text-sm rounded-lg border border-[var(--loop-border)] hover:bg-white/5">取消</button>
          <button onClick={() => onAdd(key, alias)} disabled={!key} className="px-4 py-2 text-sm rounded-lg bg-[var(--loop-primary)] text-white hover:opacity-90 disabled:opacity-40">添加</button>
        </div>
      </div>
    </div>
  );
}
