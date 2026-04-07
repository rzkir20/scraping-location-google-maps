import L from "leaflet";

import "leaflet/dist/leaflet.css";

const DEFAULT_CENTER: L.LatLngTuple = [-6.2088, 106.8456];

let map: L.Map | null = null;
let demoLayer: L.LayerGroup | null = null;
let resultsLayer: L.LayerGroup | null = null;
let searchDisc: L.Circle | null = null;
let lastCenter: L.LatLngTuple = DEFAULT_CENTER;
let resizeObserver: ResizeObserver | null = null;

function parseCenter(mc: MapCenter | null | undefined): L.LatLngTuple | null {
  if (!mc?.lat || !mc.lng) return null;
  const lat = parseFloat(mc.lat);
  const lng = parseFloat(mc.lng);
  if (!Number.isFinite(lat) || !Number.isFinite(lng)) return null;
  return [lat, lng];
}

function addDemoPins() {
  if (!map || !demoLayer) return;
  demoLayer.clearLayers();
  const offsets: L.LatLngTuple[] = [
    [0.035, 0.025],
    [-0.028, 0.038],
    [0.018, -0.042],
    [-0.04, -0.018],
    [0.012, 0.048],
  ];
  const colors = ["#FFD700", "#0066cc", "#0066cc", "#FF8C00", "#FF4444"];
  offsets.forEach(([dLat, dLng], i) => {
    const lat = DEFAULT_CENTER[0] + dLat;
    const lng = DEFAULT_CENTER[1] + dLng;
    L.circleMarker([lat, lng], {
      radius: 7,
      color: "#ffffff",
      weight: 2,
      fillColor: colors[i] ?? "#666666",
      fillOpacity: 0.95,
    })
      .bindTooltip(`Contoh ${i + 1}`, { direction: "top" })
      .addTo(demoLayer!);
  });
}

function wireZoomButtons() {
  document
    .getElementById("map-zoom-in")
    ?.addEventListener("click", () => map?.zoomIn());
  document
    .getElementById("map-zoom-out")
    ?.addEventListener("click", () => map?.zoomOut());
  document.getElementById("map-recenter")?.addEventListener("click", () => {
    if (map)
      map.setView(lastCenter, Math.max(map.getZoom(), 12), { animate: true });
  });
}

/** Peta Leaflet + tile gelap; dipanggil sekali. */
export function ensureMap(): L.Map | null {
  const el = document.getElementById("map-preview");
  if (!el || map) return map;

  map = L.map(el, {
    zoomControl: false,
    attributionControl: true,
  }).setView(DEFAULT_CENTER, 11);

  L.tileLayer("https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png", {
    attribution:
      '&copy; <a href="https://www.openstreetmap.org/copyright">OSM</a> &copy; CARTO',
    subdomains: "abcd",
    maxZoom: 19,
  }).addTo(map);

  demoLayer = L.layerGroup().addTo(map);
  resultsLayer = L.layerGroup().addTo(map);

  wireZoomButtons();

  const stage = document.getElementById("map-stage");
  if (stage && typeof ResizeObserver !== "undefined") {
    resizeObserver = new ResizeObserver(() => {
      map?.invalidateSize();
    });
    resizeObserver.observe(stage);
  }

  scheduleInvalidate();
  return map;
}

export function scheduleInvalidate() {
  setTimeout(() => map?.invalidateSize(), 50);
  setTimeout(() => map?.invalidateSize(), 400);
}

export function invalidateMapSize() {
  scheduleInvalidate();
}

export function setIdleBannerVisible(visible: boolean) {
  const b = document.getElementById("map-idle-banner");
  if (b) b.classList.toggle("hidden", !visible);
}

/** Sebelum scraping: Jakarta + pin contoh. */
export function showIdlePreviewMap() {
  ensureMap();
  if (!map || !demoLayer) return;
  lastCenter = DEFAULT_CENTER;
  map.setView(DEFAULT_CENTER, 11);
  if (searchDisc && map.hasLayer(searchDisc)) {
    map.removeLayer(searchDisc);
    searchDisc = null;
  }
  resultsLayer?.clearLayers();
  addDemoPins();
  setIdleBannerVisible(true);
}

/** Setelah POST / polling: fokus ke lokasi pencarian (geocode backend). */
export function focusSearchOnMap(mc: MapCenter | null | undefined) {
  ensureMap();
  if (!map) return;
  const c = parseCenter(mc) ?? DEFAULT_CENTER;
  lastCenter = c;
  map.flyTo(c, 13, { duration: 1.1 });
  if (searchDisc) {
    map.removeLayer(searchDisc);
    searchDisc = null;
  }
  searchDisc = L.circle(c, {
    radius: 2400,
    color: "#0066cc",
    weight: 1,
    fillColor: "#0066cc",
    fillOpacity: 0.14,
  }).addTo(map);
  demoLayer?.clearLayers();
  setIdleBannerVisible(false);
  scheduleInvalidate();
}

/** Titik hasil: posisi perkiraan mengelilingi pusat (data scraper tanpa koordinat). */
export function plotStoresOnMap(
  stores: Store[],
  mc: MapCenter | null | undefined,
) {
  ensureMap();
  if (!map || !resultsLayer) return;
  const base = parseCenter(mc) ?? lastCenter;
  lastCenter = base;
  map.flyTo(base, 13, { duration: 0.6 });
  resultsLayer.clearLayers();
  if (!stores.length) return;
  const n = stores.length;
  const radius = 0.0075;
  stores.forEach((s, i) => {
    const angle = (2 * Math.PI * i) / n;
    const jitter = 1 + (i % 4) * 0.12;
    const lat = base[0] + radius * jitter * Math.sin(angle);
    const lng = base[1] + radius * jitter * Math.cos(angle);
    const name = esc(s.name || "—");
    const phone = esc(s.phone || "");
    L.circleMarker([lat, lng], {
      radius: 6,
      color: "#ffffff",
      weight: 2,
      fillColor: "#0066cc",
      fillOpacity: 0.9,
    })
      .bindPopup(
        `<strong>${name}</strong><br/><span style="opacity:.85">${phone}</span>`,
      )
      .addTo(resultsLayer);
  });
  scheduleInvalidate();
}

function esc(s: string) {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/"/g, "&quot;");
}
