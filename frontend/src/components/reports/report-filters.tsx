"use client";

import { CalendarBlank, ArrowRight } from "@phosphor-icons/react";

export function ReportFilters({ from, to, loading, error, onFromChange, onToChange, onApply }: {
  from: string; to: string; loading: boolean; error?: string;
  onFromChange: (value: string) => void; onToChange: (value: string) => void; onApply: () => void;
}) {
  return <section className="report-filter-card"><div className="report-filter-intro"><span><CalendarBlank size={21} weight="duotone" /></span><div><h2>Reporting period</h2><p>Issue dates are inclusive at both boundaries.</p></div></div><div className="report-date-fields"><div className="editor-field"><label htmlFor="report-from">From</label><input id="report-from" type="date" value={from} max={to || undefined} onChange={(event) => onFromChange(event.target.value)} /></div><ArrowRight className="date-arrow" size={17} /><div className="editor-field"><label htmlFor="report-to">To</label><input id="report-to" type="date" value={to} min={from || undefined} onChange={(event) => onToChange(event.target.value)} /></div><button type="button" onClick={onApply} disabled={loading}>{loading ? "Loading…" : "Apply range"}</button></div>{error && <p className="report-filter-error" role="alert">{error}</p>}</section>;
}
