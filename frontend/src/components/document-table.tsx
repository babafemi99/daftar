import Link from "next/link";
import { ArrowRight } from "@phosphor-icons/react";
import { Document } from "@/lib/api";
import { MoneyAmount } from "@/components/money-amount";
import { StatusBadge } from "@/components/status-badge";

export function DocumentTable({ documents }: { documents: Document[] }) {
  return <div className="document-table-wrap"><table className="document-table">
    <caption className="sr-only">Owned financial documents</caption><thead><tr><th scope="col">Document</th><th scope="col">Customer</th><th scope="col">Issue date</th><th scope="col">Status</th><th scope="col" className="align-right">Total</th><th scope="col"><span className="sr-only">Open</span></th></tr></thead>
    <tbody>{documents.map((document) => <tr key={document.id}>
      <td><Link className="document-title" href={`/documents/${document.id}`}>{document.title || "Untitled document"}</Link><span className="document-reference">{document.reference}</span></td>
      <td>{document.customer || "—"}</td>
      <td><time dateTime={document.issueDate}>{formatDate(document.issueDate)}</time></td>
      <td><StatusBadge document={document} /></td>
      <td className="align-right"><MoneyAmount amount={document.totals.grandTotal} currency={document.currency} /></td>
      <td><Link className="row-action" href={`/documents/${document.id}`} aria-label={`Open ${document.title}`}><ArrowRight size={16} weight="bold" /></Link></td>
    </tr>)}</tbody>
  </table></div>;
}

function formatDate(value: string) {
  const [year, month, day] = value.slice(0, 10).split("-");
  return new Intl.DateTimeFormat("en", { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }).format(new Date(Date.UTC(Number(year), Number(month) - 1, Number(day))));
}
