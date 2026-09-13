import React from 'react';
import './ScheduleGrid.css';

/**
 * Renders a CP-SAT nurse roster as a color-coded grid.
 *
 * Props:
 *   schedule: string[][]   — one row per nurse, one column per day
 *   shiftNames: string[]   — expected shift labels (optional)
 */
function ScheduleGrid({ schedule, shiftNames = ['morning', 'evening', 'night', 'off'] }) {
  if (!Array.isArray(schedule) || schedule.length === 0) {
    return <div className="schedule-empty">No schedule available</div>;
  }

  const numNurses = schedule.length;
  const numDays = schedule[0].length;

  return (
    <div className="schedule-wrapper">
      <div className="schedule-scroll">
        <table className="schedule-grid">
          <thead>
            <tr>
              <th className="schedule-corner">Nurse</th>
              {Array.from({ length: numDays }).map((_, d) => (
                <th key={d} className="schedule-day-header">
                  Day {d + 1}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {schedule.map((row, n) => (
              <tr key={n}>
                <td className="schedule-nurse-label">N{n + 1}</td>
                {row.map((shift, d) => (
                  <td
                    key={d}
                    className={`schedule-cell shift-${shift}`}
                    title={`Nurse ${n + 1}, Day ${d + 1}: ${shift}`}
                  >
                    <span className="shift-abbr">{abbrev(shift)}</span>
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="schedule-legend">
        <span className="legend-item">
          <span className="legend-swatch shift-morning" /> Morning
        </span>
        <span className="legend-item">
          <span className="legend-swatch shift-evening" /> Evening
        </span>
        <span className="legend-item">
          <span className="legend-swatch shift-night" /> Night
        </span>
        <span className="legend-item">
          <span className="legend-swatch shift-off" /> Off
        </span>
      </div>
    </div>
  );
}

function abbrev(shift) {
  switch (shift) {
    case 'morning': return 'M';
    case 'evening': return 'E';
    case 'night':   return 'N';
    case 'off':     return '—';
    default:        return '?';
  }
}

export default ScheduleGrid;