import { apiRequest } from "./api";
import type {
  Achievement,
  AdminQuotaInfo,
  AdminServer,
  Allocation,
  ApiKey,
  AuditEntry,
  BackupEntry,
  CoinResult,
  CreateNodeResult,
  Egg,
  EggVariable,
  HeartbeatResult,
  LeaderboardEntry,
  ListFilesResult,
  Node,
  NodeSummary,
  Permission,
  QuotaInfo,
  ReadFileResult,
  RedeemCode,
  RotateNodeTokenResult,
  Schedule,
  ScheduleAction,
  Session,
  Server,
  StoreItem,
  Subuser,
  TokenPair,
  TotpSetup,
  User,
  Wallet,
} from "../types/api";

export const accountApi = {
  changePassword: (current_password: string, new_password: string) =>
    apiRequest<void>("/api/v1/me/password", { method: "POST", body: { current_password, new_password } }),
  listSessions: (currentRefreshToken?: string) =>
    apiRequest<Session[]>(
      `/api/v1/me/sessions${currentRefreshToken ? `?current=${encodeURIComponent(currentRefreshToken)}` : ""}`,
    ),
  revokeSession: (id: string) => apiRequest<void>(`/api/v1/me/sessions/${id}`, { method: "DELETE" }),
  revokeOtherSessions: (current_refresh_token: string) =>
    apiRequest<void>("/api/v1/me/sessions/revoke-others", { method: "POST", body: { current_refresh_token } }),
  listApiKeys: () => apiRequest<ApiKey[]>("/api/v1/me/api-keys"),
  createApiKey: (name: string) =>
    apiRequest<{ name: string; key: string }>("/api/v1/me/api-keys", { method: "POST", body: { name } }),
  deleteApiKey: (id: string) => apiRequest<void>(`/api/v1/me/api-keys/${id}`, { method: "DELETE" }),
};

export const schedulesApi = {
  list: (serverId: string) => apiRequest<Schedule[]>(`/api/v1/servers/${serverId}/schedules`),
  create: (serverId: string, body: { name: string; action: ScheduleAction; payload?: string; interval_minutes: number }) =>
    apiRequest<Schedule>(`/api/v1/servers/${serverId}/schedules`, { method: "POST", body }),
  toggle: (serverId: string, id: string, enabled: boolean) =>
    apiRequest<void>(`/api/v1/servers/${serverId}/schedules/${id}/toggle`, { method: "POST", body: { enabled } }),
  remove: (serverId: string, id: string) =>
    apiRequest<void>(`/api/v1/servers/${serverId}/schedules/${id}`, { method: "DELETE" }),
};

export const leaderboardApi = {
  list: () => apiRequest<LeaderboardEntry[]>("/api/v1/leaderboard"),
};

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
    disk_bytes?: number;
    variables?: Record<string, string>;
  }) => apiRequest<Server>("/api/v1/servers", { method: "POST", body: input }),
  update: (
    id: string,
    input: {
      name: string;
      memory_bytes: number;
      cpu_limit: number;
      disk_bytes: number;
      variables?: Record<string, string>;
      backup_interval_hours: number;
    },
  ) => apiRequest<Server>(`/api/v1/servers/${id}`, { method: "PATCH", body: input }),
  reinstall: (id: string, eggId?: string) =>
    apiRequest<void>(`/api/v1/servers/${id}/reinstall`, { method: "POST", body: { egg_id: eggId ?? "" } }),
  activity: (id: string) => apiRequest<AuditEntry[]>(`/api/v1/servers/${id}/activity`),
  remove: (id: string) => apiRequest<void>(`/api/v1/servers/${id}`, { method: "DELETE" }),
  power: (id: string, action: "start" | "stop" | "kill") =>
    apiRequest<void>(`/api/v1/servers/${id}/power`, { method: "POST", body: { action } }),
  consoleInput: (id: string, input: string) =>
    apiRequest<void>(`/api/v1/servers/${id}/console`, { method: "POST", body: { input } }),
  setDescription: (id: string, description: string) =>
    apiRequest<void>(`/api/v1/servers/${id}/description`, { method: "PUT", body: { description } }),
  clone: (id: string) => apiRequest<Server>(`/api/v1/servers/${id}/clone`, { method: "POST" }),
  favorite: (id: string) => apiRequest<void>(`/api/v1/servers/${id}/favorite`, { method: "POST" }),
  unfavorite: (id: string) => apiRequest<void>(`/api/v1/servers/${id}/favorite`, { method: "DELETE" }),
};

export const favoritesApi = {
  list: () => apiRequest<string[]>("/api/v1/me/favorites"),
};

