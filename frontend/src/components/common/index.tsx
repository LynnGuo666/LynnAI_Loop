import { useEffect, useState, type ReactNode } from "react";
import type { APIKey, Channel } from "../../types";

interface StatCardProps {
  label: string;
  value: string | number;
  sub?: string;
  color?: string;
}

export function StatCard({ label, value, sub, color }: StatCardProps) {
  return (
    <div className="rounded-xl border border-[var(--loop-border)] bg-[var(--loop-card)] p-5">
      <div className="text-xs text-[var(--loop-muted)] uppercase tracking-wider mb-1">{label}</div>
      <div className={`text-2xl font-bold ${color || "text-[var(--loop-text)]"}`}>{value}</div>
      {sub && <div className="text-xs text-[var(--loop-muted)] mt-1">{sub}</div>}
    </div>
  );
}

interface Column<T> {
  key: string;
  label: string;
  render?: (row: T) => ReactNode;
}

interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  empty?: string;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function DataTable<T extends Record<string, any>>({ columns, data, empty }: DataTableProps<T>) {
  return (
    <div className="overflow-x-auto rounded-xl border border-[var(--loop-border)]">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-[var(--loop-border)] bg-[var(--loop-card)]">
            {columns.map((c) => (
              <th key={c.key} className="px-4 py-3 text-left text-xs font-medium text-[var(--loop-muted)] uppercase tracking-wider">
                {c.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="px-4 py-8 text-center text-[var(--loop-muted)]">
                {empty || "暂无数据"}
              </td>
            </tr>
          ) : (
            data.map((row, i) => (
              <tr key={i} className="border-b border-[var(--loop-border)] last:border-0 hover:bg-white/[0.02]">
                {columns.map((c) => (
                  <td key={c.key} className="px-4 py-3">
                    {c.render ? c.render(row) : String(row[c.key] ?? "")}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: string;
  onConfirm: () => void;
  onCancel: () => void;
  danger?: boolean;
}

export function ConfirmDialog({ open, title, message, onConfirm, onCancel, danger }: ConfirmDialogProps) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={onCancel}>
      <div className="bg-[var(--loop-card)] border border-[var(--loop-border)] rounded-2xl p-6 w-full max-w-sm" onClick={(e) => e.stopPropagation()}>
        <h3 className="text-lg font-semibold mb-2">{title}</h3>
        <p className="text-sm text-[var(--loop-muted)] mb-6">{message}</p>
        <div className="flex justify-end gap-3">
          <button onClick={onCancel} className="px-4 py-2 text-sm rounded-lg border border-[var(--loop-border)] hover:bg-white/5 transition">
            取消
          </button>
          <button onClick={onConfirm} className={`px-4 py-2 text-sm rounded-lg transition ${danger ? "bg-red-500 hover:bg-red-600 text-white" : "bg-[var(--loop-primary)] hover:opacity-90 text-white"}`}>
            确认
          </button>
        </div>
      </div>
    </div>
  );
}

interface KeyFormModalProps {
  title: string;
  channels?: Channel[];
  fixedChannelId?: number;
  initialKey?: APIKey | null;
  onClose: () => void;
  onSubmit: (data: { channelId: number; keyValue: string; alias: string }) => Promise<void> | void;
}

export function KeyFormModal({ title, channels = [], fixedChannelId, initialKey, onClose, onSubmit }: KeyFormModalProps) {
  const [channelId, setChannelId] = useState<number | "">(fixedChannelId ?? initialKey?.channel_id ?? "");
  const [keyValue, setKeyValue] = useState(initialKey?.key_value ?? "");
  const [alias, setAlias] = useState(initialKey?.alias ?? "");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const canChooseChannel = !initialKey && fixedChannelId === undefined;

  useEffect(() => {
    setChannelId(fixedChannelId ?? initialKey?.channel_id ?? "");
    setKeyValue(initialKey?.key_value ?? "");
    setAlias(initialKey?.alias ?? "");
    setError("");
    setSaving(false);
  }, [fixedChannelId, initialKey]);

  const handleSubmit = async () => {
    const trimmedKey = keyValue.trim();
    if (channelId === "") {
      setError("请选择渠道");
      return;
    }
    if (!trimmedKey) {
      setError("请填写 API Key");
      return;
    }

    setSaving(true);
    setError("");
    try {
      await onSubmit({ channelId: Number(channelId), keyValue: trimmedKey, alias: alias.trim() });
      onClose();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "";
      setError(msg.includes("already exists") || msg.includes("409") ? "这个 API Key 已存在，不能重复添加" : "保存失败");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={onClose}>
      <div className="bg-[var(--loop-card)] border border-[var(--loop-border)] rounded-2xl p-6 w-full max-w-md space-y-4" onClick={(e) => e.stopPropagation()}>
        <h2 className="text-lg font-semibold">{title}</h2>
        {canChooseChannel && (
          <select
            value={channelId}
            onChange={(e) => setChannelId(e.target.value === "" ? "" : Number(e.target.value))}
            className="w-full px-4 py-2.5 rounded-xl bg-[var(--loop-bg)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm focus:outline-none focus:border-[var(--loop-primary)]"
          >
            <option value="">选择渠道</option>
            {channels.map((channel) => (
              <option key={channel.id} value={channel.id}>{channel.name}</option>
            ))}
          </select>
        )}
        <input
          value={alias}
          onChange={(e) => setAlias(e.target.value)}
          placeholder="别名（可选）"
          className="w-full px-4 py-2.5 rounded-xl bg-[var(--loop-bg)] border border-[var(--loop-border)] text-[var(--loop-text)] placeholder:text-[var(--loop-muted)] focus:outline-none focus:border-[var(--loop-primary)]"
        />
        <input
          value={keyValue}
          onChange={(e) => setKeyValue(e.target.value)}
          placeholder="sk-ant-..."
          className="w-full px-4 py-2.5 rounded-xl bg-[var(--loop-bg)] border border-[var(--loop-border)] text-[var(--loop-text)] placeholder:text-[var(--loop-muted)] focus:outline-none focus:border-[var(--loop-primary)] font-mono text-sm"
        />
        {error && <div className="text-sm text-red-400">{error}</div>}
        <div className="flex justify-end gap-3">
          <button onClick={onClose} className="px-4 py-2 text-sm rounded-lg border border-[var(--loop-border)] hover:bg-white/5">取消</button>
          <button
            onClick={handleSubmit}
            disabled={saving || !keyValue.trim() || channelId === ""}
            className="px-4 py-2 text-sm rounded-lg bg-[var(--loop-primary)] text-white hover:opacity-90 disabled:opacity-40"
          >
            {saving ? "保存中..." : "保存"}
          </button>
        </div>
      </div>
    </div>
  );
}
