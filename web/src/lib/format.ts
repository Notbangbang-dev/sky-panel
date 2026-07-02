// Formatting helpers shared across the economy/quota UI.

const MB = 1024 * 1024;
const GB = 1024 * MB;

/** Human-readable byte size, e.g. 1610612736 -> "1.5 GB". */
export function formatBytes(bytes: number): string {
  if (bytes >= GB) return `${(bytes / GB).toFixed(bytes % GB === 0 ? 0 : 1)} GB`;
  if (bytes >= MB) return `${Math.round(bytes / MB)} MB`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${bytes} B`;
}

/** CPU percent-of-one-core, e.g. 150 -> "1.5 cores" (0 -> "unlimited"). */
export function formatCpu(percent: number): string {
  if (percent === 0) return "unlimited";
  return `${(percent / 100).toFixed(percent % 100 === 0 ? 0 : 1)} core${percent === 100 ? "" : "s"}`;
}

export const bytesPerMB = MB;
export const bytesPerGB = GB;
