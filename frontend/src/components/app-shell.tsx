"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ReactNode } from "react";
import { ChartBar, FileText, SignOut, SquaresFour } from "@phosphor-icons/react";
import { useSession } from "@/components/session-provider";

const navigation = [
  { href: "/dashboard", label: "Overview", icon: SquaresFour },
  { href: "/documents", label: "Documents", icon: FileText },
  { href: "/reports", label: "Reports", icon: ChartBar },
];

export function AppShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const { user, logout } = useSession();
  const initials = `${user?.first_name?.[0] ?? ""}${user?.last_name?.[0] ?? ""}`.toUpperCase();

  return <div className="app-shell"><a className="skip-link" href="#main-content">Skip to main content</a>
    <aside className="app-sidebar">
      <Link className="sidebar-brand" href="/dashboard" aria-label="Daftar overview"><Image src="/daftar-primary.svg" width={224} height={78} alt="Daftar" priority /></Link>
      <nav className="sidebar-nav" aria-label="Main navigation">{navigation.map(({ href, label, icon: Icon }) => {
        const active = pathname.startsWith(href);
        return <Link className={active ? "sidebar-link sidebar-link--active" : "sidebar-link"} href={href} aria-current={active ? "page" : undefined} key={href}><Icon aria-hidden="true" size={20} weight={active ? "fill" : "regular"} /><span>{label}</span></Link>;
      })}</nav>
      <div className="sidebar-account"><span className="sidebar-avatar" aria-hidden="true">{initials}</span><div><strong>{user?.first_name} {user?.last_name}</strong><span>{user?.email}</span></div><button type="button" onClick={() => void logout()} aria-label="Sign out"><SignOut size={19} /></button></div>
    </aside>
    <div className="app-content">
      <header className="mobile-bar"><Link href="/dashboard"><Image src="/daftar-primary.svg" width={224} height={78} alt="Daftar" /></Link><div className="mobile-account"><span className="user-avatar" aria-hidden="true">{initials}</span><button type="button" onClick={() => void logout()} aria-label="Sign out"><SignOut size={19} /></button></div></header>
      {children}
      <nav className="mobile-nav" aria-label="Mobile navigation">{navigation.map(({ href, label, icon: Icon }) => { const active = pathname.startsWith(href); return <Link className={active ? "mobile-nav__active" : ""} href={href} aria-current={active ? "page" : undefined} key={href}><Icon aria-hidden="true" size={20} weight={active ? "fill" : "regular"} /><span>{label}</span></Link>; })}</nav>
    </div>
  </div>;
}
