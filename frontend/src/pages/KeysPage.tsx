import { useEffect, useState } from "react";
import { listAllKeys, enableKey, probeKey, listChannels, deleteKey, createKey, updateKey, exportKeys, importKeys } from "../api/client";
import { DataTable, ConfirmDialog, KeyFormModal, KeyImportModal } from "../components/common";
import type { APIKey, Channel, KeyImportItem, KeyProbe } from "../types";

export function KeysPage() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [filterChannel, setFilterChannel] = useState<number | "">("");
  const [filterStatus, setFilterStatus] = useState<"all" | "active" | "disabled">("all");
  const [showAddKey, setShowAddKey] = useState(false);
  const [showImportKeys, setShowImportKeys] = useState(false);
  const [editingKey, setEditingKey] = useState<APIKey | null>(null);
  const [delKeyId, setDelKeyId] = useState<number | null>(null);
  const [probeResult, setProbeResult] = useState<KeyProbe | null>(null);

  const load = () => {
    listAllKeys().then(setKeys).catch(() => {});
    listChannels().then(setChannels).catch(() => {});
  };
  useEffect(() => { load(); }, []);

  const channelMap = new Map(channels.map((c) => [c.id, c.name]));

  const filtered = keys.filter((k) => {
    if (filterChannel !== "" && k.channel_id !== filterChannel) return false;
    if (filterStatus === "active" && !k.is_active) return false;
    if (filterStatus === "disabled" && k.is_active) return false;
    return true;
  });

  const handleEnable = async (id: number) => { await enableKey(id); load(); };
  const handleProbe = async (id: number) => { const r = await probeKey(id); setProbeResult(r); load(); };
  const handleDelete = async () => { if (delKeyId) await deleteKey(delKeyId); setDelKeyId(null); load(); };
  const handleAddKey = async (data: { channelId: number; keyValue: string; alias: string }) => {
    await createKey(data.channelId, { key_value: data.keyValue, alias: data.alias });
    load();
  };
  const handleImportKeys = async (data: { channelId?: number; keys: KeyImportItem[] }) => {
    const result = await importKeys({ channel_id: data.channelId, keys: data.keys });
    load();
    return result;
  };
  const handleExportKeys = async () => {
    const channelId = filterChannel === "" ? undefined : filterChannel;
    const exported = await exportKeys(channelId);
    downloadJSON(exported, `loop-keys${channelId ? `-channel-${channelId}` : ""}.json`);
  };
  const handleUpdateKey = async (data: { keyValue: string; alias: string }) => {
    if (!editingKey) return;
    await updateKey(editingKey.id, { key_value: data.keyValue, alias: data.alias });
    setEditingKey(null);
    load();
  };

  const handleBatchEnable = async () => {
    const disabled = filtered.filter((k) => !k.is_active);
    await Promise.all(disabled.map((k) => enableKey(k.id)));
    load();
  };

  const maskKey = (k: string) => (k.length <= 10 ? "****" : k.slice(0, 7) + "..." + k.slice(-4));

  const columns = [
    { key: "id", label: "ID" },
    {
      key: "channel_id",
      label: "渠道",
      render: (k: APIKey) => channelMap.get(k.channel_id) || `#${k.channel_id}`,
    },
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
          {!k.is_active && <button onClick={() => handleEnable(k.id)} className="text-green-400 hover:text-green-300">启用</button>}
          <button onClick={() => setEditingKey(k)} className="text-[var(--loop-primary)] hover:opacity-80">编辑</button>
          <button onClick={() => handleProbe(k.id)} className="text-blue-400 hover:text-blue-300">探测</button>
          <button onClick={() => setDelKeyId(k.id)} className="text-red-400 hover:text-red-300">删除</button>
        </div>
      ),
    },
  ];

  const disabledCount = filtered.filter((k) => !k.is_active).length;

  return (
    <div className="space-y-4 md:space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <h1 className="text-xl md:text-2xl font-bold">API 密钥</h1>
        <div className="flex flex-wrap gap-2 sm:gap-3">
          {disabledCount > 0 && (
            <button onClick={handleBatchEnable} className="px-3 md:px-4 py-2 rounded-xl bg-green-500/20 text-green-400 text-xs sm:text-sm font-medium hover:bg-green-500/30 transition whitespace-nowrap">
              启用全部停用密钥（{disabledCount}）
            </button>
          )}
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
      <div className="flex flex-wrap gap-2 md:gap-3">
        <select
          value={filterChannel}
          onChange={(e) => setFilterChannel(e.target.value === "" ? "" : Number(e.target.value))}
          className="px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm"
        >
          <option value="">全部渠道</option>
          {channels.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
        <select
          value={filterStatus}
          onChange={(e) => setFilterStatus(e.target.value as "all" | "active" | "disabled")}
          className="px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm"
        >
          <option value="all">全部状态</option>
          <option value="active">启用</option>
          <option value="disabled">停用</option>
        </select>
      </div>
      <DataTable columns={columns} data={filtered} empty="未找到密钥" />
      {showAddKey && (
        <KeyFormModal
          title="添加 API 密钥"
          channels={channels}
          onClose={() => setShowAddKey(false)}
          onSubmit={handleAddKey}
        />
      )}
      {showImportKeys && (
        <KeyImportModal
          channels={channels}
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
      <ConfirmDialog open={delKeyId !== null} title="删除密钥" message="确定要永久移除这个 API 密钥吗？" onConfirm={handleDelete} onCancel={() => setDelKeyId(null)} danger />
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

function downloadJSON(data: unknown, filename: string) {
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
