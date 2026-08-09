import { CurrencyReport } from "@/lib/api";
import { MoneyAmount } from "@/components/money-amount";

export function CurrencyPulse({ summary }: { summary: CurrencyReport }) {
  const taxShare = Number(summary.grandTotal) > 0 ? Math.min(100, (Number(summary.totalTax) / Number(summary.grandTotal)) * 100) : 0;
  return <article className="currency-pulse"><header><div><span>{summary.currency}</span><p>{summary.documentCount} finalized</p></div><strong><MoneyAmount amount={summary.grandTotal} currency={summary.currency} /></strong></header><div className="currency-pulse__track"><i style={{ width: `${Math.max(4, taxShare)}%` }} /></div><footer><span>Tax <strong>{summary.currency} {summary.totalTax}</strong></span><span>Discount <strong>{summary.currency} {summary.totalDiscount}</strong></span></footer></article>;
}
