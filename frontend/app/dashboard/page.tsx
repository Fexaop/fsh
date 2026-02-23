"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";

import { FamilyLiveMap } from "@/components/family-live-map";
import { Button } from "@/components/ui/button";
import {
  BACKEND_BASE_URL,
  clearSessionTokenCookie,
  getSessionTokenFromCookieString,
  type GoogleAuthSessionResponse,
} from "@/lib/auth";

type FamilyMember = {
  id: number;
  name: string;
  email: string;
  relation: string;
  avatarUrl?: string;
};

type FamilyMembersResponse = {
  members: FamilyMember[];
};

type MemberLocation = {
  memberId: string;
  latitude: number;
  longitude: number;
  updatedAt: number;
};

type LocationSnapshotMessage = {
  type: "location_snapshot";
  locations: MemberLocation[];
};

type LocationUpdateMessage = {
  type: "location_update";
  location: MemberLocation;
};

type LocationMessage = LocationSnapshotMessage | LocationUpdateMessage;

function normalizeMemberId(value: string): string {
  return value.trim().toLowerCase();
}

function toWebSocketBaseURL(httpURL: string): string {
  const trimmed = httpURL.replace(/\/+$/, "");

  if (trimmed.startsWith("https://")) {
    return `wss://${trimmed.slice("https://".length)}`;
  }
  if (trimmed.startsWith("http://")) {
    return `ws://${trimmed.slice("http://".length)}`;
  }

  return trimmed;
}

function isValidLocation(value: unknown): value is MemberLocation {
  if (!value || typeof value !== "object") {
    return false;
  }

  const candidate = value as Record<string, unknown>;
  return (
    typeof candidate.memberId === "string" &&
    typeof candidate.latitude === "number" &&
    typeof candidate.longitude === "number" &&
    typeof candidate.updatedAt === "number"
  );
}

function getGeolocationErrorMessage(error: GeolocationPositionError): string {
  if (error.code === error.PERMISSION_DENIED) {
    return "Location permission is blocked. Allow location access for this site.";
  }
  if (error.code === error.POSITION_UNAVAILABLE) {
    return "Location is currently unavailable. Check device location services and network.";
  }
  if (error.code === error.TIMEOUT) {
    return "Location request timed out. Trying a lower-accuracy fallback.";
  }

  return "Unable to acquire location from the browser. Verify location services are enabled.";
}

