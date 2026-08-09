"use client";

import { Sparkle } from "@phosphor-icons/react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { ApiClientError, documentsApi } from "@/lib/api";
import { useToast } from "@/components/toast-provider";

export function SampleDocumentButton() {
  const router = useRouter();
  const { showToast } = useToast();
  const [creating, setCreating] = useState(false);

  async function createSample() {
    setCreating(true);
    try {
      const document = await documentsApi.create({
        title: "CrossVal pricing sample", customer: "CrossVal", issueDate: new Date().toISOString().slice(0, 10), currency: "USD",
        lineItems: [
          { description: "Widget A", quantity: 2, unitPrice: "100.00", discount: { type: "percentage", value: "10" }, taxRate: "5" },
          { description: "Widget B", quantity: 1, unitPrice: "50.00", taxRate: "5" },
          { description: "Service fee", quantity: 1, unitPrice: "200.00", discount: { type: "fixed", value: "20.00" }, taxRate: "0" },
        ],
      });
      showToast({ tone: "success", title: "CrossVal sample created", message: "Server-calculated grand total: USD 421.50" });
      router.push(`/documents/${document.id}`);
    } catch (cause) {
      showToast({ tone: "error", title: "Sample couldn’t be created", message: cause instanceof ApiClientError ? cause.message : "We could not reach Daftar." });
    } finally { setCreating(false); }
  }

  return <button className="sample-document-button" type="button" onClick={() => void createSample()} disabled={creating}><Sparkle size={18} weight="fill" /> {creating ? "Building sample…" : "Create the 421.50 sample"}</button>;
}
