export interface HubSettings {
  hub_name: string;
  hub_region: string;
  hub_timezone: string;
  default_sync_interval: number;
  catalog_auto_publish: boolean;
}

export interface IAMSettings {
  sso_url: string;
  client_id: string;
  redirect_uri: string;
  role_mappings: RoleMapping[];
}

export interface RoleMapping {
  zitadel_group: string;
  hub_role: string;
  permissions: string;
}

export interface WebhookSettings {
  api_endpoint: string;
  webhook_url: string;
  event_subscriptions: string[];
}
