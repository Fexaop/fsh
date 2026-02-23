"use client";

import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import {
  BACKEND_BASE_URL,
  clearSessionTokenCookie,
  getSessionTokenFromCookieString,
  type GoogleAuthSessionResponse,
} from "@/lib/auth";

type FamilyInvitation = {
  id: number;
  inviteId: string;
  inviterName: string;
  inviterEmail: string;
  inviteeEmail?: string;
  relation: string;
  status: "pending" | "accepted" | "declined";
  canRespond: boolean;
  createdAt: number;
  expiresAt: number;
  respondedAt?: number;
};

type InvitationPayload = {
  invitation: FamilyInvitation;
};

type InvitationListPayload = {
  invitations: FamilyInvitation[];
};

function formatTimestamp(unixSeconds: number): string {
  return new Date(unixSeconds * 1000).toLocaleString();
}

function statusBadgeClass(status: FamilyInvitation["status"]): string {
  if (status === "accepted") {
    return "bg-emerald-100 text-emerald-700";
  }
  if (status === "declined") {
    return "bg-rose-100 text-rose-700";
  }
  return "bg-amber-100 text-amber-700";
}

export default function InvitationsPage() {
  const [sessionToken, setSessionToken] = useState<string | null>(null);
  const [user, setUser] = useState<GoogleAuthSessionResponse | null>(null);
  const [isLoadingUser, setIsLoadingUser] = useState(true);

  const [relationInput, setRelationInput] = useState("");
  const [createdInvite, setCreatedInvite] = useState<FamilyInvitation | null>(null);
  const [isCreatingInvite, setIsCreatingInvite] = useState(false);
  const [createInviteError, setCreateInviteError] = useState<string | null>(null);

  const [outgoingInvitations, setOutgoingInvitations] = useState<FamilyInvitation[]>([]);
  const [isLoadingOutgoing, setIsLoadingOutgoing] = useState(false);
  const [outgoingError, setOutgoingError] = useState<string | null>(null);

  const [lookupInviteId, setLookupInviteId] = useState("");
  const [lookupResult, setLookupResult] = useState<FamilyInvitation | null>(null);
  const [lookupError, setLookupError] = useState<string | null>(null);
  const [isLookingUp, setIsLookingUp] = useState(false);
  const [isResponding, setIsResponding] = useState(false);

  const router = useRouter();

  const withAuthHeaders = useCallback(
    (token: string) => ({
      "Content-Type": "application/json",
      "X-Session-Token": token,
    }),
    [],
  );

  const loadOutgoingInvitations = useCallback(
    async (token: string) => {
      setIsLoadingOutgoing(true);
      setOutgoingError(null);

      try {
        const response = await fetch(`${BACKEND_BASE_URL}/family/invitations/outgoing`, {
          headers: {
            "X-Session-Token": token,
          },
        });

        if (response.status === 401) {
          clearSessionTokenCookie();
          router.replace("/login");
          return;
        }

        if (!response.ok) {
          throw new Error("Failed to fetch outgoing invitations.");
        }

        const payload = (await response.json()) as InvitationListPayload;
        setOutgoingInvitations(payload.invitations ?? []);
      } catch (error) {
        setOutgoingError(
          error instanceof Error
            ? error.message
            : "Failed to fetch outgoing invitations.",
        );
      } finally {
        setIsLoadingOutgoing(false);
      }
    },
    [router],
  );

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
        if (!cancelled) {
          setUser(payload);
          setSessionToken(token);
          setIsLoadingUser(false);
        }
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

  useEffect(() => {
    if (!sessionToken) {
      return;
    }

    void loadOutgoingInvitations(sessionToken);
  }, [loadOutgoingInvitations, sessionToken]);

  const handleCreateInvite = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!sessionToken) {
      return;
    }

    setIsCreatingInvite(true);
    setCreateInviteError(null);

    try {
      const response = await fetch(`${BACKEND_BASE_URL}/family/invitations`, {
        method: "POST",
        headers: withAuthHeaders(sessionToken),
        body: JSON.stringify({
          relation: relationInput.trim(),
        }),
      });

      if (!response.ok) {
        let message = "Failed to create invite.";
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

      const payload = (await response.json()) as InvitationPayload;
      setCreatedInvite(payload.invitation);
      await loadOutgoingInvitations(sessionToken);
    } catch (error) {
      setCreateInviteError(
        error instanceof Error ? error.message : "Failed to create invite.",
      );
    } finally {
      setIsCreatingInvite(false);
    }
  };

  const handleLookupInvite = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!sessionToken) {
      return;
    }

    const inviteID = lookupInviteId.trim().toUpperCase();
    if (!inviteID) {
      setLookupError("Enter an invite ID.");
      return;
    }

    setIsLookingUp(true);
    setLookupError(null);

    try {
      const response = await fetch(`${BACKEND_BASE_URL}/family/invitations/${encodeURIComponent(inviteID)}`, {
        headers: {
          "X-Session-Token": sessionToken,
        },
      });

      if (!response.ok) {
        let message = "Invite not found.";
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

      const payload = (await response.json()) as InvitationPayload;
      setLookupResult(payload.invitation);
    } catch (error) {
      setLookupResult(null);
      setLookupError(error instanceof Error ? error.message : "Failed to lookup invite.");
    } finally {
      setIsLookingUp(false);
    }
  };

  const handleRespondToInvite = async (action: "accept" | "decline") => {
    if (!sessionToken || !lookupResult) {
      return;
    }

    setIsResponding(true);
    setLookupError(null);

    try {
      const response = await fetch(
        `${BACKEND_BASE_URL}/family/invitations/${encodeURIComponent(lookupResult.inviteId)}/respond`,
        {
          method: "POST",
          headers: withAuthHeaders(sessionToken),
          body: JSON.stringify({ action }),
        },
      );

      if (!response.ok) {
        let message = `Failed to ${action} invitation.`;
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

      const payload = (await response.json()) as InvitationPayload;
      setLookupResult(payload.invitation);
      await loadOutgoingInvitations(sessionToken);
    } catch (error) {
      setLookupError(
        error instanceof Error
          ? error.message
          : `Failed to ${action} invitation.`,
      );
    } finally {
      setIsResponding(false);
    }
  };

  const handleCopyInviteId = async () => {
    if (!createdInvite?.inviteId) {
      return;
    }
    try {
      await navigator.clipboard.writeText(createdInvite.inviteId);
    } catch {
      // Ignore clipboard failures.
    }
  };

  const sortedOutgoing = useMemo(
    () => [...outgoingInvitations].sort((a, b) => b.createdAt - a.createdAt),
    [outgoingInvitations],
  );

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
            FSH Invitations
          </span>
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={() => router.push("/dashboard")}>Back to Dashboard</Button>
            <Button variant="outline" onClick={handleLogout}>Logout</Button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl space-y-6 px-4 py-10">
        <div className="rounded-xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-zinc-900">
          {isLoadingUser ? (
            <p className="text-zinc-600 dark:text-zinc-300">Loading profile...</p>
          ) : (
            <>
              <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
                Invite Family Using an Invite ID
              </h1>
              <p className="mt-2 text-zinc-600 dark:text-zinc-300">
                Signed in as {user?.email ?? "unknown user"}
              </p>
            </>
          )}
        </div>

        <section className="grid gap-6 lg:grid-cols-2">
          <div className="space-y-4 rounded-xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-zinc-900">
            <h2 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100">
              Create Invite
            </h2>

            <form onSubmit={handleCreateInvite} className="space-y-3">
              <input
                type="text"
                value={relationInput}
                onChange={(event) => setRelationInput(event.target.value)}
                placeholder="Relation label (optional, e.g. Brother)"
                className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-500 dark:border-zinc-700 dark:bg-zinc-950"
              />
              <Button type="submit" disabled={isCreatingInvite}>
                {isCreatingInvite ? "Creating..." : "Create Invite ID"}
              </Button>
            </form>

            {createInviteError ? (
              <p className="text-sm text-red-600">{createInviteError}</p>
            ) : null}

            {createdInvite ? (
              <div className="rounded-lg border border-zinc-200 bg-zinc-50 p-4 dark:border-zinc-700 dark:bg-zinc-950">
                <p className="text-sm text-zinc-600 dark:text-zinc-300">Share this Invite ID:</p>
                <p className="mt-1 break-all font-mono text-2xl font-semibold tracking-widest text-zinc-900 dark:text-zinc-100">
                  {createdInvite.inviteId}
                </p>
                <div className="mt-3 flex flex-wrap items-center gap-2">
                  <Button variant="outline" onClick={handleCopyInviteId}>Copy ID</Button>
                  <span className="text-xs text-zinc-500 dark:text-zinc-400">
                    Expires {formatTimestamp(createdInvite.expiresAt)}
                  </span>
                </div>
              </div>
            ) : null}
          </div>

          <div className="space-y-4 rounded-xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-zinc-900">
            <h2 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100">
              Accept / Decline Invite
            </h2>

            <form onSubmit={handleLookupInvite} className="space-y-3">
              <input
                type="text"
                value={lookupInviteId}
                onChange={(event) => setLookupInviteId(event.target.value.toUpperCase())}
                placeholder="Enter Invite ID"
                className="w-full rounded-md border border-zinc-300 px-3 py-2 font-mono text-sm uppercase tracking-wide outline-none focus:border-zinc-500 dark:border-zinc-700 dark:bg-zinc-950"
              />
              <Button type="submit" disabled={isLookingUp}>
                {isLookingUp ? "Checking..." : "Lookup Invite"}
              </Button>
            </form>

            {lookupError ? <p className="text-sm text-red-600">{lookupError}</p> : null}

            {lookupResult ? (
              <div className="rounded-lg border border-zinc-200 p-4 dark:border-zinc-700">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-mono text-sm tracking-wide">{lookupResult.inviteId}</span>
                  <span className={`rounded-full px-2 py-1 text-xs font-medium ${statusBadgeClass(lookupResult.status)}`}>
                    {lookupResult.status}
                  </span>
                </div>
                <p className="mt-2 text-sm text-zinc-700 dark:text-zinc-300">
                  From: {lookupResult.inviterName || "Unknown"} ({lookupResult.inviterEmail})
                </p>
                <p className="text-sm text-zinc-700 dark:text-zinc-300">Relation: {lookupResult.relation}</p>
                <p className="text-xs text-zinc-500 dark:text-zinc-400">
                  Expires: {formatTimestamp(lookupResult.expiresAt)}
                </p>

                {lookupResult.canRespond && lookupResult.status === "pending" ? (
                  <div className="mt-3 flex gap-2">
                    <Button disabled={isResponding} onClick={() => void handleRespondToInvite("accept")}>
                      Accept
                    </Button>
                    <Button
                      variant="outline"
                      disabled={isResponding}
                      onClick={() => void handleRespondToInvite("decline")}
                    >
                      Decline
                    </Button>
                  </div>
                ) : null}
              </div>
            ) : null}
          </div>
        </section>

        <section className="rounded-xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-zinc-900">
          <h2 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100">
            Your Outgoing Invitations
          </h2>

          {outgoingError ? (
            <p className="mt-3 text-sm text-red-600">{outgoingError}</p>
          ) : null}

          {isLoadingOutgoing ? (
            <p className="mt-3 text-sm text-zinc-600 dark:text-zinc-300">Loading invitations...</p>
          ) : sortedOutgoing.length === 0 ? (
            <p className="mt-3 text-sm text-zinc-600 dark:text-zinc-300">No invites yet.</p>
          ) : (
            <ul className="mt-4 space-y-2">
              {sortedOutgoing.map((invitation) => (
                <li
                  key={invitation.id}
                  className="rounded-lg border border-black/10 p-3 dark:border-white/10"
                >
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <span className="font-mono text-sm tracking-wide">{invitation.inviteId}</span>
                    <span className={`rounded-full px-2 py-1 text-xs font-medium ${statusBadgeClass(invitation.status)}`}>
                      {invitation.status}
                    </span>
                  </div>
                  <p className="mt-2 text-sm text-zinc-700 dark:text-zinc-300">
                    Relation: {invitation.relation}
                  </p>
                  <p className="text-sm text-zinc-700 dark:text-zinc-300">
                    Invitee: {invitation.inviteeEmail || "Not responded"}
                  </p>
                  <p className="text-xs text-zinc-500 dark:text-zinc-400">
                    Created: {formatTimestamp(invitation.createdAt)}
                  </p>
                  <p className="text-xs text-zinc-500 dark:text-zinc-400">
                    Expires: {formatTimestamp(invitation.expiresAt)}
                  </p>
                </li>
              ))}
            </ul>
          )}
        </section>
      </main>
    </div>
  );
}
