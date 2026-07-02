import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { storeApi } from "../lib/endpoints";
import { useAuthStore } from "../lib/authStore";
import { ApiError } from "../lib/api";
import { QuotaMeters } from "../components/QuotaMeters";
import { formatBytes } from "../lib/format";
import type { StoreItem } from "../types/api";

const DIMENSION_GLYPH: Record<string, string> = { memory: "▮", cpu: "◈", disk: "▤" };

export function StorePage() {
  const queryClient = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const updateUser = useAuthStore((s) => s.updateUser);

  const { data: items } = useQuery({ queryKey: ["store"], queryFn: storeApi.list });
  const [pendingId, setPendingId] = useState<string | null>(null);
  const [message, setMessage] = useState<{ text: string; ok: boolean } | null>(null);

  const purchase = useMutation({
    mutationFn: (itemId: string) => storeApi.purchase(itemId),
    onMutate: (itemId) => setPendingId(itemId),
    onSuccess: (result, itemId) => {
      const current = useAuthStore.getState().user;
      if (current) updateUser({ ...current, coins: result.balance });
      queryClient.invalidateQueries({ queryKey: ["quota"] });
      const item = items?.find((i) => i.id === itemId);
      setMessage({ text: `Purchased ${item?.name ?? "upgrade"} — quota raised.`, ok: true });
    },
    onError: (err) =>
      setMessage({
        text:
          err instanceof ApiError && err.code === "insufficient_balance"
            ? "Not enough coins for that yet — earn more on the AFK page."
            : "Purchase failed.",
        ok: false,
      }),
    onSettled: () => setPendingId(null),
  });

  const balance = user?.coins ?? 0;

  return (
    <div>
      <p className="sp-kicker">Coin store</p>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-end", marginBottom: 18 }}>
        <h1 className="sp-page-title" style={{ marginBottom: 0 }}>
          Store
        </h1>
        <div style={{ textAlign: "right" }}>
          <span className="sp-stat__label" style={{ margin: 0 }}>
            your balance
          </span>
          <div className="sp-mono" style={{ fontSize: 24, fontVariantNumeric: "tabular-nums" }}>
            {balance.toLocaleString()} ⧫
          </div>
        </div>
      </div>

      <div className="sp-surface sp-card" style={{ marginBottom: 20 }}>
        <p className="sp-stat__label">Your quota</p>
        <QuotaMeters compact />
        <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 12 }}>
          Buying an upgrade permanently raises the matching limit. Spend the coins you earn idling on the AFK page.
        </p>
      </div>

      {message && (
        <p className="sp-mono" style={{ fontSize: 13, color: message.ok ? "var(--sp-accent)" : "#ff9b9b", marginBottom: 14 }}>
          {message.text}
        </p>
      )}

      <div className="sp-store-grid">
        {items?.map((item) => (
          <StoreCard
            key={item.id}
            item={item}
            affordable={balance >= item.price}
            pending={pendingId === item.id}
            onBuy={() => {
              setMessage(null);
              purchase.mutate(item.id);
            }}
          />
        ))}
      </div>
    </div>
  );
}

function amountLabel(item: StoreItem): string {
  if (item.dimension === "cpu") return `+${item.amount}% CPU`;
  return `+${formatBytes(item.amount)}`;
}

function StoreCard({
  item,
  affordable,
  pending,
  onBuy,
}: {
  item: StoreItem;
  affordable: boolean;
  pending: boolean;
  onBuy: () => void;
}) {
  return (
    <div className="sp-surface sp-card" style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        <span style={{ fontSize: 18, color: "var(--sp-text-muted)" }}>{DIMENSION_GLYPH[item.dimension] ?? "◆"}</span>
        <span style={{ fontFamily: "var(--sp-font-display)", fontSize: 22 }}>{amountLabel(item)}</span>
      </div>
      <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", margin: 0, flex: 1 }}>
        {item.description}
      </p>
      <div className="sp-store-price">
        {item.price.toLocaleString()}
        <span className="sp-store-price__unit">coins</span>
      </div>
      <button
        className="sp-btn sp-btn--primary"
        onClick={onBuy}
        disabled={pending || !affordable}
        title={affordable ? undefined : "Not enough coins"}
      >
        {pending ? "Buying…" : affordable ? "Buy" : "Can't afford"}
      </button>
    </div>
  );
}
