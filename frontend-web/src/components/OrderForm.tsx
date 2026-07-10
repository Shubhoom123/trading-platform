import { useState, type FormEvent } from "react";
import { api, ApiError } from "../api/client";
import type { OrderType, Side } from "../api/types";

export function OrderForm({
  symbol,
  onPlaced,
}: {
  symbol: string;
  onPlaced: () => void;
}) {
  const [side, setSide] = useState<Side>("BUY");
  const [type, setType] = useState<OrderType>("LIMIT");
  const [price, setPrice] = useState("100.00");
  const [quantity, setQuantity] = useState("5");
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setMsg(null);
    try {
      const order = await api.placeOrder({
        symbol,
        side,
        type,
        price: type === "LIMIT" ? Number(price) : null,
        quantity: Number(quantity),
      });
      setMsg({ ok: true, text: `Order #${order.id} ${order.status}` });
      onPlaced();
    } catch (err) {
      setMsg({
        ok: false,
        text: err instanceof ApiError ? err.message : "Order failed",
      });
    } finally {
      setBusy(false);
    }
  };

  return (
    <form className="card order-form" onSubmit={submit}>
      <div className="card-head">
        <h2>Place Order</h2>
        <span className="muted">{symbol}</span>
      </div>

      <div className="seg">
        <button
          type="button"
          className={`seg-btn buy ${side === "BUY" ? "active" : ""}`}
          onClick={() => setSide("BUY")}
        >
          Buy
        </button>
        <button
          type="button"
          className={`seg-btn sell ${side === "SELL" ? "active" : ""}`}
          onClick={() => setSide("SELL")}
        >
          Sell
        </button>
      </div>

      <label>
        Type
        <select value={type} onChange={(e) => setType(e.target.value as OrderType)}>
          <option value="LIMIT">Limit</option>
          <option value="MARKET">Market</option>
        </select>
      </label>

      {type === "LIMIT" && (
        <label>
          Price
          <input
            type="number"
            step="0.01"
            min="0.01"
            value={price}
            onChange={(e) => setPrice(e.target.value)}
            required
          />
        </label>
      )}

      <label>
        Quantity
        <input
          type="number"
          step="1"
          min="1"
          value={quantity}
          onChange={(e) => setQuantity(e.target.value)}
          required
        />
      </label>

      <button
        className={`btn ${side === "BUY" ? "buy" : "sell"}`}
        type="submit"
        disabled={busy}
      >
        {busy ? "…" : `${side === "BUY" ? "Buy" : "Sell"} ${symbol}`}
      </button>

      {msg && <p className={msg.ok ? "ok" : "error"}>{msg.text}</p>}
    </form>
  );
}
