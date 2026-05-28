import type {
  Channel,
  APIKey,
  UsageLog,
  UsageStats,
  TimeseriesPoint,
  KeyProbe,
  KeyExportResponse,
  KeyImportItem,
  KeyImportResponse,
} from "../types";
import { addToast } from "@heroui/toast";
import { useAuthStore } from "../stores/auth";

const BASE = "";
let lastUnauthorizedNoticeAt = 0;

function getToken(): string {
  return localStorage.getItem("loop_token") || "";
}

function handleUnauthorized() {
  const now = Date.now();
  if (now - lastUnauthorizedNoticeAt < 2000) return;

  lastUnauthorizedNoticeAt = now;
  useAuthStore.getState().logout();
  addToast({
    title: "登录密钥已失效",
    description: "请重新登录",
    color: "danger",
    severity: "danger",
    timeout: 3000,
  });

  if (window.location.pathname !== "/login") {
    window.setTimeout(() => {
      window.location.assign("/login");
    }, 600);
  }
}

async function request<T>(
  path: string,
  opts: RequestInit = {}
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    Authorization: `Bearer ${getToken()}`,
    ...(opts.headers as Record<string, string> || {}),
  };
  const res = await fetch(`${BASE}${path}`, { ...opts, headers });
  if (!res.ok) {
    if (res.status === 401) {
      handleUnauthorized();
    }
    const body = await res.text();
    throw new Error(`${res.status}: ${body}`);
  }
  return res.json();
}

async function requestArray<T>(
  path: string,
  opts: RequestInit = {}
): Promise<T[]> {
  const data = await request<T[] | null>(path, opts);
  return Array.isArray(data) ? data : [];
}

// Channels
export const listChannels = () => requestArray<Channel>("/api/channels");
export const createChannel = (data: Partial<Channel>) =>
  request<Channel>("/api/channels", { method: "POST", body: JSON.stringify(data) });
export const getChannel = (id: number) => request<Channel>(`/api/channels/${id}`);
export const updateChannel = (id: number, data: Partial<Channel>) =>
  request<Channel>(`/api/channels/${id}`, { method: "PUT", body: JSON.stringify(data) });
export const deleteChannel = (id: number) =>
  request<{ status: string }>(`/api/channels/${id}`, { method: "DELETE" });
export const listChannelModels = (id: number) =>
  request<{ data?: Array<{ id?: string }> }>(`/channel/${id}/v1/models`).then((r) =>
    Array.isArray(r.data)
      ? r.data.map((m) => m.id?.trim()).filter((id): id is string => Boolean(id))
      : []
  );

// Keys
export const listKeysByChannel = (channelId: number) =>
  requestArray<APIKey>(`/api/channels/${channelId}/keys`);
export const listAllKeys = () => requestArray<APIKey>("/api/keys");
export const createKey = (channelId: number, data: Partial<APIKey>) =>
  request<APIKey>(`/api/channels/${channelId}/keys`, { method: "POST", body: JSON.stringify(data) });
export const getKey = (id: number) => request<APIKey>(`/api/keys/${id}`);
export const updateKey = (id: number, data: Partial<APIKey>) =>
  request<APIKey>(`/api/keys/${id}`, { method: "PUT", body: JSON.stringify(data) });
export const exportKeys = (channelId?: number) => {
  const params = new URLSearchParams();
  if (channelId) params.set("channel_id", String(channelId));
  const suffix = params.toString();
  return request<KeyExportResponse>(`/api/keys/export${suffix ? `?${suffix}` : ""}`);
};
export const importKeys = (data: { channel_id?: number; keys: KeyImportItem[] }) =>
  request<KeyImportResponse>("/api/keys/import", { method: "POST", body: JSON.stringify(data) });
export const deleteKey = (id: number) =>
  request<{ status: string }>(`/api/keys/${id}`, { method: "DELETE" });
export const enableKey = (id: number) =>
  request<APIKey>(`/api/keys/${id}/enable`, { method: "POST" });
export const probeKey = (id: number) =>
  request<KeyProbe>(`/api/keys/${id}/probe`, { method: "POST" });

// Usage
export interface UsageFilter {
  channel_id?: number;
  api_key_id?: number;
  success?: boolean;
  status?: string;
  start_date?: string;
  end_date?: string;
  model?: string;
  page?: number;
  page_size?: number;
}

export const listUsage = (f: UsageFilter = {}) => {
  const params = new URLSearchParams();
  Object.entries(f).forEach(([k, v]) => {
    if (v !== undefined && v !== "") params.set(k, String(v));
  });
  return request<{ data: UsageLog[] | null; total: number; page: number }>(
    `/api/usage?${params}`
  ).then((r) => ({ ...r, data: Array.isArray(r.data) ? r.data : [] }));
};
export const getUsageStats = (startDate?: string, endDate?: string) => {
  const params = new URLSearchParams();
  if (startDate) params.set("start_date", startDate);
  if (endDate) params.set("end_date", endDate);
  return request<UsageStats>(`/api/usage/stats?${params}`);
};
export const getUsageTimeseries = (days = 7) =>
  requestArray<TimeseriesPoint>(`/api/usage/timeseries?days=${days}`);
export const getUsageModels = () => requestArray<string>("/api/usage/models");

// Settings
export const getSettings = () => request<Record<string, string>>("/api/settings");
export const updateSettings = (data: Record<string, string>) =>
  request<Record<string, string>>("/api/settings", { method: "PUT", body: JSON.stringify(data) });

// Health
export const healthz = () => request<{ status: string }>("/api/healthz");
