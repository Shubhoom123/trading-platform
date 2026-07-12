// Client for the Python analytics service (real data + impact models).
// Separate from the gateway — this is a distinct analytics concern.

const SIM_BASE = import.meta.env.VITE_SIM_BASE ?? "http://localhost:8100";

export interface Bar {
  date: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}
export interface History {
  symbol: string;
  source: "stooq" | "synthetic";
  bars: Bar[];
}

export interface BacktestStats {
  totalReturnPct: number;
  buyHoldReturnPct: number;
  sharpe: number;
  maxDrawdownPct: number;
  trades: number;
  winRatePct: number;
  exposurePct: number;
}
export interface Backtest {
  symbol: string;
  source: string;
  params: { fast: number; slow: number; capital: number };
  equity: { date: string; value: number }[];
  buyHoldFinal: number;
  trades: { date: string; side: string; price: number; pnlPct?: number }[];
  stats: BacktestStats;
}

export interface Impact {
  symbol: string;
  source: string;
  side: string;
  quantity: number;
  refPrice: number;
  avgFillPrice: number;
  finalPrice: number;
  slippageBps: number;
  priceMovePct: number;
  participationPct: number;
  adv: number;
  dailyVolPct: number;
  notional: number;
  model: string;
}

export interface Scenario {
  symbol: string;
  source: string;
  shockPct: number;
  shockDate: string;
  base: { curve: { date: string; pnl: number }[]; finalPnl: number };
  shocked: { curve: { date: string; pnl: number }[]; finalPnl: number };
  impactPnl: number;
}

export interface MlTrain {
  symbol: string;
  source: string;
  model: string;
  metrics: {
    trainAccuracyPct: number;
    testAccuracyPct: number;
    baselineAccuracyPct: number;
    edgePct: number;
    nTrain: number;
    nTest: number;
    features: number;
  };
  signal: { testReturnPct: number; buyHoldReturnPct: number };
  test: { dates: string[]; strategyEquity: number[]; buyHoldEquity: number[] };
  topFeatures: { name: string; weight: number }[];
  latest: { asOf: string; direction: "UP" | "DOWN"; probabilityUp: number | null };
  disclaimer: string;
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(SIM_BASE + path);
  if (!res.ok) throw new Error(`sim ${res.status}`);
  return res.json() as Promise<T>;
}
async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(SIM_BASE + path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`sim ${res.status}`);
  return res.json() as Promise<T>;
}

export const sim = {
  history: (symbol: string, lookback = 500) =>
    get<History>(`/api/sim/history?symbol=${encodeURIComponent(symbol)}&lookback=${lookback}`),
  backtest: (req: { symbol: string; fast: number; slow: number; capital: number }) =>
    post<Backtest>("/api/sim/backtest", req),
  impact: (req: { symbol: string; side: string; quantity: number }) =>
    post<Impact>("/api/sim/impact", req),
  scenario: (req: {
    symbol: string;
    side: string;
    quantity: number;
    shockPct: number;
    fromFrac: number;
  }) => post<Scenario>("/api/sim/scenario", req),
  models: () => get<{ models: string[] }>("/api/sim/models"),
  train: (req: { symbol: string; model: string }) =>
    post<MlTrain>("/api/sim/train", req),
};
