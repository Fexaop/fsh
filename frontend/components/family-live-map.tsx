"use client";

import Script from "next/script";
import { useEffect, useMemo, useRef, useState } from "react";

type FamilyMember = {
  id: number;
  name: string;
  email: string;
  relation: string;
  avatarUrl?: string;
};

type MemberLocation = {
  memberId: string;
  latitude: number;
  longitude: number;
  updatedAt: number;
};

type FamilyLiveMapProps = {
  familyMembers: FamilyMember[];
  locationsByMember: Record<string, MemberLocation>;
  geofenceCenter: MemberLocation | null;
  geofenceRadiusMeters: number;
};

type LeafletMap = {
  setView(center: [number, number], zoom: number): LeafletMap;
  fitBounds(
    bounds: LeafletBounds,
    options?: {
      padding?: [number, number];
      maxZoom?: number;
    },
  ): LeafletMap;
  remove(): void;
};

type LeafletMarker = {
  addTo(map: LeafletMap): LeafletMarker;
  bindPopup(content: string): LeafletMarker;
  remove(): void;
};

type LeafletCircle = {
  addTo(map: LeafletMap): LeafletCircle;
  bindPopup(content: string): LeafletCircle;
  remove(): void;
};

type LeafletBounds = {
  isValid(): boolean;
};

type Leaflet = {
  map(element: HTMLElement): LeafletMap;
  tileLayer(
    urlTemplate: string,
    options: {
      attribution: string;
      maxZoom?: number;
    },
  ): { addTo(map: LeafletMap): void };
  marker(
    position: [number, number],
    options?: {
      icon?: unknown;
    },
  ): LeafletMarker;
  divIcon(options: {
    className?: string;
    html: string;
    iconSize?: [number, number];
    iconAnchor?: [number, number];
    popupAnchor?: [number, number];
  }): unknown;
  circle(
    position: [number, number],
    options: {
      radius: number;
      color?: string;
      fillColor?: string;
      fillOpacity?: number;
      weight?: number;
      dashArray?: string;
    },
  ): LeafletCircle;
  latLngBounds(points: [number, number][]): LeafletBounds;
};

declare global {
  interface Window {
    L?: Leaflet;
  }
}

function normalizeMemberId(value: string): string {
  return value.trim().toLowerCase();
}

function getInitials(value: string): string {
  const parts = value
    .trim()
    .split(/\s+/)
    .filter(Boolean);
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }
  return `${parts[0][0] ?? ""}${parts[1][0] ?? ""}`.toUpperCase();
}

