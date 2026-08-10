"use client";

import { createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { ApiClientError, authApi, User } from "@/lib/api";
import { useToast } from "@/components/toast-provider";
import { BrandLoadingScreen } from "@/components/brand-loading-screen";

type SessionStatus = "loading" | "authenticated" | "unauthenticated";
type SessionContextValue = {
  user: User | null;
  status: SessionStatus;
  establishSession: (user: User) => void;
  logout: () => Promise<void>;
};

const SessionContext = createContext<SessionContextValue | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const { showToast } = useToast();
  const [user, setUser] = useState<User | null>(null);
  const [status, setStatus] = useState<SessionStatus>("loading");
  const statusRef = useRef<SessionStatus>("loading");
  const deliberateLogout = useRef(false);

  const clearSession = useCallback((notify: boolean) => {
    statusRef.current = "unauthenticated";
    setUser(null);
    setStatus("unauthenticated");
    if (notify) showToast({ tone: "info", title: "Session expired", message: "Sign in again to continue." });
    if (pathname !== "/" && pathname !== "/register") router.replace("/");
  }, [pathname, router, showToast]);

  useEffect(() => {
    let active = true;
    authApi.me()
      .then((currentUser) => { if (active) { statusRef.current = "authenticated"; setUser(currentUser); setStatus("authenticated"); } })
      .catch((cause) => {
        if (!active) return;
        if (cause instanceof ApiClientError && cause.status === 401) clearSession(false);
        else { statusRef.current = "unauthenticated"; setStatus("unauthenticated"); showToast({ tone: "error", title: "Session check failed", message: "We could not verify your session." }); }
      });
    return () => { active = false; };
  }, [clearSession, showToast]);

  useEffect(() => {
    const expired = () => {
      if (deliberateLogout.current || statusRef.current !== "authenticated") return;
      clearSession(true);
    };
    window.addEventListener("daftar:session-expired", expired);
    return () => window.removeEventListener("daftar:session-expired", expired);
  }, [clearSession]);

  const establishSession = useCallback((currentUser: User) => {
    statusRef.current = "authenticated";
    setUser(currentUser);
    setStatus("authenticated");
  }, []);

  const logout = useCallback(async () => {
    deliberateLogout.current = true;
    statusRef.current = "unauthenticated";
    setUser(null);
    setStatus("unauthenticated");
    try { await authApi.logout(); }
    finally {
      router.replace("/");
      showToast({ tone: "success", title: "Signed out", message: "Your session has ended safely." });
      deliberateLogout.current = false;
    }
  }, [router, showToast]);

  const value = useMemo(() => ({ user, status, establishSession, logout }), [user, status, establishSession, logout]);
  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession() {
  const context = useContext(SessionContext);
  if (!context) throw new Error("useSession must be used within SessionProvider");
  return context;
}

export function GuestOnly({ children }: { children: ReactNode }) {
  const { status } = useSession();
  const router = useRouter();
  useEffect(() => { if (status === "authenticated") router.replace("/dashboard"); }, [router, status]);
  if (status !== "unauthenticated") return <SessionLoading />;
  return children;
}

export function Protected({ children }: { children: ReactNode }) {
  const { status } = useSession();
  const router = useRouter();
  useEffect(() => { if (status === "unauthenticated") router.replace("/"); }, [router, status]);
  if (status !== "authenticated") return <SessionLoading />;
  return children;
}

function SessionLoading() {
  return <BrandLoadingScreen />;
}
