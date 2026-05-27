export interface Channel {
  [key: string]: unknown;
  id: number;
  name: string;
  base_url: string;
  description: string;
  probe_model: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface APIKey {
  [key: string]: unknown;
  id: number;
  channel_id: number;
  key_value: string;
  alias: string;
  is_active: boolean;
  consecutive_failures: number;
  total_failures: number;
  total_successes: number;
  last_used_at: string | null;
  last_failure_at: string | null;
  disabled_at: string | null;
  next_probe_at: string | null;
  probe_backoff_min: number;
  created_at: string;
  updated_at: string;
}

export interface UsageLog {
  [key: string]: unknown;
  id: number;
  channel_id: number;
  api_key_id: number;
  model: string;
  endpoint: string;
  input_tokens: number;
  output_tokens: number;
  cache_creation_tokens: number;
  cache_read_tokens: number;
  is_stream: boolean;
  status_code: number;
  latency_ms: number;
  first_token_ms: number;
  output_tokens_per_sec: number;
  success: boolean;
  error_message: string;
  client_ip: string;
  created_at: string;
}

export interface UsageStats {
  total_requests: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_cache_tokens: number;
  success_count: number;
  failure_count: number;
  avg_first_token_ms: number;
  avg_output_tokens_per_sec: number;
}

export interface TimeseriesPoint {
  date: string;
  requests: number;
  input_tokens: number;
  output_tokens: number;
}

export interface KeyProbe {
  id: number;
  api_key_id: number;
  success: boolean;
  latency_ms: number;
  status_code: number;
  error_msg: string;
  created_at: string;
}
