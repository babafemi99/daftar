import { CurrencyReport } from "@/lib/api";
import { MoneyAmount } from "@/components/money-amount";
import { Coins, FileText } from "@phosphor-icons/react";

export function CurrencySummaryCard({ summary }: { summary: CurrencyReport }) {
  return <article className="currency-report-card"><header><div><span className="currency-icon"><Coins size={21} weight="duotone" /></span><div><p>{summary.currency}</p><h2><MoneyAmount amount={summary.grandTotal} currency={summary.currency} /></h2></div></div><span className="document-count"><FileText size={14} /> {summary.documentCount} {summary.documentCount === 1 ? "document" : "documents"}</span></header>
    <dl className="report-totals"><div><dt>Subtotal</dt><dd><MoneyAmount amount={summary.subtotal} currency={summary.currency} /></dd></div><div><dt>Discount</dt><dd className="discount-total">− <MoneyAmount amount={summary.totalDiscount} currency={summary.currency} /></dd></div><div><dt>Total tax</dt><dd><MoneyAmount amount={summary.totalTax} currency={summary.currency} /></dd></div></dl>
    <section className="report-tax"><h3>Tax breakdown</h3>{summary.taxBreakdown.length ? <div className="tax-table"><div className="tax-row tax-row--head"><span>Rate</span><span>Taxable amount</span><span>Tax amount</span></div>{summary.taxBreakdown.map((tax) => <div className="tax-row" key={tax.rate}><strong>{tax.rate}%</strong><span><MoneyAmount amount={tax.taxableAmount} currency={summary.currency} /></span><strong><MoneyAmount amount={tax.taxAmount} currency={summary.currency} /></strong></div>)}</div> : <p>No taxable amounts in this currency.</p>}</section>
  </article>;
}
