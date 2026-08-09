"use client";

import Link from "next/link";
import { ArrowLeft } from "@phosphor-icons/react";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { ConfirmationDialog } from "@/components/confirmation-dialog";
import { DocumentActions } from "@/components/document-actions";
import { DocumentAuditTimeline } from "@/components/document-audit-timeline";
import { DocumentEditor } from "@/components/document-editor/document-editor";
import { DocumentsError, DocumentsLoading } from "@/components/document-states";
import { DocumentView } from "@/components/document-view";
import { Protected } from "@/components/session-provider";
import { StatusBadge } from "@/components/status-badge";
import { useToast } from "@/components/toast-provider";
import { ApiClientError, Document, documentsApi } from "@/lib/api";

type Confirmation = "finalize" | "archive" | null;

export default function DocumentPage() {
  const { documentId } = useParams<{ documentId: string }>();
  const router = useRouter();
  const { showToast } = useToast();
  const [document, setDocument] = useState<Document | null>(null);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [revision, setRevision] = useState(0);
  const [busy, setBusy] = useState(false);
  const [confirmation, setConfirmation] = useState<Confirmation>(null);

  useEffect(() => {
    const controller = new AbortController();
    documentsApi.get(documentId, controller.signal).then((result) => { setDocument(result); setState("ready"); }).catch((cause) => {
      if (cause instanceof DOMException && cause.name === "AbortError") return;
      setState("error");
    });
    return () => controller.abort();
  }, [documentId, revision]);

  async function mutate(action: "finalize" | "archive" | "restore" | "duplicate") {
    if (!document) return;
    setBusy(true);
    try {
      const result = action === "finalize" ? await documentsApi.finalize(document.id, document.version)
        : action === "archive" ? await documentsApi.archive(document.id, document.version)
        : action === "restore" ? await documentsApi.restore(document.id, document.version)
        : await documentsApi.duplicate(document.id);
      showToast({ tone: "success", title: action === "finalize" ? "Document finalized" : action === "archive" ? "Draft archived" : action === "restore" ? "Draft restored" : "Draft duplicated", message: result.reference });
      if (action === "duplicate") router.push(`/documents/${result.id}`); else setDocument(result);
    } catch (cause) {
      const error = cause instanceof ApiClientError ? cause : null;
      showToast({ tone: "error", title: "Action couldn’t be completed", message: error?.code === "DOCUMENT_VERSION_CONFLICT" ? "This document changed elsewhere. Refresh and try again." : error?.message ?? "We could not reach Daftar." });
    } finally { setBusy(false); setConfirmation(null); }
  }

  return <Protected><AppShell><main id="main-content" className="editor-page" tabIndex={-1}>
    {state === "loading" && <DocumentsLoading />}
    {state === "error" && <DocumentsError retry={() => { setState("loading"); setRevision((value) => value + 1); }} />}
    {state === "ready" && document && <>
      <div className="detail-page-heading"><div><Link href="/documents"><ArrowLeft size={15} weight="bold" /> Documents</Link><div className="detail-title-line"><div><p className="eyebrow">{document.reference}</p><h1>{document.title}</h1></div><StatusBadge document={document} /></div></div><DocumentActions document={document} busy={busy} onFinalize={() => setConfirmation("finalize")} onArchive={() => setConfirmation("archive")} onRestore={() => void mutate("restore")} onDuplicate={() => void mutate("duplicate")} /></div>
      {document.status === "draft" && !document.archivedAt ? <DocumentEditor key={document.version} formId="edit-document-form" document={document} onSaved={setDocument} /> : <DocumentView document={document} />}
      <DocumentAuditTimeline documentId={document.id} version={document.version} />
    </>}
  </main></AppShell>
    <ConfirmationDialog open={confirmation === "finalize"} title="Finalize this document?" message="Finalization is permanent. The document will become read-only and its current calculated totals will be locked." confirmLabel="Finalize document" busy={busy} onClose={() => setConfirmation(null)} onConfirm={() => void mutate("finalize")} />
    <ConfirmationDialog open={confirmation === "archive"} title="Archive this draft?" message="The draft will leave your active ledger, but you can restore it later." confirmLabel="Archive draft" tone="danger" busy={busy} onClose={() => setConfirmation(null)} onConfirm={() => void mutate("archive")} />
  </Protected>;
}
