import type { ReactNode } from "react";

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
