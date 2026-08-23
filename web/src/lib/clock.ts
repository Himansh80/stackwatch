// Shared formatters for date/time. Used by the topbar clock widget
// on the Dashboard and Profile pages so the user sees the same
// format in both places.

export function formatClock(d: Date): string {
  // 12-hour clock with leading zero minutes, no seconds.
  let h = d.getHours();
  const m = d.getMinutes().toString().padStart(2, '0');
  const period = h >= 12 ? 'PM' : 'AM';
  h = h % 12;
  if (h === 0) h = 12;
  return `${h.toString().padStart(2, '0')}:${m} ${period}`;
}

export function formatDayLabel(d: Date): string {
  // "Sunday, 23 Aug 2026"
  const weekday = d.toLocaleDateString('en-US', { weekday: 'short' });
  const day = d.toLocaleDateString('en-US', { day: '2-digit' });
  const month = d.toLocaleDateString('en-US', { month: 'short' });
  const year = d.toLocaleDateString('en-US', { year: 'numeric' });
  return `${weekday}, ${day} ${month} ${year}`;
}
