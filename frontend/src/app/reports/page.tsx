"use client";

import { useCallback, useEffect, useState } from "react";
import { ChartBar, LockSimple } from "@phosphor-icons/react";
import { AppShell } from "@/components/app-shell";
import { Protected } from "@/components/session-provider";
import { CurrencySummaryCard } from "@/components/reports/currency-summary-card";
import { ReportFilters } from "@/components/reports/report-filters";
import { ReportEmpty, ReportError, ReportLoading } from "@/components/reports/report-states";
import { reportsApi, SummaryReport } from "@/lib/api";

const today = new Date().toISOString().slice(0, 10);
const monthStart = `${today.slice(0, 7)}-01`;

export default function ReportsPage() {
  const [from, setFrom] = useState(monthStart);
  const [to, setTo] = useState(today);
  const [range, setRange] = useState({ from: monthStart, to: today });
  const [report, setReport] = useState<SummaryReport | null>(null);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [validation, setValidation] = useState("");
  const [revision, setRevision] = useState(0);

  const apply = useCallback(() => {
    if (!from || !to) { setValidation("Choose both a from date and a to date."); return; }
    if (from > to) { setValidation("From cannot be after to."); return; }
    setValidation(""); setState("loading"); setRange({ from, to });
  }, [from, to]);

  useEffect(() => {
    const controller = new AbortController();
    reportsApi.summary(range.from, range.to, controller.signal).then((result) => { setReport(result); setState("ready"); }).catch((cause) => {
      if (cause instanceof DOMException && cause.name === "AbortError") return;
      setState("error");
    });
    return () => controller.abort();
  }, [range, revision]);

  return <Protected><AppShell><main id="main-content" className="reports-page" tabIndex={-1}><div className="page-heading report-heading"><div><p className="eyebrow">Reporting</p><h1>Summary</h1><p>Finalized document totals, separated clearly by currency.</p></div><span className="finalized-note"><LockSimple size={15} /> Finalized records only</span></div>
    <ReportFilters from={from} to={to} loading={state === "loading"} error={validation} onFromChange={setFrom} onToChange={setTo} onApply={apply} />
    {state === "loading" && <ReportLoading />}
    {state === "error" && <ReportError onRetry={() => { setState("loading"); setRevision((value) => value + 1); }} />}
    {state === "ready" && report && <><div className="report-overview"><div><span><ChartBar size={20} weight="duotone" /></span><div><p>Finalized documents</p><strong>{report.documentCount}</strong></div></div><p>{formatRange(report.from, report.to)}</p></div>{report.currencies.length ? <div className="currency-report-grid">{report.currencies.map((summary) => <CurrencySummaryCard summary={summary} key={summary.currency} />)}</div> : <ReportEmpty />}</>}
  </main></AppShell></Protected>;
}

function formatRange(from: string, to: string) { const formatter = new Intl.DateTimeFormat("en", { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }); return `${formatter.format(new Date(`${from}T00:00:00Z`))} — ${formatter.format(new Date(`${to}T00:00:00Z`))}`; }
