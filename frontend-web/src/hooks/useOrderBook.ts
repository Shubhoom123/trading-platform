import { useEffect, useState } from "react";
import { api, ApiError } from "../api/client";
import type { BookSnapshot } from "../api/types";

// Polls the order book snapshot for a symbol. The gateway serves these from
// Redis with a short TTL, so a ~1s poll is cheap.
export function useOrderBook(symbol: string, intervalMs = 1000) {
  const [book, setBook] = useState<BookSnapshot | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    let timer: number;

    const tick = async () => {
      try {
        const snap = await api.book(symbol);
        if (active) {
          setBook(snap);
          setError(null);
        }
      } catch (e) {
        if (active) setError(e instanceof ApiError ? e.message : "book unavailable");
      } finally {
        if (active) timer = window.setTimeout(tick, intervalMs);
      }
    };
    tick();

    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [symbol, intervalMs]);

  return { book, error };
}
