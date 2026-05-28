import { useEffect, useState } from "react";

type ToastType = "error" | "info" | "success";

export interface ToastPayload {
  message: string;
  type?: ToastType;
  durationMs?: number;
}

interface ToastItem extends Required<ToastPayload> {
  id: number;
}

export const TOAST_EVENT = "loop:toast";

export function showToast(payload: ToastPayload) {
  window.dispatchEvent(new CustomEvent<ToastPayload>(TOAST_EVENT, { detail: payload }));
}

export function ToastHost() {
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  useEffect(() => {
    const handleToast = (event: Event) => {
      const detail = (event as CustomEvent<ToastPayload>).detail;
      const item: ToastItem = {
        id: Date.now() + Math.random(),
        message: detail.message,
        type: detail.type ?? "info",
        durationMs: detail.durationMs ?? 3000,
      };

      setToasts((current) => [...current, item]);
      window.setTimeout(() => {
        setToasts((current) => current.filter((toast) => toast.id !== item.id));
      }, item.durationMs);
    };

    window.addEventListener(TOAST_EVENT, handleToast);
    return () => window.removeEventListener(TOAST_EVENT, handleToast);
  }, []);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed top-4 left-1/2 z-50 flex w-[calc(100%-2rem)] max-w-sm -translate-x-1/2 flex-col gap-2">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          role="status"
          className={`rounded-lg border px-4 py-3 text-sm shadow-xl backdrop-blur ${
            toast.type === "error"
              ? "border-red-400/40 bg-red-950/90 text-red-100"
              : toast.type === "success"
                ? "border-emerald-400/40 bg-emerald-950/90 text-emerald-100"
                : "border-white/15 bg-[var(--loop-card)]/95 text-[var(--loop-text)]"
          }`}
        >
          {toast.message}
        </div>
      ))}
    </div>
  );
}
