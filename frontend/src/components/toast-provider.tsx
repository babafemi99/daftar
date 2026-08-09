"use client";

import { createContext, ReactNode, useCallback, useContext, useMemo } from "react";
import { toast, Toaster } from "sonner";

type ToastTone = "success" | "error" | "info";
type ToastInput = { title: string; message?: string; tone: ToastTone };
type ToastContextValue = { showToast: (notification: ToastInput) => void };

const ToastContext = createContext<ToastContextValue | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
  const showToast = useCallback(({ title, message, tone }: ToastInput) => {
    toast[tone](title, { description: message });
  }, []);
  const value = useMemo(() => ({ showToast }), [showToast]);

  return <ToastContext.Provider value={value}>
    {children}
    <Toaster
      position="top-right"
      closeButton
      duration={5000}
      gap={10}
      visibleToasts={4}
      toastOptions={{
        classNames: {
          toast: "daftar-toast",
          title: "daftar-toast__title",
          description: "daftar-toast__description",
          closeButton: "daftar-toast__close",
        },
      }}
    />
  </ToastContext.Provider>;
}

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) throw new Error("useToast must be used within ToastProvider");
  return context;
}
