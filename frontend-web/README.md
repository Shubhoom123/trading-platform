# frontend-web

A React + TypeScript (Vite) trading terminal for the platform. It talks to the
**Go gateway** as its single backend — auth, order placement, order/account
reads, the order book, and the live fills WebSocket all go through `:8090`.

## Features

- Register / login (JWT), with a transparent refresh-on-401 in the API client.
- Live order book (polled from the gateway's Redis-cached snapshot) with depth bars.
- Place limit/market buy & sell orders; inline success/error from the API.
- Your orders table, polled so async `NEW → FILLED` transitions appear.
- Live fills feed streamed over the gateway WebSocket.
- Account balance (new accounts get a demo balance so buys work immediately).

## Run (dev)

The gateway (and the rest of the stack) must be up — see the repo-root README.

```sh
cd frontend-web
cp .env.example .env.local     # defaults point at http://localhost:8090
npm install
npm run dev                    # http://localhost:5173
```

Register a user, then trade. Place a SELL and a crossing BUY on the same symbol
(e.g. `AAPL`) to see a fill stream in live and the orders flip to `FILLED`.

## Scripts

- `npm run dev` — Vite dev server
- `npm run build` — type-check (`tsc -b`) + production build to `dist/`
- `npm run typecheck` — types only

## Structure

```
src/
  api/        typed client against the gateway (client.ts) + wire types
  auth/       AuthContext (JWT storage, login/register/logout)
  hooks/      useOrderBook (poll), useFills (WebSocket)
  components/ LoginPage, TopBar, OrderBook, OrderForm, OrderHistory, FillsFeed, TradingView
```

## Notes

- Prices are integer "ticks" on the wire (1 tick = 0.01); the UI converts for display.
- Tokens are kept in `localStorage`. The honest tradeoff (XSS exposure) is noted
  in `api/client.ts`; a hardened setup would use an httpOnly refresh cookie.
- CORS is handled by the gateway (`CORS_ALLOWED_ORIGIN`, default `*`).
