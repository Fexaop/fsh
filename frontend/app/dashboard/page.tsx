"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import {
  BACKEND_BASE_URL,
  clearSessionTokenCookie,
  getSessionTokenFromCookieString,
  type GoogleAuthSessionResponse,
} from "@/lib/auth";

export default function DashboardPage() {
  const [active, setActive] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [user, setUser] = useState<GoogleAuthSessionResponse | null>(null);
  const router = useRouter();

  useEffect(() => {
    let cancelled = false;

    const loadCurrentUser = async () => {
      const token = getSessionTokenFromCookieString(document.cookie);
      if (!token) {
        router.replace("/login");
        return;
      }

      try {
        const response = await fetch(`${BACKEND_BASE_URL}/auth/me`, {
          headers: {
            "X-Session-Token": token,
          },
        });

        if (!response.ok) {
          throw new Error("Session is invalid.");
        }

        const payload = (await response.json()) as GoogleAuthSessionResponse;
        if (!cancelled) {
          setUser(payload);
          setIsLoading(false);
        }
      } catch {
        clearSessionTokenCookie();
        if (!cancelled) {
          router.replace("/login");
        }
      }
    };

    void loadCurrentUser();

    return () => {
      cancelled = true;
    };
  }, [router]);

  const handleLogout = async () => {
    const token = getSessionTokenFromCookieString(document.cookie);

    try {
      if (token) {
        await fetch(`${BACKEND_BASE_URL}/auth/logout`, {
          method: "POST",
          headers: {
            "X-Session-Token": token,
          },
        });
      }
    } finally {
      clearSessionTokenCookie();
      router.replace("/login");
    }
  };

  return (
    <div className="min-h-screen bg-zinc-100 dark:bg-zinc-950">
      <header className="sticky top-0 z-50 border-b border-black/10 bg-white/65 backdrop-blur-md dark:border-white/10 dark:bg-black/45">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4">
          <span className="text-sm font-semibold tracking-wide text-zinc-900 dark:text-zinc-100">
            FSH Dashboard
          </span>
          <Button variant="outline" onClick={handleLogout}>
            Logout
          </Button>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-10">
        <div className="rounded-xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-zinc-900">
          {isLoading ? (
            <p className="text-zinc-600 dark:text-zinc-300">Loading profile...</p>
          ) : (
            <>
              <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
                Welcome to your dashboard
              </h1>
              <p className="mt-2 text-zinc-600 dark:text-zinc-300">
                Signed in as {user?.email ?? "unknown user"}
              </p>
              {user?.name ? (
                <p className="mt-1 text-zinc-600 dark:text-zinc-300">
                  Name: {user.name}
                </p>
              ) : null}
            </>
          )}
        </div>
      </main>
    </div>
  );
}
