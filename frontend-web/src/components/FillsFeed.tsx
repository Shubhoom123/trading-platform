import { formatTicks } from "../api/types";
import { useFills } from "../hooks/useFills";

// Live executions streamed over the WebSocket.
export function FillsFeed({ symbol }: { symbol: string }) {
  const { fills, status } = useFills(symbol);

  return (
    <div className="card fills">
      <div className="card-head">
        <h2>Live Fills</h2>
        <span className={`dot ${status}`} title={status}>
          {status === "open" ? "live" : status}
        </span>
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th className="num">Seq</th>
              <th className="num">Price</th>
              <th className="num">Qty</th>
              <th className="num">Maker</th>
              <th className="num">Taker</th>
            </tr>
          </thead>
          <tbody>
            {fills.map((f) => (
              <tr key={f.sequence}>
                <td className="num">{f.sequence}</td>
                <td className="num">{formatTicks(f.priceTicks)}</td>
                <td className="num">{f.quantity}</td>
                <td className="num">{f.makerOrderId}</td>
                <td className="num">{f.takerOrderId}</td>
              </tr>
            ))}
            {fills.length === 0 && (
              <tr>
                <td colSpan={5} className="book-empty">waiting for fills…</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
