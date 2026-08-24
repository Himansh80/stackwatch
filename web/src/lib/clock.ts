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

/**
 * Time-of-day greeting. Datadog/Linear style: <12 "Good morning",
 * 12-17 "Good afternoon", 17-21 "Good evening", else "Working late".
 */
export function greetingFor(d: Date): string {
  const hour = d.getHours();
  if (hour < 5) return 'Working late';
  if (hour < 12) return 'Good morning';
  if (hour < 17) return 'Good afternoon';
  if (hour < 21) return 'Good evening';
  return 'Working late';
}

/**
 * Relative time formatter. "3 sec ago", "4 min ago", "1 hr ago",
 * "yesterday". Falls back to absolute time after a day.
 */
export function formatRelative(date: Date, now: Date): string {
  const seconds = Math.max(0, Math.floor((now.getTime() - date.getTime()) / 1000));
  if (seconds < 5) return 'just now';
  if (seconds < 60) return `${seconds} sec ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} min ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} hr ago`;
  return date.toLocaleString();
}

