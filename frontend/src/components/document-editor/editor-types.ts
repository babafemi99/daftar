import { Currency, DiscountInput } from "@/lib/api";

export type EditableLine = {
  key: string;
  id?: string;
  description: string;
  quantity: string;
  unitPrice: string;
  discountType: "none" | DiscountInput["type"];
  discountValue: string;
  taxRate: string;
};

export type EditorValue = {
  title: string;
  customer: string;
  issueDate: string;
  currency: Currency;
  lines: EditableLine[];
};

export function newLine(): EditableLine {
  return { key: crypto.randomUUID(), description: "", quantity: "1", unitPrice: "0.00", discountType: "none", discountValue: "", taxRate: "0" };
}
