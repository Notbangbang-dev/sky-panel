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
  suspended: boolean;
  status_message?: string;
  description: string;
  public_status: boolean;
}

export interface AdminServer extends Server {
  owner_username: string;
}

export interface AdminAnalytics {
  users: number;
  admins: number;
  servers: number;
  servers_by_status: Record<string, number>;
  suspended: number;
  servers_by_egg: Record<string, number>;
  nodes: number;
  nodes_connected: number;
  eggs: number;
  coins_in_circulation: number;
}

export interface RedeemCode {
  id: string;
  code: string;
  coins: number;
  max_uses: number;
  uses: number;
  created_at: string;
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
  databases: number;
}

export interface QuotaUsage extends QuotaDims {
  servers: number;
}

export interface QuotaInfo {
  usage: QuotaUsage;
  limit: QuotaDims;
  allow_unlimited_cpu: boolean;
}

export interface AdminQuotaInfo extends QuotaInfo {
  bonus: QuotaDims;
}

export type StoreDimension = "memory" | "cpu" | "disk" | "databases";

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

export interface Session {
  id: string;
  created_at: string;
  expires_at: string;
  current: boolean;
}

export interface ApiKey {
  id: string;
  name: string;
  last_used_at?: string;
  created_at: string;
}

export type ScheduleAction = "start" | "stop" | "restart" | "kill" | "backup" | "command";

export interface Schedule {
  id: string;
  name: string;
  action: ScheduleAction;
  payload?: string;
  interval_minutes: number;
  enabled: boolean;
  last_run_at?: string;
}

export interface LeaderboardEntry {
  rank: number;
  username: string;
  coins: number;
}

export interface Achievement {
  id: string;
  name: string;
  description: string;
  unlocked: boolean;
}

export interface ModrinthHit {
  project_id: string;
  slug: string;
  title: string;
  description: string;
  author: string;
  project_type: string;
  downloads: number;
  follows: number;
  categories: string[];
  versions: string[];
  icon_url: string;
}

export interface ModrinthSearchResult {
  hits: ModrinthHit[];
  total_hits: number;
}

export interface ModrinthVersionFile {
  url: string;
  filename: string;
  primary: boolean;
  size: number;
}

export interface ModrinthVersion {
  id: string;
  name: string;
  version_number: string;
  version_type: string;
  game_versions: string[];
  loaders: string[];
  files: ModrinthVersionFile[];
}

export interface PlayerInfo {
  players: string[];
  max: number;
  version: string;
}

export interface PublicServerStatus {
  name: string;
  online: boolean;
  players: string[];
  player_count: number;
  max_players: number;
  version: string;
  cpu_percent: number;
  mem_used_bytes: number;
  mem_limit_bytes: number;
}

export const PERMISSIONS = ["console", "files", "power", "settings", "databases"] as const;
export type Permission = (typeof PERMISSIONS)[number];

export interface Database {
  id: string;
  owner_id: string;
  server_id: string;
  node_id: string;
  name: string;
  username: string;
  password: string;
  host: string;
  port: number;
  created_at: string;
}

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
