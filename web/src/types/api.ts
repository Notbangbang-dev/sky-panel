export interface User {
  id: string;
  email: string;
  username: string;
  role: "admin" | "user";
  totp_enabled: boolean;
  coins: number;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  user: User;
}

export interface Node {
  id: string;
  name: string;
  address: string;
  docker_socket: string;
  expires_at: string;
}

export interface CreateNodeResult extends Node {
  node_token: string;
}

export interface RotateNodeTokenResult {
  node_token: string;
  expires_at: string;
}

export interface NodeSummary {
  id: string;
  name: string;
  address: string;
  connected: boolean;
}

export interface Allocation {
  id: string;
  port: number;
  server_id?: string;
  server_name?: string;
}

export interface EggVariable {
  name: string;
  env: string;
  default: string;
  user_editable: boolean;
}

export interface Egg {
  id: string;
  name: string;
  category: string;
  description: string;
  docker_image: string;
  startup: string;
  stop_command: string;
  variables: EggVariable[];
}

export type ServerStatus = "installing" | "offline" | "running" | "stopping" | "errored";

export interface Server {
  id: string;
  owner_id: string;
  node_id: string;
  egg_id: string;
  name: string;
  status: ServerStatus;
  memory_bytes: number;
  cpu_limit: number;
  disk_bytes: number;
  primary_port: number;
  variables: Record<string, string>;
  backup_interval_hours: number;
  last_backup_at?: string;
}

export interface BackupEntry {
  filename: string;
  size_bytes: number;
  created_at: number;
}

export interface QuotaDims {
  memory_bytes: number;
  cpu_percent: number;
  disk_bytes: number;
}

export interface QuotaUsage extends QuotaDims {
  servers: number;
}

export interface QuotaInfo {
  usage: QuotaUsage;
  limit: QuotaDims;
}

export type StoreDimension = "memory" | "cpu" | "disk";

export interface StoreItem {
  id: string;
  name: string;
  description: string;
  dimension: StoreDimension;
  amount: number;
  price: number;
}

export interface HeartbeatResult {
  credited: number;
  balance: number;
  session_started_at: string;
}

export interface CoinResult {
  credited: number;
  balance: number;
}

export interface LedgerEntry {
  amount: number;
  reason: string;
  metadata?: string;
  created_at: string;
}

export interface Wallet {
  balance: number;
  history: LedgerEntry[];
}

export interface AuditEntry {
  actor_id: string;
  action: string;
  target?: string;
  metadata?: string;
  created_at: string;
}

export interface ContainerHeartbeat {
  server_id: string;
  running: boolean;
  cpu_percent: number;
  mem_used_bytes: number;
  mem_limit_bytes: number;
  net_rx_bytes: number;
  net_tx_bytes: number;
}

export interface TotpSetup {
  secret: string;
  url: string;
}

export const PERMISSIONS = ["console", "files", "power", "settings"] as const;
export type Permission = (typeof PERMISSIONS)[number];

export interface Subuser {
  user_id: string;
  permissions: Permission[];
}

export interface FileEntry {
  name: string;
  is_dir: boolean;
  size_bytes: number;
}

export interface ListFilesResult {
  entries: FileEntry[];
}

export interface ReadFileResult {
  content_base64: string;
  size_bytes: number;
}
