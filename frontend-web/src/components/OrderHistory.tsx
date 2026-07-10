import { useEffect, useState } from "react";
import { api } from "../api/client";
import { formatTicks, type OrderResponse } from "../api/types";

// Lists the user's orders. Polls every 2s so async status changes (NEW ->
// FILLED, driven by the fill consumer) show up, and reloads immediately when
// `version` bumps after placing an order.
export function OrderHistory({ version }: { version: number }) {
  const [orders, setOrders] = useState<OrderResponse[]>([]);

  useEffect(() => {
    let active = true;
    const load = () => api.orders().then((o) => active && setOrders(o)).catch(() => {});
    load();
    const id = window.setInterval(load, 2000);
    return () => {
      active = false;
      window.clearInterval(id);
    };
  }, [version]);

  return (
    <div className="card history">
      <div className="card-head">
        <h2>Your Orders</h2>
        <span className="muted">{orders.length}</span>
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Side</th>
              <th>Type</th>
              <th className="num">Price</th>
              <th className="num">Qty</th>
              <th className="num">Filled</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {orders.map((o) => (
              <tr key={o.id}>
                <td>{o.id}</td>
                <td className={o.side === "BUY" ? "buy-text" : "sell-text"}>{o.side}</td>
                <td>{o.type}</td>
                <td className="num">{o.type === "MARKET" ? "—" : formatTicks(o.priceTicks)}</td>
                <td className="num">{o.quantity}</td>
                <td className="num">{o.filledQuantity}</td>
                <td>
                  <span className={`badge ${o.status.toLowerCase()}`}>{o.status}</span>
                </td>
              </tr>
            ))}
            {orders.length === 0 && (
              <tr>
                <td colSpan={7} className="book-empty">no orders yet</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