export const achievementsApi = {
  list: () => apiRequest<Achievement[]>("/api/v1/achievements"),
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
  heartbeat: (sessionId: string) =>
    apiRequest<HeartbeatResult>("/api/v1/afk/heartbeat", { method: "POST", body: { session_id: sessionId } }),
  claimDaily: () => apiRequest<CoinResult>("/api/v1/daily-reward/claim", { method: "POST" }),
  gift: (username: string, amount: number) =>
    apiRequest<CoinResult>("/api/v1/coins/gift", { method: "POST", body: { username, amount } }),
  redeem: (code: string) => apiRequest<CoinResult>("/api/v1/coins/redeem", { method: "POST", body: { code } }),
};

export const quotaApi = {
  mine: () => apiRequest<QuotaInfo>("/api/v1/me/quota"),
};

export const storeApi = {
  list: () => apiRequest<StoreItem[]>("/api/v1/store"),
  purchase: (itemId: string) =>
    apiRequest<{ item_id: string; balance: number }>("/api/v1/store/purchase", {
      method: "POST",
      body: { item_id: itemId },
    }),
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
  getUserQuota: (userId: string) => apiRequest<AdminQuotaInfo>(`/api/v1/admin/users/${userId}/quota`),
  setUserQuota: (userId: string, quota: { memory_bytes: number; cpu_percent: number; disk_bytes: number }) =>
    apiRequest<AdminQuotaInfo>(`/api/v1/admin/users/${userId}/quota`, { method: "PUT", body: quota }),
  impersonate: (userId: string) =>
    apiRequest<TokenPair>(`/api/v1/admin/users/${userId}/impersonate`, { method: "POST" }),

  listAllServers: () => apiRequest<AdminServer[]>("/api/v1/admin/servers"),
  transferServer: (id: string, ownerId: string) =>
    apiRequest<void>(`/api/v1/admin/servers/${id}/owner`, { method: "POST", body: { owner_id: ownerId } }),
  suspendServer: (id: string) => apiRequest<void>(`/api/v1/admin/servers/${id}/suspend`, { method: "POST" }),
  unsuspendServer: (id: string) => apiRequest<void>(`/api/v1/admin/servers/${id}/unsuspend`, { method: "POST" }),

  listRedeemCodes: () => apiRequest<RedeemCode[]>("/api/v1/admin/redeem-codes"),
  createRedeemCode: (code: string, coins: number, maxUses: number) =>
    apiRequest<RedeemCode>("/api/v1/admin/redeem-codes", { method: "POST", body: { code, coins, max_uses: maxUses } }),
  deleteRedeemCode: (id: string) => apiRequest<void>(`/api/v1/admin/redeem-codes/${id}`, { method: "DELETE" }),

  listNodes: () => apiRequest<Node[]>("/api/v1/admin/nodes"),
  createNode: (name: string, address: string) =>
    apiRequest<CreateNodeResult>("/api/v1/admin/nodes", { method: "POST", body: { name, address } }),
  deleteNode: (id: string) => apiRequest<void>(`/api/v1/admin/nodes/${id}`, { method: "DELETE" }),
  rotateNodeToken: (id: string) =>
    apiRequest<RotateNodeTokenResult>(`/api/v1/admin/nodes/${id}/rotate-token`, { method: "POST" }),

  listAllocations: (nodeId: string) => apiRequest<Allocation[]>(`/api/v1/admin/nodes/${nodeId}/allocations`),
  createAllocations: (nodeId: string, body: { port?: number; port_start?: number; port_end?: number }) =>
    apiRequest<{ created: number }>(`/api/v1/admin/nodes/${nodeId}/allocations`, { method: "POST", body }),
  deleteAllocation: (allocationId: string) =>
    apiRequest<void>(`/api/v1/admin/allocations/${allocationId}`, { method: "DELETE" }),

  createEgg: (input: EggInput) => apiRequest<Egg>("/api/v1/admin/eggs", { method: "POST", body: input }),
  updateEgg: (id: string, input: EggInput) => apiRequest<Egg>(`/api/v1/admin/eggs/${id}`, { method: "PUT", body: input }),
  exportEgg: (id: string) => apiRequest<EggInput>(`/api/v1/admin/eggs/${id}/export`),
  deleteEgg: (id: string) => apiRequest<void>(`/api/v1/admin/eggs/${id}`, { method: "DELETE" }),

  getSettings: () => apiRequest<Record<string, string>>("/api/v1/admin/settings"),
  setSetting: (key: string, value: string) =>
    apiRequest<void>(`/api/v1/admin/settings/${key}`, { method: "PUT", body: { value } }),

  auditLog: () => apiRequest<AuditEntry[]>("/api/v1/admin/audit-log"),
  broadcast: (message: string) => apiRequest<void>("/api/v1/admin/broadcast", { method: "POST", body: { message } }),
};
