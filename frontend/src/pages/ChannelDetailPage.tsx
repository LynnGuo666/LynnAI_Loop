import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { getChannel, listKeysByChannel, createKey, deleteKey, enableKey, probeKey, getUsageTimeseries, updateChannel, updateKey, listChannelModels, exportKeys, importKeys, listUsage } from "../api/client";
import { StatCard, DataTable, ConfirmDialog, KeyFormModal, KeyImportModal } from "../components/common";
import type { Channel, APIKey, TimeseriesPoint, KeyProbe, KeyImportItem, UsageLog } from "../types";
import { AreaChart, Area, CartesianGrid, Line, LineChart, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts";

interface ChannelPerformancePoint {
  time: string;
  first_token_ms: number;
  output_tokens_per_sec: number;
}

export function ChannelDetailPage() {
  const { id } = useParams<{ id: string }>();
  const channelId = Number(id);
  const [channel, setChannel] = useState<Channel | null>(null);
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [timeseries, setTimeseries] = useState<TimeseriesPoint[]>([]);
  const [showAddKey, setShowAddKey] = useState(false);
  const [showImportKeys, setShowImportKeys] = useState(false);
  const [editingKey, setEditingKey] = useState<APIKey | null>(null);
  const [delKeyId, setDelKeyId] = useState<number | null>(null);
  const [probeResult, setProbeResult] = useState<KeyProbe | null>(null);
  const [probeModel, setProbeModel] = useState("");
  const [modelOptions, setModelOptions] = useState<string[]>([]);
  const [modelLoading, setModelLoading] = useState(false);
  const [modelMsg, setModelMsg] = useState("");
  const [performanceLogs, setPerformanceLogs] = useState<UsageLog[]>([]);

  const load = () => {
    getChannel(channelId).then(setChannel).catch(() => {});
    listKeysByChannel(channelId).then(setKeys).catch(() => {});
    getUsageTimeseries(7).then(setTimeseries).catch(() => {});
    listUsage({ channel_id: channelId, page: 1, page_size: 100 }).then((r) => setPerformanceLogs(r.data || [])).catch(() => {});
  };
  useEffect(() => { load(); }, [channelId]);
  useEffect(() => {
    if (channel) setProbeModel(channel.probe_model || "");
  }, [channel?.id, channel?.probe_model]);

  const activeKeys = keys.filter((k) => k.is_active).length;

  const maskKey = (k: string) => {
    if (k.length <= 10) return "****";
    return k.slice(0, 7) + "..." + k.slice(-4);
  };

  const handleAddKey = async (data: { keyValue: string; alias: string }) => {
    await createKey(channelId, { key_value: data.keyValue, alias: data.alias });
    load();
  };

  const handleImportKeys = async (data: { keys: KeyImportItem[] }) => {
    const result = await importKeys({ channel_id: channelId, keys: data.keys });
    load();
    return result;
  };

  const handleExportKeys = async () => {
    const exported = await exportKeys(channelId);
    downloadJSON(exported, `loop-keys-channel-${channelId}.json`);
  };

  const handleUpdateKey = async (data: { keyValue: string; alias: string }) => {
    if (!editingKey) return;
    await updateKey(editingKey.id, { key_value: data.keyValue, alias: data.alias });
    setEditingKey(null);
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

  const handleLoadModels = async () => {
    setModelLoading(true);
    setModelMsg("");
    try {
      const models = await listChannelModels(channelId);
      setModelOptions(models);
      if (!probeModel && models.length > 0) setProbeModel(models[0]);
      setModelMsg(models.length > 0 ? `已获取 ${models.length} 个模型` : "未获取到模型，可手动填写");
    } catch (err) {
      setModelMsg(err instanceof Error ? `获取模型失败：${err.message}` : "获取模型失败");
    } finally {
      setModelLoading(false);
      setTimeout(() => setModelMsg(""), 3000);
    }
  };

  const handleSaveProbeModel = async () => {
    if (!channel) return;
    setModelMsg("");
    try {
      const updated = await updateChannel(channel.id, { probe_model: probeModel.trim() });
      setChannel(updated);
      setModelMsg("探测模型已保存");
    } catch {
      setModelMsg("保存探测模型失败");
    } finally {
      setTimeout(() => setModelMsg(""), 3000);
    }
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
          <button onClick={() => setEditingKey(k)} className="text-[var(--loop-primary)] hover:opacity-80">编辑</button>
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
  const performanceData = buildPerformanceData(performanceLogs);

  if (!channel) return <div className="text-[var(--loop-muted)]">加载中...</div>;

  return (
    <div className="space-y-4 md:space-y-6">
      <div>
        <h1 className="text-xl md:text-2xl font-bold">{channel.name}</h1>
        <p className="text-xs md:text-sm text-[var(--loop-muted)] mt-1 break-all">{channel.base_url}</p>
        {channel.description && <p className="text-sm text-[var(--loop-muted)]">{channel.description}</p>}
      </div>
      <div className="rounded-xl border border-[var(--loop-border)] bg-[var(--loop-card)] p-5 space-y-2">
        <h2 className="text-sm font-medium">调用提示</h2>
        <div className="text-xs text-[var(--loop-muted)] space-y-1">
          <p className="break-all">外部调用此渠道：<span className="font-mono text-[var(--loop-text)]">/channel/{channel.id}/v1/messages</span></p>
          <p className="break-all">请求必须携带管理员令牌：<span className="font-mono text-[var(--loop-text)]">Authorization: Bearer &lt;adminToken&gt;</span> 或 <span className="font-mono text-[var(--loop-text)]">x-api-key: &lt;adminToken&gt;</span>。</p>
          <p>探测会消耗上游少量请求，但只记录到探测历史，不计入用量统计。</p>
        </div>
      </div>
      <div className="rounded-xl border border-[var(--loop-border)] bg-[var(--loop-card)] p-5 space-y-3">
        <div>
          <h2 className="text-sm font-medium">探测模型</h2>
          <p className="text-xs text-[var(--loop-muted)] mt-1">用于手动探测和自动恢复探测。端点不支持模型列表时，可以直接手填模型 ID；探测结果不会进入用量页。</p>
        </div>
        <div className="flex flex-col sm:flex-row flex-wrap gap-3">
          {modelOptions.length > 0 && (
            <select
              value={modelOptions.includes(probeModel) ? probeModel : ""}
              onChange={(e) => setProbeModel(e.target.value)}
              className="w-full sm:min-w-64 px-3 py-2 rounded-xl bg-[var(--loop-bg)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm"
            >
              <option value="">从模型列表选择</option>
              {modelOptions.map((model) => <option key={model} value={model}>{model}</option>)}
            </select>
          )}
          <input
            value={probeModel}
            onChange={(e) => setProbeModel(e.target.value)}
            placeholder="手动填写模型 ID"
            className="w-full sm:min-w-80 flex-1 px-3 py-2 rounded-xl bg-[var(--loop-bg)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm placeholder:text-[var(--loop-muted)] focus:outline-none focus:border-[var(--loop-primary)]"
          />
          <button onClick={handleLoadModels} disabled={modelLoading} className="px-4 py-2 rounded-xl border border-[var(--loop-border)] text-sm hover:bg-white/5 disabled:opacity-40">
            {modelLoading ? "获取中..." : "获取模型列表"}
          </button>
          <button onClick={handleSaveProbeModel} className="px-4 py-2 rounded-xl bg-[var(--loop-primary)] text-white text-sm hover:opacity-90">
            保存
          </button>
        </div>
        {modelMsg && <div className="text-xs text-[var(--loop-muted)]">{modelMsg}</div>}
      </div>
      <div className="grid grid-cols-3 gap-3 md:gap-4">
        <StatCard label="密钥总数" value={keys.length} />
        <StatCard label="可用密钥" value={activeKeys} color="text-green-400" />
        <StatCard label="停用密钥" value={keys.length - activeKeys} color="text-red-400" />
      </div>
      <div className="rounded-xl border border-[var(--loop-border)] bg-[var(--loop-card)] p-6">
        <div className="mb-4">
          <h2 className="text-sm font-medium">请求性能</h2>
          <p className="mt-1 text-xs text-[var(--loop-muted)]">基于此渠道最近 100 条业务请求，绘制首字耗时和输出速度。</p>
        </div>
        {performanceData.length > 0 ? (
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={performanceData}>
              <CartesianGrid stroke="var(--loop-chart-grid)" vertical={false} />
              <XAxis dataKey="time" tick={{ fill: "var(--loop-muted)", fontSize: 11 }} tickLine={false} axisLine={false} />
              <YAxis yAxisId="latency" tick={{ fill: "var(--loop-muted)", fontSize: 11 }} tickLine={false} axisLine={false} />
              <YAxis yAxisId="speed" orientation="right" tick={{ fill: "var(--loop-muted)", fontSize: 11 }} tickLine={false} axisLine={false} />
              <Tooltip
                contentStyle={{
                  background: "var(--loop-card)",
                  border: "1px solid var(--loop-border)",
                  borderRadius: 8,
                  color: "var(--loop-text)",
                  fontSize: 12,
                }}
                labelStyle={{ color: "var(--loop-muted)" }}
                formatter={(value, name) => [
                  name === "首字" ? `${Math.round(Number(value))}ms` : `${Number(value).toFixed(1)} tok/s`,
                  name,
                ]}
              />
              <Line yAxisId="latency" type="monotone" dataKey="first_token_ms" name="首字" stroke="#6366f1" strokeWidth={2} dot={{ r: 2 }} />
              <Line yAxisId="speed" type="monotone" dataKey="output_tokens_per_sec" name="输出速度" stroke="#0891b2" strokeWidth={2} dot={{ r: 2 }} />
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <div className="h-48 flex items-center justify-center text-center text-sm text-[var(--loop-muted)]">暂无业务请求性能数据</div>
        )}
      </div>
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <h2 className="text-lg font-semibold">API 密钥</h2>
        <div className="flex flex-wrap gap-2 sm:gap-3">
          <button onClick={() => setShowAddKey(true)} className="px-3 md:px-4 py-2 rounded-xl bg-[var(--loop-primary)] text-white text-sm font-medium hover:opacity-90 transition whitespace-nowrap">
            + 添加密钥
          </button>
          <button onClick={() => setShowImportKeys(true)} className="px-3 md:px-4 py-2 rounded-xl border border-[var(--loop-border)] text-sm hover:bg-white/5 whitespace-nowrap">
            导入
          </button>
          <button onClick={handleExportKeys} className="px-3 md:px-4 py-2 rounded-xl border border-[var(--loop-border)] text-sm hover:bg-white/5 whitespace-nowrap">
            导出
          </button>
        </div>
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
      {showAddKey && (
        <KeyFormModal
          title="添加 API 密钥"
          fixedChannelId={channelId}
          onClose={() => setShowAddKey(false)}
          onSubmit={handleAddKey}
        />
      )}
      {showImportKeys && (
        <KeyImportModal
          fixedChannelId={channelId}
          onClose={() => setShowImportKeys(false)}
          onImport={handleImportKeys}
        />
      )}
      {editingKey && (
        <KeyFormModal
          title="编辑 API 密钥"
          initialKey={editingKey}
          onClose={() => setEditingKey(null)}
          onSubmit={handleUpdateKey}
        />
      )}
      <ConfirmDialog open={delKeyId !== null} title="删除密钥" message="这会永久移除该 API 密钥。" onConfirm={handleDeleteKey} onCancel={() => setDelKeyId(null)} danger />
      {probeResult && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4" onClick={() => setProbeResult(null)}>
          <div className="bg-[var(--loop-card)] border border-[var(--loop-border)] rounded-2xl p-5 md:p-6 w-full max-w-sm" onClick={(e) => e.stopPropagation()}>
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

function buildPerformanceData(logs: UsageLog[]): ChannelPerformancePoint[] {
  return [...logs]
    .reverse()
    .filter((log) => log.first_token_ms > 0 || log.output_tokens_per_sec > 0)
    .map((log) => ({
      time: new Date(log.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
      first_token_ms: log.first_token_ms,
      output_tokens_per_sec: log.output_tokens_per_sec,
    }));
}

function downloadJSON(data: unknown, filename: string) {
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
