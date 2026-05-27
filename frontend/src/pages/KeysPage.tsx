import { useEffect, useState } from "react";
import { listAllKeys, enableKey, probeKey, listChannels, deleteKey } from "../api/client";
import { DataTable, ConfirmDialog } from "../components/common";
import type { APIKey, Channel, KeyProbe } from "../types";

export function KeysPage() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [filterChannel, setFilterChannel] = useState<number | "">("");
  const [filterStatus, setFilterStatus] = useState<"all" | "active" | "disabled">("all");
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
      label: "Channel",
      render: (k: APIKey) => channelMap.get(k.channel_id) || `#${k.channel_id}`,
    },
    { key: "alias", label: "Alias" },
    {
      key: "key_value",
      label: "Key",
      render: (k: APIKey) => <span className="font-mono text-xs text-[var(--loop-muted)]">{maskKey(k.key_value)}</span>,
    },
    {
      key: "is_active",
      label: "Status",
      render: (k: APIKey) => (
        <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${k.is_active ? "bg-green-500/20 text-green-400" : "bg-red-500/20 text-red-400"}`}>
          {k.is_active ? "Active" : "Disabled"}
        </span>
      ),
    },
    { key: "consecutive_failures", label: "Failures" },
    { key: "total_successes", label: "Successes" },
    {
      key: "actions",
      label: "",
      render: (k: APIKey) => (
        <div className="flex gap-2 text-xs">
          {!k.is_active && <button onClick={() => handleEnable(k.id)} className="text-green-400 hover:text-green-300">Enable</button>}
          <button onClick={() => handleProbe(k.id)} className="text-blue-400 hover:text-blue-300">Probe</button>
          <button onClick={() => setDelKeyId(k.id)} className="text-red-400 hover:text-red-300">Delete</button>
        </div>
      ),
    },
  ];

  const disabledCount = filtered.filter((k) => !k.is_active).length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">API Keys</h1>
        {disabledCount > 0 && (
          <button onClick={handleBatchEnable} className="px-4 py-2 rounded-xl bg-green-500/20 text-green-400 text-sm font-medium hover:bg-green-500/30 transition">
            Enable All Disabled ({disabledCount})
          </button>
        )}
      </div>
      <div className="flex gap-3">
        <select
          value={filterChannel}
          onChange={(e) => setFilterChannel(e.target.value === "" ? "" : Number(e.target.value))}
          className="px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm"
        >
          <option value="">All Channels</option>
          {channels.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
        <select
          value={filterStatus}
          onChange={(e) => setFilterStatus(e.target.value as "all" | "active" | "disabled")}
          className="px-3 py-2 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm"
        >
          <option value="all">All Status</option>
          <option value="active">Active</option>
          <option value="disabled">Disabled</option>
        </select>
      </div>
      <DataTable columns={columns} data={filtered} empty="No keys found" />
      <ConfirmDialog open={delKeyId !== null} title="Delete Key" message="Permanently remove this API key?" onConfirm={handleDelete} onCancel={() => setDelKeyId(null)} danger />
      {probeResult && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setProbeResult(null)}>
          <div className="bg-[var(--loop-card)] border border-[var(--loop-border)] rounded-2xl p-6 w-full max-w-sm" onClick={(e) => e.stopPropagation()}>
            <h3 className="text-lg font-semibold mb-3">Probe Result</h3>
            <div className="space-y-2 text-sm">
              <div>Success: <span className={probeResult.success ? "text-green-400" : "text-red-400"}>{probeResult.success ? "Yes" : "No"}</span></div>
              <div>Status: {probeResult.status_code}</div>
              <div>Latency: {probeResult.latency_ms}ms</div>
              {probeResult.error_msg && <div className="text-red-400">Error: {probeResult.error_msg}</div>}
            </div>
            <button onClick={() => setProbeResult(null)} className="mt-4 px-4 py-2 text-sm rounded-lg border border-[var(--loop-border)] hover:bg-white/5">Close</button>
          </div>
        </div>
      )}
    </div>
  );
}
