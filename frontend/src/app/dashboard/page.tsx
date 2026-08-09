"use client";

import Link from "next/link";
import { ArrowRight, CheckCircle, FileText, Plus, Receipt, TrendUp } from "@phosphor-icons/react";
import { useEffect, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { CurrencyPulse } from "@/components/dashboard/currency-pulse";
import { SampleDocumentButton } from "@/components/dashboard/sample-document-button";
import { StatCard } from "@/components/dashboard/stat-card";
import { DocumentTable } from "@/components/document-table";
import { DocumentsLoading } from "@/components/document-states";
import { Protected, useSession } from "@/components/session-provider";
import { Document, documentsApi, SummaryReport, reportsApi } from "@/lib/api";

type DashboardData = { drafts: number; finalized: number; recent: Document[]; report: SummaryReport };

export default function DashboardPage() {
  const { user } = useSession();
  const [data, setData] = useState<DashboardData | null>(null);
  const [failed, setFailed] = useState(false);
  const year = new Date().getUTCFullYear();

  useEffect(() => {
    const controller = new AbortController();
    Promise.all([
      documentsApi.list({ status: "draft" }, 1, 1, controller.signal),
      documentsApi.list({ status: "finalized" }, 1, 1, controller.signal),
      documentsApi.list({}, 1, 5, controller.signal),
      reportsApi.summary(`${year}-01-01`, `${year}-12-31`, controller.signal),
    ]).then(([drafts, finalized, recent, report]) => setData({ drafts: drafts.page.totalItems, finalized: finalized.page.totalItems, recent: recent.items, report })).catch((cause) => {
      if (cause instanceof DOMException && cause.name === "AbortError") return;
      setFailed(true);
    });
    return () => controller.abort();
  }, [year]);

  return <Protected><AppShell><main id="main-content" className="dashboard-page" tabIndex={-1}>
    <header className="dashboard-hero"><div><p className="eyebrow">Financial overview</p><h1>Good day, {user?.first_name}.</h1><p>Your ledger is balanced, current, and ready for the next decision.</p></div><Link className="primary-link" href="/documents/new"><Plus size={17} weight="bold" /> New document</Link></header>
    {!data && !failed && <DocumentsLoading />}
    {failed && <section className="dashboard-error"><h2>We couldn’t prepare your overview</h2><p>Your documents are still safe. Refresh to try again.</p></section>}
    {data && <><section className="dashboard-stats"><StatCard label="All documents" value={data.drafts + data.finalized} detail="Active ledger records" icon={FileText} tone="ink" /><StatCard label="Drafts" value={data.drafts} detail="Ready for refinement" icon={Receipt} tone="copper" /><StatCard label="Finalized" value={data.finalized} detail="Immutable records" icon={CheckCircle} /><StatCard label={`${year} activity`} value={data.report.documentCount} detail="Included in reporting" icon={TrendUp} /></section>
      <section className="dashboard-grid"><div className="dashboard-panel"><header><div><p className="eyebrow">Currency pulse</p><h2>{year} finalized value</h2></div><Link href="/reports">Full report <ArrowRight size={14} /></Link></header>{data.report.currencies.length ? <div className="currency-pulse-list">{data.report.currencies.map((summary) => <CurrencyPulse summary={summary} key={summary.currency} />)}</div> : <div className="dashboard-panel-empty"><p>Finalize a document to begin your financial pulse.</p></div>}</div>
      <aside className="sample-card"><span><TrendUp size={23} weight="duotone" /></span><p className="eyebrow">Reviewer shortcut</p><h2>Prove the maths in one click.</h2><p>Create the exact mixed-discount, multi-tax assignment example. Daftar calculates the expected grand total entirely on the server.</p><div className="sample-total"><small>Expected grand total</small><strong>USD 421.50</strong></div><SampleDocumentButton /></aside></section>
      <section className="dashboard-recent"><header><div><p className="eyebrow">Recently issued</p><h2>Latest documents</h2></div><Link href="/documents">View ledger <ArrowRight size={14} /></Link></header>{data.recent.length ? <DocumentTable documents={data.recent} /> : <div className="dashboard-panel-empty"><p>Your first document will appear here.</p></div>}</section></>}
  </main></AppShell></Protected>;
}
