import { apiRequest } from "./api";
import type {
  AuditEntry,
  BackupEntry,
  CoinResult,
  CreateNodeResult,
  Egg,
  EggVariable,
  ListFilesResult,
  Node,
  NodeSummary,
  Permission,
  ReadFileResult,
  RotateNodeTokenResult,
  Server,
  Subuser,
  TokenPair,
  TotpSetup,
  User,
  Wallet,
} from "../types/api";

export const authApi = {
  register: (email: string, username: string, password: string) =>
    apiRequest<TokenPair>("/api/v1/auth/register", { method: "POST", body: { email, username, password }, auth: false }),
  login: (email: string, password: string, totp_code?: string) =>
    apiRequest<TokenPair>("/api/v1/auth/login", { method: "POST", body: { email, password, totp_code }, auth: false }),
  logout: (refresh_token: string) =>
    apiRequest<void>("/api/v1/auth/logout", { method: "POST", body: { refresh_token }, auth: false }),
  me: () => apiRequest<User>("/api/v1/me"),
  totpSetup: () => apiRequest<TotpSetup>("/api/v1/me/totp/setup", { method: "POST" }),
  totpConfirm: (code: string) => apiRequest<void>("/api/v1/me/totp/confirm", { method: "POST", body: { code } }),
  totpDisable: (code: string) => apiRequest<void>("/api/v1/me/totp/disable", { method: "POST", body: { code } }),
  registrationStatus: () => apiRequest<{ enabled: boolean }>("/api/v1/public/registration-status", { auth: false }),
};

export const serversApi = {
  list: () => apiRequest<Server[]>("/api/v1/servers"),
  get: (id: string) => apiRequest<Server>(`/api/v1/servers/${id}`),
  create: (input: {
    node_id: string;
    egg_id: string;
    name: string;
    memory_bytes: number;
    cpu_limit?: number;
    variables?: Record<string, string>;
  }) => apiRequest<Server>("/api/v1/servers", { method: "POST", body: input }),
  update: (
    id: string,
    input: {
      name: string;
      memory_bytes: number;
      cpu_limit: number;
      variables?: Record<string, string>;
      backup_interval_hours: number;
    },
  ) => apiRequest<Server>(`/api/v1/servers/${id}`, { method: "PATCH", body: input }),
  reinstall: (id: string) => apiRequest<void>(`/api/v1/servers/${id}/reinstall`, { method: "POST" }),
  activity: (id: string) => apiRequest<AuditEntry[]>(`/api/v1/servers/${id}/activity`),
  remove: (id: string) => apiRequest<void>(`/api/v1/servers/${id}`, { method: "DELETE" }),
  power: (id: string, action: "start" | "stop" | "kill") =>
    apiRequest<void>(`/api/v1/servers/${id}/power`, { method: "POST", body: { action } }),
  consoleInput: (id: string, input: string) =>
    apiRequest<void>(`/api/v1/servers/${id}/console`, { method: "POST", body: { input } }),
};

export const backupsApi = {
  list: (serverId: string) =>
    apiRequest<{ backups: BackupEntry[] }>(`/api/v1/servers/${serverId}/backups`),
  create: (serverId: string) =>
    apiRequest<{ filename: string; size_bytes: number }>(`/api/v1/servers/${serverId}/backups`, { method: "POST" }),
  restore: (serverId: string, filename: string) =>
    apiRequest<void>(`/api/v1/servers/${serverId}/backups/restore`, { method: "POST", body: { filename } }),
  remove: (serverId: string, filename: string) =>
    apiRequest<void>(`/api/v1/servers/${serverId}/backups?filename=${encodeURIComponent(filename)}`, {
      method: "DELETE",
    }),
};

export const subusersApi = {
  list: (serverId: string) => apiRequest<Subuser[]>(`/api/v1/servers/${serverId}/subusers`),
  add: (serverId: string, username: string, permissions: Permission[]) =>
    apiRequest<Subuser>(`/api/v1/servers/${serverId}/subusers`, { method: "POST", body: { username, permissions } }),
  remove: (serverId: string, userId: string) =>
    apiRequest<void>(`/api/v1/servers/${serverId}/subusers/${userId}`, { method: "DELETE" }),
};

