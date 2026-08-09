"use client";

import Link from "next/link";
import { Plus } from "@phosphor-icons/react";
import { useEffect, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { DocumentFilters } from "@/components/document-filters";
import { DocumentPagination } from "@/components/document-pagination";
import { DocumentsEmpty, DocumentsError, DocumentsLoading } from "@/components/document-states";
import { DocumentTable } from "@/components/document-table";
import { Protected } from "@/components/session-provider";
import { Document, DocumentListFilters, documentsApi, PageMeta } from "@/lib/api";

const pageSize = 10;

export default function DocumentsPage() {
  const [filters, setFilters] = useState<DocumentListFilters>({});
  const [documents, setDocuments] = useState<Document[]>([]);
  const [pageNumber, setPageNumber] = useState(1);
  const [page, setPage] = useState<PageMeta | null>(null);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    documentsApi.list(filters, pageNumber, pageSize, controller.signal).then((result) => { setDocuments(result.items); setPage(result.page); setState("ready"); }).catch((cause) => {
      if (cause instanceof DOMException && cause.name === "AbortError") return;
      setState("error");
    });
    return () => controller.abort();
  }, [filters, pageNumber, revision]);

  const changeFilters = (next: DocumentListFilters) => { setState("loading"); setPageNumber(1); setFilters(next); };
  const changePage = (next: number) => { setState("loading"); setPageNumber(next); window.scrollTo({ top: 0, behavior: "smooth" }); };

  return <Protected><AppShell><main id="main-content" className="documents-page" tabIndex={-1}>
    <div className="page-heading"><div><p className="eyebrow">Ledger</p><h1>Documents</h1><p>Every draft, total, and finalized record in one calm view.</p></div><Link className="primary-link" href="/documents/new"><Plus size={17} weight="bold" /> New document</Link></div>
    <DocumentFilters filters={filters} onChange={changeFilters} onClear={() => changeFilters({})} />
    <section className="documents-panel" aria-label="Owned documents">
      <div className="panel-heading"><h2>Your records</h2>{state === "ready" && page && <span>{page.totalItems} {page.totalItems === 1 ? "document" : "documents"}</span>}</div>
      {state === "loading" && <DocumentsLoading />}
      {state === "error" && <DocumentsError retry={() => { setState("loading"); setRevision((value) => value + 1); }} />}
      {state === "ready" && documents.length === 0 && <DocumentsEmpty filtered={Object.keys(filters).length > 0} />}
      {state === "ready" && documents.length > 0 && <DocumentTable documents={documents} />}
      {state === "ready" && page && <DocumentPagination page={page} onChange={changePage} />}
    </section>
  </main></AppShell></Protected>;
}