function escapeHTML(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function createAvatarMarkerHTML(member: FamilyMember | undefined, memberId: string): string {
  const displayName = member?.name?.trim() || memberId;
  const avatarUrl = member?.avatarUrl?.trim() || "";
  const initials = escapeHTML(getInitials(displayName));

  const avatarContent = avatarUrl
    ? `<img src="${escapeHTML(avatarUrl)}" alt="${escapeHTML(displayName)}" style="width:100%;height:100%;object-fit:cover;display:block;" />`
    : `<span style="font:600 12px/1 sans-serif;color:#fff;letter-spacing:0.02em;">${initials}</span>`;

  return `
    <div style="display:flex;flex-direction:column;align-items:center;transform:translateY(-2px);">
      <div style="width:40px;height:40px;border-radius:9999px;overflow:hidden;background:#1f2937;border:2px solid #fff;box-shadow:0 6px 16px rgba(0,0,0,0.28);display:flex;align-items:center;justify-content:center;">
        ${avatarContent}
      </div>
      <div style="margin-top:-1px;width:0;height:0;border-left:6px solid transparent;border-right:6px solid transparent;border-top:10px solid #1f2937;"></div>
    </div>
  `;
}

export function FamilyLiveMap({
  familyMembers,
  locationsByMember,
  geofenceCenter,
  geofenceRadiusMeters,
}: FamilyLiveMapProps) {
  const [leafletReady, setLeafletReady] = useState(false);
  const mapContainerRef = useRef<HTMLDivElement | null>(null);
  const mapRef = useRef<LeafletMap | null>(null);
  const markersRef = useRef<LeafletMarker[]>([]);
  const geofenceCircleRef = useRef<LeafletCircle | null>(null);

  const membersByEmail = useMemo(() => {
    const map = new Map<string, FamilyMember>();
    for (const member of familyMembers) {
      map.set(normalizeMemberId(member.email), member);
    }
    return map;
  }, [familyMembers]);

  const locations = useMemo(() => {
    return Object.values(locationsByMember);
  }, [locationsByMember]);

  useEffect(() => {
    if (!leafletReady || !mapContainerRef.current || mapRef.current || !window.L) {
      return;
    }

    const map = window.L.map(mapContainerRef.current).setView([20.5937, 78.9629], 4);
    window.L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution: "&copy; OpenStreetMap contributors",
      maxZoom: 19,
    }).addTo(map);
    mapRef.current = map;
  }, [leafletReady]);

  useEffect(() => {
    if (!leafletReady || !mapRef.current || !window.L) {
      return;
    }

    for (const marker of markersRef.current) {
      marker.remove();
    }
    markersRef.current = [];
    geofenceCircleRef.current?.remove();
    geofenceCircleRef.current = null;

    const points: [number, number][] = [];

    for (const location of locations) {
      const member = membersByEmail.get(normalizeMemberId(location.memberId));
      const markerLabel = member
        ? `${member.name} (${member.relation})`
        : location.memberId;
      const updatedAt = new Date(location.updatedAt * 1000).toLocaleTimeString();
      const escapedMarkerLabel = escapeHTML(markerLabel);
      const iconHTML = createAvatarMarkerHTML(member, location.memberId);
      const icon = window.L.divIcon({
        className: "fsh-avatar-marker",
        html: iconHTML,
        iconSize: [42, 54],
        iconAnchor: [21, 54],
        popupAnchor: [0, -46],
      });

      const marker = window.L.marker([location.latitude, location.longitude], { icon })
        .addTo(mapRef.current)
        .bindPopup(
          `${escapedMarkerLabel}<br/>${location.latitude.toFixed(5)}, ${location.longitude.toFixed(5)}<br/>Updated ${updatedAt}`,
        );

      markersRef.current.push(marker);
      points.push([location.latitude, location.longitude]);
    }

    if (geofenceCenter && geofenceRadiusMeters > 0) {
      geofenceCircleRef.current = window.L
        .circle([geofenceCenter.latitude, geofenceCenter.longitude], {
          radius: geofenceRadiusMeters,
          color: "#dc2626",
          fillColor: "#ef4444",
          fillOpacity: 0.12,
          weight: 2,
          dashArray: "4 4",
        })
        .addTo(mapRef.current)
        .bindPopup(`Safety radius (${Math.round(geofenceRadiusMeters)}m)`);
      points.push([geofenceCenter.latitude, geofenceCenter.longitude]);
    }

    if (points.length === 1) {
      mapRef.current.setView(points[0], 13);
      return;
    }

    if (points.length > 1) {
      const bounds = window.L.latLngBounds(points);
      if (bounds.isValid()) {
        mapRef.current.fitBounds(bounds, { padding: [36, 36], maxZoom: 15 });
      }
    }
  }, [geofenceCenter, geofenceRadiusMeters, leafletReady, locations, membersByEmail]);

  useEffect(() => {
    return () => {
      for (const marker of markersRef.current) {
        marker.remove();
      }
      markersRef.current = [];
      geofenceCircleRef.current?.remove();
      geofenceCircleRef.current = null;

      if (mapRef.current) {
        mapRef.current.remove();
        mapRef.current = null;
      }
    };
  }, []);

  return (
    <div className="relative">
      <Script
        src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"
        strategy="afterInteractive"
        onLoad={() => setLeafletReady(true)}
      />

      <div
        ref={mapContainerRef}
        className="h-[420px] w-full rounded-xl border border-black/10"
      />

      {!leafletReady ? (
        <div className="pointer-events-none absolute inset-0 grid place-items-center rounded-xl bg-white/80 text-sm text-zinc-600">
          Loading map...
        </div>
      ) : null}
    </div>
  );
}
