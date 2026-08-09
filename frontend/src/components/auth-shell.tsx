import Image from "next/image";
import { ReactNode } from "react";

export function AuthShell({ children }: { children: ReactNode }) {
  return (
    <main className="auth-shell">
      <section className="brand-panel" aria-labelledby="brand-title">
        <div className="brand-panel__inner">
          <Image className="logo" src="/daftar-primary.svg" width={1120} height={390} alt="Daftar" priority />
          <div className="brand-copy">
            <p className="eyebrow">Modern ledger</p>
            <h1 id="brand-title">Your business,<br />clearly recorded.</h1>
            <p>Build accurate documents across discounts, currencies, and tax rates—without losing the thread.</p>
          </div>
          <div className="ledger-card" aria-hidden="true">
            <div className="ledger-card__top"><span>INV-0042</span><span className="status">Finalized</span></div>
            <div className="ledger-line"><span>Design consultation</span><strong>$2,400.00</strong></div>
            <div className="ledger-line"><span>Implementation</span><strong>$4,850.00</strong></div>
            <div className="ledger-total"><span>Grand total</span><strong>$7,794.50</strong></div>
          </div>
          <p className="brand-note">Calm records. Exact totals. Grounded decisions.</p>
        </div>
      </section>
      <section className="form-panel"><div className="form-wrap">
        <div className="mobile-logo"><Image src="/daftar-primary.svg" width={1120} height={390} alt="Daftar" priority /></div>
        {children}
      </div></section>
    </main>
  );
}
