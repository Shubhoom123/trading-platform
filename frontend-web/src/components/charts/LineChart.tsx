// Dependency-free SVG line chart. Auto-scales one or more series sharing an
// index-based x-axis, draws smooth-ish polylines, a zero baseline when the data
// crosses zero, and start/end x labels.

export interface ChartSeries {
  values: number[];
  color: string;
  label?: string;
  fill?: boolean;
}

interface Props {
  series: ChartSeries[];
  xLabels?: [string, string]; // [first, last]
  height?: number;
  format?: (v: number) => string;
}

const W = 800; // viewBox width; scales responsively via CSS

export function LineChart({ series, xLabels, height = 240, format }: Props) {
  const all = series.flatMap((s) => s.values).filter((v) => Number.isFinite(v));
  if (all.length === 0) return <div className="chart-empty">no data</div>;

  const min = Math.min(...all);
  const max = Math.max(...all);
  const pad = (max - min) * 0.08 || 1;
  const lo = min - pad;
  const hi = max + pad;
  const n = Math.max(...series.map((s) => s.values.length));

  const x = (i: number) => (n <= 1 ? 0 : (i / (n - 1)) * W);
  const y = (v: number) => height - ((v - lo) / (hi - lo)) * height;

  const fmt = format ?? ((v: number) => v.toFixed(2));
  const zeroY = lo < 0 && hi > 0 ? y(0) : null;

  return (
    <div className="chart">
      <svg viewBox={`0 0 ${W} ${height}`} preserveAspectRatio="none" className="chart-svg">
        {zeroY !== null && (
          <line x1={0} x2={W} y1={zeroY} y2={zeroY} className="chart-zero" />
        )}
        {series.map((s, si) => {
          const pts = s.values
            .map((v, i) => (Number.isFinite(v) ? `${x(i)},${y(v)}` : null))
            .filter(Boolean)
            .join(" ");
          return (
            <g key={si}>
              {s.fill && (
                <polygon
                  points={`0,${height} ${pts} ${W},${height}`}
                  fill={s.color}
                  opacity={0.12}
                />
              )}
              <polyline points={pts} fill="none" stroke={s.color} strokeWidth={2} />
            </g>
          );
        })}
      </svg>

      <div className="chart-axis">
        <span>{fmt(hi)}</span>
        <span>{fmt(lo)}</span>
      </div>
      {xLabels && (
        <div className="chart-xaxis">
          <span>{xLabels[0]}</span>
          <span>{xLabels[1]}</span>
        </div>
      )}
      {series.some((s) => s.label) && (
        <div className="chart-legend">
          {series.map(
            (s, i) =>
              s.label && (
                <span key={i} className="legend-item">
                  <i style={{ background: s.color }} /> {s.label}
                </span>
              ),
          )}
        </div>
      )}
    </div>
  );
}
