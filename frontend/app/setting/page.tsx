"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import {
  BACKEND_BASE_URL,
  clearSessionTokenCookie,
  getSessionTokenFromCookieString,
  type GoogleAuthSessionResponse,
} from "@/lib/auth";

function getInitial(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return "?";
  }
  return trimmed[0].toUpperCase();
}

export default function SettingPage() {
  const router = useRouter();
  const [sessionToken, setSessionToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const [email, setEmail] = useState("");
  const [nameInput, setNameInput] = useState("");
  const [avatarInput, setAvatarInput] = useState("");

  useEffect(() => {
    let cancelled = false;

    const bootstrap = async () => {
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
        if (cancelled) {
          return;
        }

        setSessionToken(token);
        setEmail(payload.email ?? "");
        setNameInput(payload.name ?? "");
        setAvatarInput(payload.picture ?? "");
        setIsLoading(false);
      } catch {
        clearSessionTokenCookie();
        if (!cancelled) {
          router.replace("/login");
        }
      }
    };

    void bootstrap();

    return () => {
      cancelled = true;
    };
  }, [router]);

  const avatarPreview = useMemo(() => avatarInput.trim(), [avatarInput]);

  const handleSave = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!sessionToken) {
      return;
    }

    const nextName = nameInput.trim();
    const nextPicture = avatarInput.trim();
    if (!nextName) {
      setErrorMessage("Name cannot be empty.");
      setSuccessMessage(null);
      return;
    }

    setIsSaving(true);
    setErrorMessage(null);
    setSuccessMessage(null);

    try {
      const response = await fetch(`${BACKEND_BASE_URL}/auth/me`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          "X-Session-Token": sessionToken,
        },
        body: JSON.stringify({
          name: nextName,
          picture: nextPicture,
        }),
      });

      if (response.status === 401) {
        clearSessionTokenCookie();
        router.replace("/login");
        return;
      }

      if (!response.ok) {
        let message = "Failed to update profile.";
        try {
          const payload = (await response.json()) as { error?: string };
          if (payload.error) {
            message = payload.error;
          }
        } catch {
          // noop
        }
        throw new Error(message);
      }

      const payload = (await response.json()) as GoogleAuthSessionResponse;
      setNameInput(payload.name ?? nextName);
      setAvatarInput(payload.picture ?? nextPicture);
      setSuccessMessage("Profile updated.");
    } catch (error) {
      setErrorMessage(
        error instanceof Error ? error.message : "Failed to update profile.",
      );
    } finally {
      setIsSaving(false);
    }
  };

  const handleBackToDashboard = () => {
    router.push("/dashboard");
  };

  return (
    <div className="min-h-screen bg-zinc-100 px-4 py-10 dark:bg-zinc-950">
      <main className="mx-auto max-w-2xl space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
            Settings
          </h1>
          <Button variant="outline" onClick={handleBackToDashboard}>
            Back to Dashboard
          </Button>
        </div>

        <section className="rounded-xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-zinc-900">
          {isLoading ? (
            <p className="text-zinc-600 dark:text-zinc-300">Loading profile...</p>
          ) : (
            <form className="space-y-5" onSubmit={handleSave}>
              <div className="flex items-center gap-4">
                {avatarPreview ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={avatarPreview}
                    alt="Avatar preview"
                    className="h-16 w-16 rounded-full border border-black/10 object-cover dark:border-white/10"
                  />
                ) : (
                  <div className="flex h-16 w-16 items-center justify-center rounded-full border border-black/10 bg-zinc-200 text-lg font-semibold text-zinc-700 dark:border-white/10 dark:bg-zinc-800 dark:text-zinc-100">
                    {getInitial(nameInput)}
                  </div>
                )}
                <div>
                  <p className="text-sm text-zinc-500 dark:text-zinc-400">
                    Signed in as
                  </p>
                  <p className="text-sm font-medium text-zinc-900 dark:text-zinc-100">
                    {email || "unknown user"}
                  </p>
                </div>
              </div>

              <label className="block space-y-1">
                <span className="text-sm font-medium text-zinc-800 dark:text-zinc-200">
                  Name
                </span>
                <input
                  type="text"
                  value={nameInput}
                  onChange={(event) => setNameInput(event.target.value)}
                  className="w-full rounded-md border border-zinc-300 bg-white px-3 py-2 text-sm text-zinc-900 outline-none transition focus:border-zinc-500 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
                  placeholder="Your display name"
                />
              </label>

              <label className="block space-y-1">
                <span className="text-sm font-medium text-zinc-800 dark:text-zinc-200">
                  Avatar URL
                </span>
                <input
                  type="url"
                  value={avatarInput}
                  onChange={(event) => setAvatarInput(event.target.value)}
                  className="w-full rounded-md border border-zinc-300 bg-white px-3 py-2 text-sm text-zinc-900 outline-none transition focus:border-zinc-500 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
                  placeholder="https://example.com/avatar.png"
                />
              </label>

              {errorMessage ? (
                <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                  {errorMessage}
                </p>
              ) : null}

              {successMessage ? (
                <p className="rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-700">
                  {successMessage}
                </p>
              ) : null}

              <div className="flex justify-end">
                <Button type="submit" disabled={isSaving}>
                  {isSaving ? "Saving..." : "Save Changes"}
                </Button>
              </div>
            </form>
          )}
        </section>
      </main>
    </div>
  );
}
