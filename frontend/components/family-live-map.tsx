"use client";

import Script from "next/script";
import { useEffect, useMemo, useRef, useState } from "react";

type FamilyMember = {
  id: number;
  name: string;
  email: string;
  relation: string;
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
  marker(position: [number, number]): LeafletMarker;
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

function escapeHTML(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

export function FamilyLiveMap({
  familyMembers,
  locationsByMember,
}: FamilyLiveMapProps) {
  const [leafletReady, setLeafletReady] = useState(false);
  const mapContainerRef = useRef<HTMLDivElement | null>(null);
  const mapRef = useRef<LeafletMap | null>(null);
  const markersRef = useRef<LeafletMarker[]>([]);

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

    const points: [number, number][] = [];

    for (const location of locations) {
      const member = membersByEmail.get(normalizeMemberId(location.memberId));
      const markerLabel = member
        ? `${member.name} (${member.relation})`
        : location.memberId;
      const updatedAt = new Date(location.updatedAt * 1000).toLocaleTimeString();
      const escapedMarkerLabel = escapeHTML(markerLabel);

      const marker = window.L.marker([location.latitude, location.longitude])
        .addTo(mapRef.current)
        .bindPopup(
          `${escapedMarkerLabel}<br/>${location.latitude.toFixed(5)}, ${location.longitude.toFixed(5)}<br/>Updated ${updatedAt}`,
        );

      markersRef.current.push(marker);
      points.push([location.latitude, location.longitude]);
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
  }, [leafletReady, locations, membersByEmail]);

  useEffect(() => {
    return () => {
      for (const marker of markersRef.current) {
        marker.remove();
      }
      markersRef.current = [];

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
