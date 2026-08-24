/**
 * Inline sparkline — a tiny SVG trend line used inside KPI cards.
 * Renders an N-point series as a polyline.
 *
 * Sized for a 60×24 box so it sits cleanly at the right edge of a
 * card. The tone prop maps to a stroke color so the line matches
 * the card's accent stripe.
 */
type SparklineTone = 'cyan' | 'indigo' | 'green' | 'amber' | 'red';

const TONE_COLORS: Record<SparklineTone, string> = {
  cyan: '#38bdf8',
  indigo: '#818cf8',
  green: '#34d399',
  amber: '#fbbf24',
  red: '#f87171',
};

export default function Sparkline({
  values,
  tone = 'cyan',
}: {
  values: number[];
  tone?: SparklineTone;
}) {
  if (!values.length) {
    return <div className="dash-spark dash-spark-empty" aria-hidden="true" />;
  }
  const max = Math.max(...values, 1);
  const min = Math.min(...values, 0);
  const range = Math.max(max - min, 1);
  const w = 60;
  const h = 24;
  const points = values
    .map((number, index) => {
      const x = (index / Math.max(values.length - 1, 1)) * w;
      const y = h - ((number - min) / range) * h;
      return `${x.toFixed(2)},${y.toFixed(2)}`;
    })
    .join(' ');
  const stroke = TONE_COLORS[tone] || TONE_COLORS.cyan;
  return (
    <svg className="dash-spark" viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none" aria-hidden="true">
      <polyline points={points} fill="none" stroke={stroke} strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}
