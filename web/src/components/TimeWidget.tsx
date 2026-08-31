import { useEffect, useMemo, useRef, useState } from 'react';
import { formatClock, formatDayLabel } from '../lib/clock';
import Button from './shared/Button';

// Indian public + popular optional holidays for 2026.
// Source: official Government of India list (gazetted) plus
// commonly observed observances. Year-specific dates where
// the holiday uses a lunar/solar calendar are tagged with the
// expected Gregorian date for 2026.
//
// Each entry: { date: 'YYYY-MM-DD', name, kind: 'public' | 'observance' }
//   - public: gazetted holiday, marked with full color + star
//   - observance: widely observed but not a gazetted day off
export const INDIA_HOLIDAYS_2026: ReadonlyArray<{ date: string; name: string; kind: 'public' | 'observance' }> = [
  // Public holidays
  { date: '2026-01-26', name: 'Republic Day', kind: 'public' },
  { date: '2026-03-25', name: 'Holi', kind: 'public' },
  { date: '2026-04-03', name: 'Good Friday', kind: 'public' },
  { date: '2026-08-15', name: 'Independence Day', kind: 'public' },
  { date: '2026-10-02', name: 'Gandhi Jayanti', kind: 'public' },
  { date: '2026-10-19', name: 'Dussehra', kind: 'public' },
  { date: '2026-11-08', name: 'Diwali', kind: 'public' },
  { date: '2026-12-25', name: 'Christmas Day', kind: 'public' },
  // Observances
  { date: '2026-01-01', name: 'New Year\u2019s Day', kind: 'observance' },
  { date: '2026-01-14', name: 'Makar Sankranti', kind: 'observance' },
  { date: '2026-02-14', name: 'Valentine\u2019s Day', kind: 'observance' },
  { date: '2026-04-14', name: 'Baisakhi / Dr. Ambedkar Jayanti', kind: 'observance' },
  { date: '2026-05-01', name: 'Labour Day', kind: 'observance' },
  { date: '2026-08-26', name: 'Raksha Bandhan', kind: 'observance' },
  { date: '2026-09-05', name: 'Teachers\u2019 Day', kind: 'observance' },
  { date: '2026-11-14', name: 'Children\u2019s Day', kind: 'observance' },
];

interface TimeWidgetProps {
  /** Optional country override. Default 'IN' (India) per user locale. */
  country?: 'IN';
}

/**
 * Compact clock pill in the topbar that expands into a popover on
 * click. Popover shows:
 *   - Live HH:MM:SS AM/PM clock (updates every second)
 *   - Full date with weekday + month name + year
 *   - Mini month calendar with prev/next month + year navigation
 *   - Today highlighted; holidays marked with green dot or star
 *
 * Click outside or press Escape to close.
 */
