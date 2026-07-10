// Wire types — kept in sync with the gateway/Java DTOs. Prices are integer
// "ticks" everywhere on the wire (1 tick = 0.01); the UI converts for display.

export type Side = "BUY" | "SELL";
export type OrderType = "LIMIT" | "MARKET";
export type OrderStatus = "NEW" | "PARTIALLY_FILLED" | "FILLED" | "REJECTED";

export interface TokenResponse {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
}

export interface OrderResponse {
  id: number;
  symbol: string;
  side: Side;
  type: OrderType;
  priceTicks: number;
  quantity: number;
  filledQuantity: number;
  status: OrderStatus;
  createdAt: string;
}

export interface BookLevel {
  priceTicks: number;
  quantity: number;
}

export interface BookSnapshot {
  symbol: string;
  bids: BookLevel[];
  asks: BookLevel[];
  bestBid?: number | null;
  bestAsk?: number | null;
}

export interface AccountResponse {
  accountId: number;
  balanceTicks: number;
}

// Live fill pushed over the WebSocket.
export interface Fill {
  type: "fill";
  symbol: string;
  priceTicks: number;
  quantity: number;
  makerOrderId: number;
  takerOrderId: number;
  sequence: number;
}

export interface PlaceOrderRequest {
  symbol: string;
  side: Side;
  type: OrderType;
  price?: number | null; // decimal; required for LIMIT
  quantity: number;
}

// 1 tick = 0.01 of the quote currency (matches the engine + Java tick scale).
export const TICK = 0.01;
export const ticksToPrice = (ticks: number): number => ticks * TICK;
export const formatTicks = (ticks: number): string => (ticks * TICK).toFixed(2);
