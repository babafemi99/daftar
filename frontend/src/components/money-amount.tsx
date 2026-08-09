import { Currency } from "@/lib/api";

export function MoneyAmount({ amount, currency }: { amount: string; currency: Currency }) {
  const [whole = "0", fraction = "00"] = amount.split(".");
  const grouped = whole.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  return <span className="money-amount"><span>{currency}</span> {grouped}.{fraction.padEnd(2, "0")}</span>;
}