export default function DashboardPage() {
  const [isLoadingUser, setIsLoadingUser] = useState(true);
  const [user, setUser] = useState<GoogleAuthSessionResponse | null>(null);
  const [sessionToken, setSessionToken] = useState<string | null>(null);

  const [familyMembers, setFamilyMembers] = useState<FamilyMember[]>([]);
  const [isFamilyLoading, setIsFamilyLoading] = useState(false);
  const [familyError, setFamilyError] = useState<string | null>(null);

  const [socketState, setSocketState] = useState<"connecting" | "connected" | "disconnected" | "error">("disconnected");
  const [geoError, setGeoError] = useState<string | null>(null);
  const geoRetryTimeoutRef = useRef<number | null>(null);

  const [locationsByMember, setLocationsByMember] = useState<
    Record<string, MemberLocation>
  >({});

  const router = useRouter();

  const loadFamilyMembers = useCallback(
    async (token: string) => {
      setIsFamilyLoading(true);
      setFamilyError(null);

      try {
        const response = await fetch(`${BACKEND_BASE_URL}/family/members`, {
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
          throw new Error("Failed to fetch family members");
        }

        const payload = (await response.json()) as FamilyMembersResponse;
        setFamilyMembers(payload.members ?? []);
      } catch (error) {
        setFamilyError(
          error instanceof Error
            ? error.message
            : "Failed to fetch family members",
        );
      } finally {
        setIsFamilyLoading(false);
      }
    },
    [router],
  );

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

    void loadCurrentUser();

    return () => {
      cancelled = true;
    };
  }, [router]);

  useEffect(() => {
    if (!sessionToken) {
      return;
    }

    void loadFamilyMembers(sessionToken);

    const refreshInterval = window.setInterval(() => {
      void loadFamilyMembers(sessionToken);
    }, 20000);

    return () => {
      window.clearInterval(refreshInterval);
    };
  }, [loadFamilyMembers, sessionToken]);

  const handleIncomingLocationMessage = useCallback((rawData: string) => {
    let message: LocationMessage;
    try {
      message = JSON.parse(rawData) as LocationMessage;
    } catch {
      return;
    }

    if (message.type === "location_snapshot") {
      if (!Array.isArray(message.locations)) {
        return;
      }

      const nextState: Record<string, MemberLocation> = {};
      for (const location of message.locations) {
        if (!isValidLocation(location)) {
          continue;
        }
        const memberId = normalizeMemberId(location.memberId);
        nextState[memberId] = {
          memberId,
          latitude: location.latitude,
          longitude: location.longitude,
          updatedAt: location.updatedAt,
        };
      }
      setLocationsByMember(nextState);
      return;
    }

    if (message.type === "location_update" && isValidLocation(message.location)) {
      const memberId = normalizeMemberId(message.location.memberId);
      setLocationsByMember((current) => ({
        ...current,
        [memberId]: {
          memberId,
          latitude: message.location.latitude,
          longitude: message.location.longitude,
          updatedAt: message.location.updatedAt,
        },
      }));
    }
  }, []);

  const familySocketKey = useMemo(
    () =>
      familyMembers
        .map((member) => normalizeMemberId(member.email))
        .sort()
        .join("|"),
    [familyMembers],
  );

  useEffect(() => {
    if (!sessionToken) {
      return;
    }

    const wsBaseURL = toWebSocketBaseURL(BACKEND_BASE_URL);
    const wsURL = `${wsBaseURL}/ws?sessionToken=${encodeURIComponent(sessionToken)}`;
    const socket = new WebSocket(wsURL);
    let geoWatchId: number | null = null;
    let usingFallbackWatch = false;

    setSocketState("connecting");

    const clearGeoRetryTimeout = () => {
      if (geoRetryTimeoutRef.current !== null) {
        window.clearTimeout(geoRetryTimeoutRef.current);
        geoRetryTimeoutRef.current = null;
      }
    };

    const primaryGeoOptions: PositionOptions = {
      enableHighAccuracy: true,
      timeout: 15000,
      maximumAge: 5000,
    };

    const fallbackGeoOptions: PositionOptions = {
      enableHighAccuracy: false,
      timeout: 30000,
      maximumAge: 60000,
    };

    const sendLocation = (position: GeolocationPosition) => {
      if (socket.readyState !== WebSocket.OPEN) {
        return;
      }

      socket.send(
        JSON.stringify({
          type: "location_update",
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
        }),
      );
    };

    const startGeoWatch = (useFallback: boolean) => {
      if (!navigator.geolocation) {
        return;
      }

      if (geoWatchId !== null) {
        navigator.geolocation.clearWatch(geoWatchId);
      }

      usingFallbackWatch = useFallback;
      geoWatchId = navigator.geolocation.watchPosition(
        sendLocation,
        (error) => {
          setGeoError(getGeolocationErrorMessage(error));

          if (
            !useFallback &&
            (error.code === error.POSITION_UNAVAILABLE || error.code === error.TIMEOUT)
          ) {
            startGeoWatch(true);
          }
        },
        useFallback ? fallbackGeoOptions : primaryGeoOptions,
      );
    };

    const retryCurrentPosition = (useFallback: boolean) => {
      clearGeoRetryTimeout();
      geoRetryTimeoutRef.current = window.setTimeout(() => {
        if (!navigator.geolocation) {
          return;
        }

        navigator.geolocation.getCurrentPosition(
          sendLocation,
          (retryError) => setGeoError(getGeolocationErrorMessage(retryError)),
          useFallback ? fallbackGeoOptions : primaryGeoOptions,
        );
      }, 4000);
    };

    const handleGeoError = (error: GeolocationPositionError) => {
      setGeoError(getGeolocationErrorMessage(error));

      if (error.code === error.POSITION_UNAVAILABLE || error.code === error.TIMEOUT) {
        if (!usingFallbackWatch) {
          startGeoWatch(true);
        }
        retryCurrentPosition(true);
      }
    };

    socket.onopen = () => {
      setSocketState("connected");
      setGeoError(null);

      if (!navigator.geolocation) {
        setGeoError("Geolocation is not supported in this browser.");
        return;
      }

      if (
        !window.isSecureContext &&
        window.location.hostname !== "localhost" &&
        window.location.hostname !== "127.0.0.1"
      ) {
        setGeoError("Location requires HTTPS (or localhost). Open the app on HTTPS.");
        return;
      }

      navigator.geolocation.getCurrentPosition(sendLocation, handleGeoError, {
        ...primaryGeoOptions,
        maximumAge: 0,
      });

      startGeoWatch(false);
    };

    socket.onmessage = (event) => {
      if (typeof event.data !== "string") {
        return;
      }
      handleIncomingLocationMessage(event.data);
    };

    socket.onerror = () => {
      setSocketState("error");
    };

    socket.onclose = () => {
      setSocketState("disconnected");
      if (geoWatchId !== null) {
        navigator.geolocation.clearWatch(geoWatchId);
      }
      clearGeoRetryTimeout();
    };

    return () => {
      if (geoWatchId !== null) {
        navigator.geolocation.clearWatch(geoWatchId);
      }
      clearGeoRetryTimeout();
      socket.close();
    };
  }, [familySocketKey, handleIncomingLocationMessage, sessionToken]);

  const mapMembers = useMemo(() => {
    const combined = [...familyMembers];
    const selfEmail = user?.email ? normalizeMemberId(user.email) : "";
    const alreadyExists = combined.some(
      (member) => normalizeMemberId(member.email) === selfEmail,
    );

    if (selfEmail && !alreadyExists) {
      combined.push({
        id: 0,
        name: user?.name ?? "You",
        email: selfEmail,
        relation: "You",
        avatarUrl: user?.picture,
      });
    }

    return combined;
  }, [familyMembers, user?.email, user?.name, user?.picture]);

  const handleOpenInvitations = () => {
    router.push("/dashboard/invitations");
  };

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

  const formatMemberLocation = (memberEmail: string): string => {
    const location = locationsByMember[normalizeMemberId(memberEmail)];
    if (!location) {
      return "No live location yet.";
    }

    const formattedTime = new Date(location.updatedAt * 1000).toLocaleTimeString();
    return `${location.latitude.toFixed(5)}, ${location.longitude.toFixed(5)} • ${formattedTime}`;
  };

  return (
    <div className="min-h-screen bg-zinc-100 dark:bg-zinc-950">
      <header className="sticky top-0 z-50 border-b border-black/10 bg-white/65 backdrop-blur-md dark:border-white/10 dark:bg-black/45">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4">
          <span className="text-sm font-semibold tracking-wide text-zinc-900 dark:text-zinc-100">
            FSH Dashboard
          </span>
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={handleOpenInvitations}>
              Invitations
            </Button>
            <Button variant="outline" onClick={handleLogout}>
              Logout
            </Button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-10">
        <div className="rounded-xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-zinc-900">
          {isLoadingUser ? (
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

        <section className="mt-6 grid gap-6 lg:grid-cols-[minmax(0,1.05fr)_minmax(0,1fr)]">
          <div className="space-y-6 rounded-xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-zinc-900">
            <div className="flex items-center justify-between">
              <h2 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100">
                Family Members
              </h2>
              <div className="flex items-center gap-2">
                <span className="text-xs text-zinc-500 dark:text-zinc-400">
                  {isFamilyLoading ? "Refreshing..." : `${familyMembers.length} members`}
                </span>
                <Button variant="outline" onClick={handleOpenInvitations}>
                  Manage
                </Button>
              </div>
            </div>

            {familyError ? (
              <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                {familyError}
              </p>
            ) : null}

            {familyMembers.length === 0 ? (
              <p className="text-sm text-zinc-600 dark:text-zinc-400">
                No accepted family members yet. Create an invite in Invitations and ask them to accept using the invite ID.
              </p>
            ) : (
              <ul className="space-y-2">
                {familyMembers.map((member) => (
                  <li
                    key={member.id}
                    className="rounded-lg border border-black/10 p-3 dark:border-white/10"
                  >
                    <p className="font-medium text-zinc-900 dark:text-zinc-100">
                      {member.name}
                    </p>
                    <p className="text-sm text-zinc-600 dark:text-zinc-300">
                      {member.relation} • {member.email}
                    </p>
                    <p className="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
                      {formatMemberLocation(member.email)}
                    </p>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="rounded-xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-zinc-900">
            <div className="mb-4 flex items-center justify-between gap-2">
              <h2 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100">
                Live Location Map
              </h2>
              <span
                className={`rounded-full px-3 py-1 text-xs font-medium ${
                  socketState === "connected"
                    ? "bg-emerald-100 text-emerald-700"
                    : socketState === "connecting"
                      ? "bg-amber-100 text-amber-700"
                      : "bg-zinc-200 text-zinc-700"
                }`}
              >
                {socketState}
              </span>
            </div>

            {geoError ? (
              <p className="mb-4 rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-700">
                {geoError}
              </p>
            ) : null}

            <FamilyLiveMap
              familyMembers={mapMembers}
              locationsByMember={locationsByMember}
            />
          </div>
        </section>
      </main>
    </div>
  );
}
