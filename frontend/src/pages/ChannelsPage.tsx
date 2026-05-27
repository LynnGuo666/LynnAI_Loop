import { useEffect, useState } from "react";
import { listChannels, createChannel, deleteChannel, updateChannel } from "../api/client";
import { DataTable, ConfirmDialog } from "../components/common";
import type { Channel } from "../types";
import { useNavigate } from "react-router-dom";

export function ChannelsPage() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const [delId, setDelId] = useState<number | null>(null);
  const navigate = useNavigate();

  const load = () => listChannels().then(setChannels).catch(() => {});
  useEffect(() => { load(); }, []);

  const handleCreate = async (data: Partial<Channel>) => {
    await createChannel(data);
    setShowCreate(false);
    load();
  };

  const handleDelete = async () => {
    if (delId) await deleteChannel(delId);
    setDelId(null);
    load();
  };

  const toggleActive = async (ch: Channel) => {
    await updateChannel(ch.id, { ...ch, is_active: !ch.is_active });
    load();
  };

  const columns = [
    { key: "id", label: "ID" },
    {
      key: "name",
      label: "名称",
      render: (ch: Channel) => (
        <button onClick={() => navigate(`/channels/${ch.id}`)} className="text-[var(--loop-primary)] hover:underline font-medium">
          {ch.name}
        </button>
      ),
    },
    { key: "base_url", label: "基础地址" },
    { key: "probe_model", label: "探测模型", render: (ch: Channel) => ch.probe_model || "未设置" },
    { key: "description", label: "描述" },
    {
      key: "is_active",
      label: "状态",
      render: (ch: Channel) => (
        <button onClick={() => toggleActive(ch)} className={`px-2 py-0.5 rounded-full text-xs font-medium ${ch.is_active ? "bg-green-500/20 text-green-400" : "bg-red-500/20 text-red-400"}`}>
          {ch.is_active ? "启用" : "停用"}
        </button>
      ),
    },
    {
      key: "actions",
      label: "",
      render: (ch: Channel) => (
        <button onClick={() => setDelId(ch.id)} className="text-xs text-red-400 hover:text-red-300">
          删除
        </button>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">渠道</h1>
        <button onClick={() => setShowCreate(true)} className="px-4 py-2 rounded-xl bg-[var(--loop-primary)] text-white text-sm font-medium hover:opacity-90 transition">
          + 新建渠道
        </button>
      </div>
      <DataTable columns={columns} data={channels} empty="暂无渠道" />
      {showCreate && <CreateChannelModal onClose={() => setShowCreate(false)} onCreate={handleCreate} />}
      <ConfirmDialog open={delId !== null} title="删除渠道" message="这会永久删除该渠道及其所有密钥。" onConfirm={handleDelete} onCancel={() => setDelId(null)} danger />
    </div>
  );
}

function CreateChannelModal({ onClose, onCreate }: { onClose: () => void; onCreate: (d: Partial<Channel>) => void }) {
  const [name, setName] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [desc, setDesc] = useState("");
  const [probeModel, setProbeModel] = useState("");

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={onClose}>
      <div className="bg-[var(--loop-card)] border border-[var(--loop-border)] rounded-2xl p-6 w-full max-w-md space-y-4" onClick={(e) => e.stopPropagation()}>
        <h2 className="text-lg font-semibold">新建渠道</h2>
        <input value={name} onChange={(e) => setName(e.target.value)} placeholder="渠道名称" className="w-full px-4 py-2.5 rounded-xl bg-[var(--loop-bg)] border border-[var(--loop-border)] text-[var(--loop-text)] placeholder:text-[var(--loop-muted)] focus:outline-none focus:border-[var(--loop-primary)]" />
        <input value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} placeholder="基础地址（https://api.anthropic.com）" className="w-full px-4 py-2.5 rounded-xl bg-[var(--loop-bg)] border border-[var(--loop-border)] text-[var(--loop-text)] placeholder:text-[var(--loop-muted)] focus:outline-none focus:border-[var(--loop-primary)]" />
        <input value={probeModel} onChange={(e) => setProbeModel(e.target.value)} placeholder="探测模型 ID（可选，例如 claude-3-5-haiku-latest）" className="w-full px-4 py-2.5 rounded-xl bg-[var(--loop-bg)] border border-[var(--loop-border)] text-[var(--loop-text)] placeholder:text-[var(--loop-muted)] focus:outline-none focus:border-[var(--loop-primary)]" />
        <input value={desc} onChange={(e) => setDesc(e.target.value)} placeholder="描述（可选）" className="w-full px-4 py-2.5 rounded-xl bg-[var(--loop-bg)] border border-[var(--loop-border)] text-[var(--loop-text)] placeholder:text-[var(--loop-muted)] focus:outline-none focus:border-[var(--loop-primary)]" />
        <div className="flex justify-end gap-3">
          <button onClick={onClose} className="px-4 py-2 text-sm rounded-lg border border-[var(--loop-border)] hover:bg-white/5 transition">取消</button>
          <button onClick={() => onCreate({ name, base_url: baseUrl, description: desc, probe_model: probeModel.trim() })} disabled={!name || !baseUrl} className="px-4 py-2 text-sm rounded-lg bg-[var(--loop-primary)] text-white hover:opacity-90 transition disabled:opacity-40">创建</button>
        </div>
      </div>
    </div>
  );
}
