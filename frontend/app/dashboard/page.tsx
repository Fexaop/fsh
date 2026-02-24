"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";

import { FamilyLiveMap } from "@/components/family-live-map";
import { Button } from "@/components/ui/button";
import {
  BACKEND_BASE_URL,
  clearSessionTokenCookie,
  getSessionTokenFromCookieString,
  type GoogleAuthSessionResponse,
} from "@/lib/auth";
import PixelBlast from "@/components/PixelBlast";
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

type SOSAlertMessage = {
  type: "sos_alert";
  memberId: string;
  message: string;
  createdAt: number;
};

const GEOFENCE_RADIUS_METERS = 100;
const GEOLOCATION_RETRY_DELAY_MS = 4000;
const MAX_GEOLOCATION_RETRIES = 3;

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

function isSOSAlertMessage(value: unknown): value is SOSAlertMessage {
  if (!value || typeof value !== "object") {
    return false;
  }

  const candidate = value as Record<string, unknown>;
  return (
    candidate.type === "sos_alert" &&
    typeof candidate.memberId === "string" &&
    typeof candidate.message === "string" &&
    typeof candidate.createdAt === "number"
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

function shouldRetryLocationRead(error: GeolocationPositionError): boolean {
  return error.code === error.POSITION_UNAVAILABLE || error.code === error.TIMEOUT;
}

function toRadians(value: number): number {
  return (value * Math.PI) / 180;
}

function distanceMeters(
  latitudeA: number,
  longitudeA: number,
  latitudeB: number,
  longitudeB: number,
): number {
  const earthRadiusMeters = 6371000;
  const dLat = toRadians(latitudeB - latitudeA);
  const dLon = toRadians(longitudeB - longitudeA);
  const lat1 = toRadians(latitudeA);
  const lat2 = toRadians(latitudeB);

  const haversine =
    Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos(lat1) *
      Math.cos(lat2) *
      Math.sin(dLon / 2) *
      Math.sin(dLon / 2);

  return 2 * earthRadiusMeters * Math.asin(Math.sqrt(haversine));
}

export default function DashboardPage() {
  const [isLoadingUser, setIsLoadingUser] = useState(true);
  const [user, setUser] = useState<GoogleAuthSessionResponse | null>(null);
  const [sessionToken, setSessionToken] = useState<string | null>(null);

  const [familyMembers, setFamilyMembers] = useState<FamilyMember[]>([]);
  const [isFamilyLoading, setIsFamilyLoading] = useState(false);
  const [familyError, setFamilyError] = useState<string | null>(null);

  const [socketState, setSocketState] = useState<
    "connecting" | "connected" | "disconnected" | "error"
  >("disconnected");
  const familySocketRef = useRef<WebSocket | null>(null);
  const familyMembersRef = useRef<FamilyMember[]>([]);
  const selfMemberIDRef = useRef("");
  const geoRetryTimeoutRef = useRef<number | null>(null);
  const outsideGeofenceMemberIDsRef = useRef<Set<string>>(new Set());

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
    familySocketRef.current = socket;
    let geoWatchId: number | null = null;
    let forcePositionIntervalId: number | null = null;
    let usingFallbackWatch = false;
    let geoRetryAttemptCount = 0;
    const isFirefoxLikeBrowser = /firefox|zen/i.test(navigator.userAgent);

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

    const positionOptionsWithFreshRead = (
      useFallback: boolean,
    ): PositionOptions => ({
      ...(useFallback ? fallbackGeoOptions : primaryGeoOptions),
      maximumAge: 0,
    });

    const sendLocation = (position: GeolocationPosition) => {
      if (socket.readyState !== WebSocket.OPEN) {
        return;
      }

      geoRetryAttemptCount = 0;
      clearGeoRetryTimeout();

      socket.send(
        JSON.stringify({
          type: "location_update",
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
        }),
      );
    };

    const scheduleGeoRetry = (
      useFallback: boolean,
      source: string,
      latestError: GeolocationPositionError,
    ) => {
      if (!navigator.geolocation) {
        return;
      }

      if (geoRetryAttemptCount >= MAX_GEOLOCATION_RETRIES) {
        console.error(
          `[geolocation] ${source} failed after ${MAX_GEOLOCATION_RETRIES} retries.`,
          latestError,
        );
        return;
      }

      geoRetryAttemptCount += 1;
      const attempt = geoRetryAttemptCount;

      clearGeoRetryTimeout();
      geoRetryTimeoutRef.current = window.setTimeout(() => {
        navigator.geolocation.getCurrentPosition(
          sendLocation,
          (retryError) => {
            const retryMessage = getGeolocationErrorMessage(retryError);
            if (!shouldRetryLocationRead(retryError)) {
              console.error(
                `[geolocation] retry ${attempt}/${MAX_GEOLOCATION_RETRIES} failed without retry: ${retryMessage}`,
                retryError,
              );
              return;
            }

            console.warn(
              `[geolocation] retry ${attempt}/${MAX_GEOLOCATION_RETRIES} failed: ${retryMessage}`,
            );
            scheduleGeoRetry(true, source, retryError);
          },
          positionOptionsWithFreshRead(useFallback),
        );
      }, GEOLOCATION_RETRY_DELAY_MS);
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
          const message = getGeolocationErrorMessage(error);
          if (shouldRetryLocationRead(error)) {
            console.warn(`[geolocation] watchPosition failed: ${message}`);
            if (!useFallback) {
              startGeoWatch(true);
            }
            scheduleGeoRetry(true, "watchPosition", error);
            return;
          }

          console.error(
            `[geolocation] watchPosition failed without retry: ${message}`,
            error,
          );
        },
        useFallback ? fallbackGeoOptions : primaryGeoOptions,
      );
    };

    const requestCurrentPosition = (useFallback: boolean, source: string) => {
      if (!navigator.geolocation) {
        return;
      }

      navigator.geolocation.getCurrentPosition(
        sendLocation,
        (error) => {
          const message = getGeolocationErrorMessage(error);
          if (shouldRetryLocationRead(error)) {
            console.warn(`[geolocation] ${source} failed: ${message}`);
            if (!usingFallbackWatch) {
              startGeoWatch(true);
            }
            scheduleGeoRetry(true, source, error);
            return;
          }

          console.error(
            `[geolocation] ${source} failed without retry: ${message}`,
            error,
          );
        },
        positionOptionsWithFreshRead(useFallback),
      );
    };

    socket.onopen = () => {
      setSocketState("connected");

      if (!navigator.geolocation) {
        console.error("[geolocation] Geolocation is not supported in this browser.");
        return;
      }

      if (
        !window.isSecureContext &&
        window.location.hostname !== "localhost" &&
        window.location.hostname !== "127.0.0.1"
      ) {
        console.error(
          "[geolocation] Location requires HTTPS (or localhost) to stream coordinates.",
        );
        return;
      }

      const startWithFallback = isFirefoxLikeBrowser;
      requestCurrentPosition(startWithFallback, "initial location request");
      startGeoWatch(startWithFallback);

      // Firefox/Zen on Linux can skip watch callbacks; poll periodically as a fallback.
      forcePositionIntervalId = window.setInterval(() => {
        requestCurrentPosition(usingFallbackWatch, "periodic location refresh");
      }, 15000);
    };

    socket.onmessage = (event) => {
      if (typeof event.data !== "string") {
        return;
      }

      let payload: unknown;
      try {
        payload = JSON.parse(event.data);
      } catch {
        return;
      }

      if (isSOSAlertMessage(payload)) {
        const senderMemberID = normalizeMemberId(payload.memberId);
        if (senderMemberID === selfMemberIDRef.current) {
          return;
        }

        const sender = familyMembersRef.current.find(
          (member) => normalizeMemberId(member.email) === senderMemberID,
        );
        const senderName = sender?.name ?? payload.memberId;
        toast.error(`SOS from ${senderName}`, {
          description: payload.message,
        });
        return;
      }

      handleIncomingLocationMessage(event.data);
    };

    socket.onerror = (event) => {
      setSocketState("error");
      console.error("[ws] Error while streaming live family locations.", event);
    };

    socket.onclose = () => {
      setSocketState("disconnected");
      familySocketRef.current = null;
      if (geoWatchId !== null) {
        navigator.geolocation.clearWatch(geoWatchId);
      }
      if (forcePositionIntervalId !== null) {
        window.clearInterval(forcePositionIntervalId);
      }
      clearGeoRetryTimeout();
    };

    return () => {
      familySocketRef.current = null;
      if (geoWatchId !== null) {
        navigator.geolocation.clearWatch(geoWatchId);
      }
      if (forcePositionIntervalId !== null) {
        window.clearInterval(forcePositionIntervalId);
      }
      clearGeoRetryTimeout();
      socket.close();
    };
  }, [familySocketKey, handleIncomingLocationMessage, sessionToken]);

  const selfMemberID = useMemo(
    () => (user?.email ? normalizeMemberId(user.email) : ""),
    [user?.email],
  );
  familyMembersRef.current = familyMembers;
  selfMemberIDRef.current = selfMemberID;

  const geofenceCenter = useMemo(() => {
    if (!selfMemberID) {
      return null;
    }
    return locationsByMember[selfMemberID] ?? null;
  }, [locationsByMember, selfMemberID]);

  const mapMembers = useMemo(() => {
    const combined = [...familyMembers];
    const alreadyExists = combined.some(
      (member) => normalizeMemberId(member.email) === selfMemberID,
    );

    if (selfMemberID && !alreadyExists) {
      combined.push({
        id: 0,
        name: user?.name ?? "You",
        email: selfMemberID,
        relation: "You",
        avatarUrl: user?.picture,
      });
    }

    return combined;
  }, [familyMembers, selfMemberID, user?.name, user?.picture]);

  useEffect(() => {
    if (!geofenceCenter) {
      outsideGeofenceMemberIDsRef.current = new Set();
      return;
    }

    const nextOutside = new Set<string>();
    for (const member of familyMembers) {
      const memberID = normalizeMemberId(member.email);
      const memberLocation = locationsByMember[memberID];
      if (!memberLocation) {
        continue;
      }

      const distance = distanceMeters(
        geofenceCenter.latitude,
        geofenceCenter.longitude,
        memberLocation.latitude,
        memberLocation.longitude,
      );

      if (distance > GEOFENCE_RADIUS_METERS) {
        nextOutside.add(memberID);
        if (!outsideGeofenceMemberIDsRef.current.has(memberID)) {
          toast.error(`${member.name} left the ${GEOFENCE_RADIUS_METERS}m safety radius.`, {
            description: `Current distance: ${Math.round(distance)}m`,
          });
        }
      }
    }

    for (const previousMemberID of outsideGeofenceMemberIDsRef.current) {
      if (nextOutside.has(previousMemberID)) {
        continue;
      }

      const member = familyMembers.find(
        (item) => normalizeMemberId(item.email) === previousMemberID,
      );
      toast.success(
        `${member?.name ?? previousMemberID} returned inside the ${GEOFENCE_RADIUS_METERS}m safety radius.`,
      );
    }

    outsideGeofenceMemberIDsRef.current = nextOutside;
  }, [familyMembers, geofenceCenter, locationsByMember]);

  const handleOpenInvitations = () => {
    router.push("/dashboard/invitations");
  };

  const handleOpenSettings = () => {
    router.push("/setting");
  };

  const waitForSocketOpenAndSend = useCallback(
    (socket: WebSocket, payload: string): Promise<void> =>
      new Promise((resolve, reject) => {
        const timeoutID = window.setTimeout(() => {
          cleanup();
          reject(new Error("Timed out waiting for WebSocket connection."));
        }, 6000);

        const cleanup = () => {
          window.clearTimeout(timeoutID);
          socket.removeEventListener("open", handleOpen);
          socket.removeEventListener("error", handleFailure);
          socket.removeEventListener("close", handleFailure);
        };

        const handleOpen = () => {
          cleanup();
          try {
            socket.send(payload);
            resolve();
          } catch (error) {
            reject(
              error instanceof Error
                ? error
                : new Error("Failed to send SOS payload."),
            );
          }
        };

        const handleFailure = () => {
          cleanup();
          reject(new Error("WebSocket closed before SOS could be sent."));
        };

        socket.addEventListener("open", handleOpen);
        socket.addEventListener("error", handleFailure);
        socket.addEventListener("close", handleFailure);
      }),
    [],
  );

  const handleSOS = async () => {
    const socket = familySocketRef.current;

    const rawMessage = window.prompt(
      "Enter SOS message for your family:",
      "I need help. Please check on me now.",
    );
    if (rawMessage === null) {
      return;
    }

    const message = rawMessage.trim();
    if (!message) {
      toast.error("SOS message cannot be empty.");
      return;
    }

    const selfLocation = selfMemberID ? locationsByMember[selfMemberID] : null;
    const locationSuffix = selfLocation
      ? ` Location: https://maps.google.com/?q=${selfLocation.latitude},${selfLocation.longitude}`
      : "";
    const payload = JSON.stringify({
      type: "sos_alert",
      message: `${message}${locationSuffix}`,
    });

    try {
      if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(payload);
      } else if (socket && socket.readyState === WebSocket.CONNECTING) {
        await waitForSocketOpenAndSend(socket, payload);
      } else {
        if (!sessionToken) {
          throw new Error("Missing session token for SOS fallback.");
        }
        const wsBaseURL = toWebSocketBaseURL(BACKEND_BASE_URL);
        const wsURL = `${wsBaseURL}/ws?sessionToken=${encodeURIComponent(sessionToken)}`;
        const fallbackSocket = new WebSocket(wsURL);
        await waitForSocketOpenAndSend(fallbackSocket, payload);
        fallbackSocket.close();
      }

      toast.error("SOS sent to family.");
    } catch (error) {
      console.error("[sos] Failed to send SOS over WebSocket.", error);
      toast.error("SOS failed.", {
        description: "Could not send the alert message. Please retry.",
      });
    }
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
    <div className="relative min-h-screen bg-zinc-100 dark:bg-zinc-950">
      {/* PixelBlast Background */}
      <div className="fixed inset-0 -z-10 w-full h-screen">
        <PixelBlast
          variant="square"
          pixelSize={4}
          color="#B19EEF"
          patternScale={2}
          patternDensity={1}
          pixelSizeJitter={0}
          enableRipples
          rippleSpeed={0.4}
          rippleThickness={0.12}
          rippleIntensityScale={1.5}
          liquid={false}
          liquidStrength={0.12}
          liquidRadius={1.2}
          liquidWobbleSpeed={5}
          speed={0.5}
          edgeFade={0.25}
          transparent
        />
      </div>

      <header className="sticky top-0 z-50 border-b border-black/10 bg-white/65 backdrop-blur-md dark:border-white/10 dark:bg-black/45">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4">
          <span className="text-sm font-semibold tracking-wide text-zinc-900 dark:text-zinc-100">
            FSH Dashboard
          </span>
          <div className="flex items-center gap-2">
            <Button variant="destructive" onClick={handleSOS}>
              SOS
            </Button>
            <Button variant="outline" onClick={handleOpenSettings}>
              Settings
            </Button>
            <Button variant="outline" onClick={handleOpenInvitations}>
              Invitations
            </Button>
            <Button variant="outline" onClick={handleLogout}>
              Logout
            </Button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-10 relative z-10">
        <div className="rounded-xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-zinc-900">
          {isLoadingUser ? (
            <p className="text-zinc-600 dark:text-zinc-300">Loading profile...</p>
          ) : (
            <>
              <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
                Welcome to your dashboard {user.name}
              </h1>
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

            <p className="mb-4 text-xs text-zinc-500 dark:text-zinc-400">
              Safety radius: {GEOFENCE_RADIUS_METERS}m around your latest location.
            </p>

            <FamilyLiveMap
              familyMembers={mapMembers}
              locationsByMember={locationsByMember}
              geofenceCenter={geofenceCenter}
              geofenceRadiusMeters={GEOFENCE_RADIUS_METERS}
            />
          </div>
        </section>
      </main>
    </div>
  );
}
