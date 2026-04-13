/** Matches Go struct: inventory.ExtendedPackageInfo */
export interface ExtendedPackageInfo {
  name: string;
  version: string;
  architecture: string;
  source: string;
  status: string;
  installed_size_kb?: number;
  maintainer?: string;
  section?: string;
  homepage?: string;
  description?: string;
  install_date?: string;
  license?: string;
  priority?: string;
  source_package?: string;
  category?: string;
}

/** Matches Go struct: inventory.ServiceInfo */
export interface ServiceInfo {
  name: string;
  description: string;
  load_state: string;
  active_state: string;
  sub_state: string;
  enabled: boolean;
  category?: string;
}
