import type {
  AccountResponse,
  BookSnapshot,
  OrderResponse,
  PlaceOrderRequest,
  TokenResponse,
} from "./types";

const API_BASE = import.meta.env.VITE_API_BASE ?? "http://localhost:8090";
export const WS_BASE = import.meta.env.VITE_WS_BASE ?? "ws://localhost:8090";

// --- token storage ----------------------------------------------------------
//
// Tokens live in localStorage for simplicity. The honest tradeoff: this is
// readable by any script on the page, so it's vulnerable to XSS. A hardened
// setup would keep the refresh token in an httpOnly cookie set by the server;
// the gateway doesn't issue cookies, so we accept localStorage here and keep
// the access token short-lived (the API defaults to 15 min).

const ACCESS_KEY = "tp.accessToken";
const REFRESH_KEY = "tp.refreshToken";

export const tokens = {
  access: () => localStorage.getItem(ACCESS_KEY),
  refresh: () => localStorage.getItem(REFRESH_KEY),
  set(t: TokenResponse) {
    localStorage.setItem(ACCESS_KEY, t.accessToken);
    localStorage.setItem(REFRESH_KEY, t.refreshToken);
  },
  clear() {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
  },
};

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

async function parseError(res: Response): Promise<string> {
  try {
    const body = await res.json();
    // Spring ProblemDetail uses "detail"; our gin handlers use "error".
    return body.detail || body.error || res.statusText;
  } catch {
    return res.statusText;
  }
}

interface RequestOpts {
  method?: string;
  body?: unknown;
  auth?: boolean; // attach the access token (default true)
  retry?: boolean; // internal: whether a refresh-retry is still allowed
}

async function request<T>(path: string, opts: RequestOpts = {}): Promise<T> {
  const { method = "GET", body, auth = true, retry = true } = opts;
  const headers: Record<string, string> = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (auth) {
    const token = tokens.access();
    if (token) headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(API_BASE + path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  // Transparently refresh once on a 401, then replay the request.
  if (res.status === 401 && auth && retry && tokens.refresh()) {
    const refreshed = await tryRefresh();
    if (refreshed) return request<T>(path, { ...opts, retry: false });
    tokens.clear();
  }

  if (!res.ok) throw new ApiError(res.status, await parseError(res));
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

async function tryRefresh(): Promise<boolean> {
  const refreshToken = tokens.refresh();
  if (!refreshToken) return false;
  try {
    const res = await fetch(API_BASE + "/api/auth/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refreshToken }),
    });
    if (!res.ok) return false;
    tokens.set((await res.json()) as TokenResponse);
    return true;
  } catch {
    return false;
  }
}

export const api = {
  async register(email: string, password: string): Promise<TokenResponse> {
    const t = await request<TokenResponse>("/api/auth/register", {
      method: "POST",
      body: { email, password },
      auth: false,
    });
    tokens.set(t);
    return t;
  },

  async login(email: string, password: string): Promise<TokenResponse> {
    const t = await request<TokenResponse>("/api/auth/login", {
      method: "POST",
      body: { email, password },
      auth: false,
    });
    tokens.set(t);
    return t;
  },

  logout() {
    tokens.clear();
  },

  placeOrder: (req: PlaceOrderRequest) =>
    request<OrderResponse>("/api/orders", { method: "POST", body: req }),

  orders: () => request<OrderResponse[]>("/api/orders"),

  account: () => request<AccountResponse>("/api/account"),

  book: (symbol: string, depth = 10) =>
    request<BookSnapshot>(`/api/book/${encodeURIComponent(symbol)}?depth=${depth}`),
};
