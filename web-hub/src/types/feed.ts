export interface Feed {
  id: string;
  name: string;
  display_name: string;
  enabled: boolean;
  sync_interval_seconds: number;
  last_sync_at: string | null;
  next_sync_at: string | null;
  status: string;
  error_count: number;
  last_error: string | null;
  entries_ingested: number;
  url: string;
  auth_type: string;
  severity_filter: string[];
  os_filter: string[];
  severity_mapping: Record<string, string>;
  recent_history: { status: string; started_at: string }[];
  new_this_week: number;
  error_rate: number;
}

export interface FeedSyncRun {
  id: string;
  started_at: string;
  finished_at: string | null;
  duration_ms: number | null;
  new_entries: number;
  updated_entries: number;
  total_scanned: number;
  error_count: number;
  status: string;
  error_message: string | null;
  log_output: string | null;
}

export interface FeedHistoryResponse {
  runs: FeedSyncRun[];
  total: number;
  limit: number;
  offset: number;
}

export interface UpdateFeedRequest {
  enabled?: boolean;
  sync_interval_seconds?: number;
  url?: string;
  auth_type?: string;
  severity_filter?: string[];
  os_filter?: string[];
  severity_mapping?: Record<string, string>;
}