export default function TimeWidget({ country = 'IN' }: TimeWidgetProps) {
  const [now, setNow] = useState<Date>(() => new Date());
  const [open, setOpen] = useState(false);
  const [viewMonth, setViewMonth] = useState<number>(() => new Date().getMonth());
  const [viewYear, setViewYear] = useState<number>(() => new Date().getFullYear());
  const popoverRef = useRef<HTMLDivElement | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  // Tick clock once per second.
  useEffect(() => {
    const id = window.setInterval(() => setNow(new Date()), 1000);
    return () => window.clearInterval(id);
  }, []);

  // Close on outside click + Escape.
  useEffect(() => {
    if (!open) return;
    const onPointerDown = (e: PointerEvent) => {
      const t = e.target as Node;
      if (popoverRef.current?.contains(t)) return;
      if (triggerRef.current?.contains(t)) return;
      setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('pointerdown', onPointerDown);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('pointerdown', onPointerDown);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  // Reset view to current month whenever the popover opens.
  useEffect(() => {
    if (open) {
      setViewMonth(new Date().getMonth());
      setViewYear(new Date().getFullYear());
    }
  }, [open]);

  const fullDate = useMemo(() => formatLongDate(now), []);
  const timezone = useMemo(() => Intl.DateTimeFormat().resolvedOptions().timeZone, []);

  return (
    <div className="tw-wrapper">
      <Button
        ref={triggerRef}
        type="button"
        variant="ghost"
        size="md"
        className="dash-topbar-clock dash-topbar-clock-clickable"
        title={`Click to expand - ${fullDate}`}
        aria-haspopup="dialog"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
      >
        <span className="dash-topbar-clock-time">{formatClock(now)}</span>
        <span className="dash-topbar-clock-day">{formatDayLabel(now).split(',')[0]}</span>
      </Button>
      {open && (
        <div
          ref={popoverRef}
          className="tw-popover"
          role="dialog"
          aria-label="Time and calendar"
        >
          <div className="tw-header">
            <div className="tw-clock-big">{formatClockWithSeconds(now)}</div>
            <div className="tw-date-full">{fullDate}</div>
            <div className="tw-tz">Timezone: <strong>{timezone}</strong></div>
          </div>
          <CalendarView
            month={viewMonth}
            year={viewYear}
            today={now}
            holidays={INDIA_HOLIDAYS_2026}
            country={country}
            onPrev={() => {
              const d = new Date(viewYear, viewMonth - 1, 1);
              setViewMonth(d.getMonth());
              setViewYear(d.getFullYear());
            }}
            onNext={() => {
              const d = new Date(viewYear, viewMonth + 1, 1);
              setViewMonth(d.getMonth());
              setViewYear(d.getFullYear());
            }}
            onPickYear={(y) => setViewYear(y)}
          />
          <div className="tw-legend">
            <span className="tw-legend-item"><span className="tw-dot tw-dot-public" /> Public holiday</span>
            <span className="tw-legend-item"><span className="tw-dot tw-dot-observance" /> Observance</span>
            <span className="tw-legend-item"><span className="tw-dot tw-dot-today" /> Today</span>
          </div>
        </div>
      )}
    </div>
  );
}

// =============================================================
// Helpers
// =============================================================

function formatClockWithSeconds(d: Date): string {
  // HH:MM:SS AM/PM, 12-hour clock.
  let h = d.getHours();
  const m = d.getMinutes().toString().padStart(2, '0');
  const s = d.getSeconds().toString().padStart(2, '0');
  const period = h >= 12 ? 'PM' : 'AM';
  h = h % 12;
  if (h === 0) h = 12;
  return `${h.toString().padStart(2, '0')}:${m}:${s} ${period}`;
}

function formatLongDate(d: Date): string {
  // "Sunday, 23 August 2026"
  const weekday = d.toLocaleDateString('en-US', { weekday: 'long' });
  const day = d.getDate();
  const month = d.toLocaleDateString('en-US', { month: 'long' });
  const year = d.getFullYear();
  return `${weekday}, ${day} ${month} ${year}`;
}

const MONTH_NAMES = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
];

interface CalendarViewProps {
  month: number;
  year: number;
  today: Date;
  holidays: ReadonlyArray<{ date: string; name: string; kind: 'public' | 'observance' }>;
  country: 'IN';
  onPrev: () => void;
  onNext: () => void;
  onPickYear: (y: number) => void;
}

function CalendarView({ month, year, today, holidays, onPrev, onNext, onPickYear }: CalendarViewProps) {
  // Build a 6-row x 7-col grid starting from the first Sunday on or
  // before day 1 of the month. Each cell is either null (padding
  // days from previous month) or a day-of-month integer.
  const firstOfMonth = new Date(year, month, 1);
  const firstWeekday = firstOfMonth.getDay(); // 0 = Sun
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const cells: Array<{ day: number; isoDate: string } | null> = [];
  for (let i = 0; i < firstWeekday; i++) cells.push(null);
  for (let d = 1; d <= daysInMonth; d++) {
    const iso = `${year}-${String(month + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
    cells.push({ day: d, isoDate: iso });
  }
  while (cells.length % 7 !== 0) cells.push(null);

  const todayIso = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`;

  // Quick year-pick buttons: current year +/- 2.
  const years = [year - 2, year - 1, year, year + 1, year + 2];

  return (
    <div className="tw-cal">
      <div className="tw-cal-header">
        <button type="button" className="tw-cal-nav" onClick={onPrev} aria-label="Previous month">‹</button>
        <div className="tw-cal-title">{MONTH_NAMES[month]}</div>
        <button type="button" className="tw-cal-nav" onClick={onNext} aria-label="Next month">›</button>
      </div>
      <div className="tw-cal-years">
        {years.map((y) => (
          <button
            key={y}
            type="button"
            className={`tw-cal-year ${y === year ? 'is-active' : ''}`}
            onClick={() => onPickYear(y)}
          >
            {y}
          </button>
        ))}
      </div>
      <div className="tw-cal-grid">
        {['S', 'M', 'T', 'W', 'T', 'F', 'S'].map((d, i) => (
          <div key={i} className="tw-cal-dow">{d}</div>
        ))}
        {cells.map((cell, idx) => {
          if (!cell) return <div key={`pad-${idx}`} className="tw-cal-cell tw-cal-cell-pad" />;
          const holiday = holidays.find((h) => h.date === cell.isoDate);
          const isToday = cell.isoDate === todayIso;
          const className = [
            'tw-cal-cell',
            holiday ? `tw-cal-cell-${holiday.kind}` : '',
            isToday ? 'tw-cal-cell-today' : '',
          ].filter(Boolean).join(' ');
          return (
            <div key={cell.isoDate} className={className} title={holiday?.name ?? ''}>
              <span className="tw-cal-day">{cell.day}</span>
              {holiday && <span className={`tw-cal-marker tw-cal-marker-${holiday.kind}`} aria-hidden />}
            </div>
          );
        })}
      </div>
    </div>
  );
}
