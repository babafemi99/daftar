"use client";

import { Document } from "@/lib/api";
import { Archive, CheckCircle, Copy, ArrowCounterClockwise, FloppyDisk, Printer } from "@phosphor-icons/react";

export function DocumentActions({ document, busy, onFinalize, onArchive, onRestore, onDuplicate }: {
  document: Document; busy: boolean; onFinalize: () => void; onArchive: () => void; onRestore: () => void; onDuplicate: () => void;
}) {
  if (document.archivedAt) return <div className="document-actions"><button type="button" onClick={onRestore} disabled={busy}><ArrowCounterClockwise size={17} weight="bold" /> Restore draft</button></div>;
  if (document.status === "finalized") return <div className="document-actions"><button type="button" className="secondary-action" onClick={() => window.print()} disabled={busy}><Printer size={17} /> Print / Save PDF</button><button type="button" className="secondary-action" onClick={onDuplicate} disabled={busy}><Copy size={17} /> Duplicate as draft</button></div>;
  return <div className="document-actions"><button type="submit" form="edit-document-form" className="secondary-action" disabled={busy}><FloppyDisk size={17} /> Save draft</button><button type="button" className="secondary-action danger-action" onClick={onArchive} disabled={busy}><Archive size={17} /> Archive</button><button type="button" onClick={onFinalize} disabled={busy}><CheckCircle size={17} weight="bold" /> Finalize</button></div>;
}