export const filesApi = {
  list: (serverId: string, path: string) =>
    apiRequest<ListFilesResult>(`/api/v1/servers/${serverId}/files?path=${encodeURIComponent(path)}`),
  read: (serverId: string, path: string) =>
    apiRequest<ReadFileResult>(`/api/v1/servers/${serverId}/files/content?path=${encodeURIComponent(path)}`),
  write: (serverId: string, path: string, contentBase64: string) =>
    apiRequest<void>(`/api/v1/servers/${serverId}/files/content`, {
      method: "PUT",
      body: { path, content_base64: contentBase64 },
    }),
  rename: (serverId: string, path: string, newPath: string) =>
    apiRequest<void>(`/api/v1/servers/${serverId}/files/rename`, { method: "POST", body: { path, new_path: newPath } }),
  remove: (serverId: string, path: string) =>
    apiRequest<void>(`/api/v1/servers/${serverId}/files?path=${encodeURIComponent(path)}`, { method: "DELETE" }),
  mkdir: (serverId: string, path: string) =>
    apiRequest<void>(`/api/v1/servers/${serverId}/files/mkdir`, { method: "POST", body: { path } }),
};

export const eggsApi = {
  list: () => apiRequest<Egg[]>("/api/v1/eggs"),
};

export const nodesApi = {
  list: () => apiRequest<NodeSummary[]>("/api/v1/nodes"),
};

export const coinsApi = {
  wallet: () => apiRequest<Wallet>("/api/v1/wallet"),
  heartbeat: () => apiRequest<CoinResult>("/api/v1/afk/heartbeat", { method: "POST" }),
  claimDaily: () => apiRequest<CoinResult>("/api/v1/daily-reward/claim", { method: "POST" }),
};

export interface EggInput {
  name: string;
  docker_image: string;
  startup: string;
  category?: string;
  description?: string;
  stop_command?: string;
  variables?: EggVariable[];
}

export const adminApi = {
  listUsers: () => apiRequest<User[]>("/api/v1/admin/users"),
  setUserRole: (userId: string, role: "admin" | "user") =>
    apiRequest<void>(`/api/v1/admin/users/${userId}/role`, { method: "POST", body: { role } }),
  deleteUser: (userId: string) => apiRequest<void>(`/api/v1/admin/users/${userId}`, { method: "DELETE" }),
  adjustCoins: (userId: string, amount: number, note?: string) =>
    apiRequest<CoinResult>(`/api/v1/admin/users/${userId}/coins/adjust`, { method: "POST", body: { amount, note } }),

  listNodes: () => apiRequest<Node[]>("/api/v1/admin/nodes"),
  createNode: (name: string, address: string) =>
    apiRequest<CreateNodeResult>("/api/v1/admin/nodes", { method: "POST", body: { name, address } }),
  deleteNode: (id: string) => apiRequest<void>(`/api/v1/admin/nodes/${id}`, { method: "DELETE" }),
  rotateNodeToken: (id: string) =>
    apiRequest<RotateNodeTokenResult>(`/api/v1/admin/nodes/${id}/rotate-token`, { method: "POST" }),

  createEgg: (input: EggInput) => apiRequest<Egg>("/api/v1/admin/eggs", { method: "POST", body: input }),
  updateEgg: (id: string, input: EggInput) => apiRequest<Egg>(`/api/v1/admin/eggs/${id}`, { method: "PUT", body: input }),
  deleteEgg: (id: string) => apiRequest<void>(`/api/v1/admin/eggs/${id}`, { method: "DELETE" }),

  getSettings: () => apiRequest<Record<string, string>>("/api/v1/admin/settings"),
  setSetting: (key: string, value: string) =>
    apiRequest<void>(`/api/v1/admin/settings/${key}`, { method: "PUT", body: { value } }),

  auditLog: () => apiRequest<AuditEntry[]>("/api/v1/admin/audit-log"),
  broadcast: (message: string) => apiRequest<void>("/api/v1/admin/broadcast", { method: "POST", body: { message } }),
};
