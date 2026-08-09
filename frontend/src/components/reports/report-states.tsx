import { ChartBar, FileMagnifyingGlass } from "@phosphor-icons/react";

export function ReportLoading() { return <div className="report-state" role="status" aria-live="polite"><span className="session-spinner" aria-hidden="true" /><p>Preparing the ledger summary…</p></div>; }
export function ReportEmpty() { return <div className="report-state"><span className="empty-ledger" aria-hidden="true"><FileMagnifyingGlass size={25} weight="duotone" /></span><h2>No finalized documents</h2><p>There are no finalized documents with issue dates inside this reporting period.</p></div>; }
export function ReportError({ onRetry }: { onRetry: () => void }) { return <div className="report-state" role="alert"><span className="empty-ledger" aria-hidden="true"><ChartBar size={25} weight="duotone" /></span><h2>We couldn’t prepare the report</h2><p>Check the connection and try the reporting period again.</p><button type="button" onClick={onRetry}>Try again</button></div>; }
