import Link from "next/link";
import { FileText, Plus } from "@phosphor-icons/react";

export function DocumentsLoading() {
  return <div className="table-state" role="status" aria-live="polite"><span className="session-spinner" aria-hidden="true" /><p>Balancing your ledger…</p></div>;
}
export function DocumentsEmpty({ filtered }: { filtered: boolean }) {
  return <div className="table-state table-state--empty"><span className="empty-ledger" aria-hidden="true"><FileText size={25} weight="duotone" /></span><h2>{filtered ? "No matching documents" : "Your ledger is ready"}</h2><p>{filtered ? "Adjust or clear the filters to see more records." : "Create your first document and Daftar will keep every total clear."}</p>{!filtered && <Link className="primary-link" href="/documents/new"><Plus size={17} weight="bold" /> Create document</Link>}</div>;
}
export function DocumentsError({ retry }: { retry: () => void }) {
  return <div className="table-state table-state--empty" role="alert"><h2>We couldn’t open the ledger</h2><p>Check the connection and try once more.</p><button className="secondary-button" type="button" onClick={retry}>Try again</button></div>;
}
