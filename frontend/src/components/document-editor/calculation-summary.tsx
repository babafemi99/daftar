import { CalculationPreview, Currency } from "@/lib/api";
import { MoneyAmount } from "@/components/money-amount";

export function CalculationSummary({ preview, currency, calculating }: { preview: CalculationPreview | null; currency: Currency; calculating: boolean }) {
  const totals = preview?.totals;
  return <aside className="calculation-card" aria-label="Server-calculated document summary" aria-live="polite" aria-busy={calculating}><div className="calculation-card__heading"><div><p className="eyebrow">Live calculation</p><h2>Summary</h2></div>{calculating && <span className="session-spinner" role="status" aria-label="Calculating" />}</div>
    <dl className="totals-list"><div><dt>Subtotal</dt><dd>{totals ? <MoneyAmount amount={totals.subtotal} currency={currency} /> : "—"}</dd></div><div><dt>Discount</dt><dd className="discount-total">{totals ? <>− <MoneyAmount amount={totals.discount} currency={currency} /></> : "—"}</dd></div><div><dt>Tax</dt><dd>{totals ? <MoneyAmount amount={totals.tax} currency={currency} /> : "—"}</dd></div><div className="grand-total"><dt>Grand total</dt><dd>{totals ? <MoneyAmount amount={totals.grandTotal} currency={currency} /> : "—"}</dd></div></dl>
    <div className="tax-summary"><h3>Tax breakdown</h3>{preview?.taxBreakdown.length ? preview.taxBreakdown.map((tax) => <div key={tax.rate}><span>{tax.rate}% on {currency} {tax.taxableAmount}</span><strong>{currency} {tax.taxAmount}</strong></div>) : <p>Add valid lines to see tax groups.</p>}</div>
    <p className="calculation-note">Totals are calculated by Daftar’s server policy. Client values are never trusted.</p>
  </aside>;
}
