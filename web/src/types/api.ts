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
}

export interface CreateNodeResult extends Node {
  node_token: string;
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
  primary_port: number;
  variables: Record<string, string>;
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
