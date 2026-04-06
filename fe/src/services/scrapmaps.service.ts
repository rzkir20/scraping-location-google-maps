import { closeModal, openModal } from "../lib/dashboardDialog";
import type { MapCenter } from "../lib/dashboardMap";

type NotificationKind = "info" | "success" | "error";

interface NotificationEntry {
  id: string;
  title: string;
  body: string;
  at: number;
  kind: NotificationKind;
}

interface SessionRow {
  at?: string;
  keyword?: string;
  location?: string;
  results?: number;
  jobId?: string;
  status?: string;
}

interface ScrapeStoreRow {
  name?: string;
  address?: string;
  phone?: string;
}

interface ScrapeStatusJson {
  status?: string;
  error?: string;
  logs?: string[];
  stores?: ScrapeStoreRow[];
  mapCenter?: MapCenter | null;
  progress?: { saved?: number; target?: number };
}

const API_BASE =
  "https://scraping-location-google-maps-production.up.railway.app";

export const scrapMapsApiBase = API_BASE;

/** Cek /api/health tanpa impor Leaflet (chunk peta tetap terpisah dari pemanggil yang ringan). */
export function runBackendHealthCheck(
  apiBase: string = scrapMapsApiBase,
): void {
  const pill = document.getElementById("header-status-pill");
  const text = document.getElementById("header-status-text");
  const dot = document.getElementById("header-status-dot");
  if (!pill || !text) return;

  const setDot = (cls: string) => {
    if (dot) dot.className = cls;
  };
  const timeoutMs = 8000;
  const ctrl = new AbortController();
  const timer = window.setTimeout(() => ctrl.abort(), timeoutMs);
  fetch(`${apiBase}/api/health`, { signal: ctrl.signal })
    .then((r) => {
      window.clearTimeout(timer);
      if (!r.ok) return Promise.reject(new Error("not ok"));
      return r.json() as Promise<{ ok?: boolean }>;
    })
    .then((j) => {
      if (j?.ok) {
        text.textContent = "Backend siap";
        pill.className =
          "flex items-center gap-2 px-3 py-1 rounded-full border text-xs font-medium bg-green-500/10 border-green-500/20 text-green-500";
        setDot("w-2 h-2 rounded-full bg-green-500 animate-pulse shrink-0");
      } else {
        throw new Error("bad");
      }
    })
    .catch(() => {
      window.clearTimeout(timer);
      text.textContent = "Backend offline";
      pill.className =
        "flex items-center gap-2 px-3 py-1 rounded-full border text-xs font-medium bg-red-500/10 border-red-500/20 text-red-400";
      setDot("w-2 h-2 rounded-full bg-red-400 shrink-0");
    });
}

/** Dynamic import: kalau Leaflet/chunk peta gagal, skrip ini tetap jalan (tombol Start, fetch API). */
type DashboardMapModule = typeof import("../lib/dashboardMap");
let mapModPromise: Promise<DashboardMapModule> | null = null;

function getMapMod() {
  if (!mapModPromise) mapModPromise = import("../lib/dashboardMap");
  return mapModPromise;
}

async function safeIdleMap() {
  try {
    (await getMapMod()).showIdlePreviewMap();
  } catch (e) {
    console.warn("Peta idle:", e);
  }
}

const LS = {
  soundOn: "msp_sound_on",
  soundPreset: "msp_sound_preset",
  sessions: "msp_sessions_v1",
} as const;

function readSoundOn(): boolean {
  const v = localStorage.getItem(LS.soundOn);
  if (v === null) return true;
  return v !== "0";
}

function readSoundPreset(): string {
  return localStorage.getItem(LS.soundPreset) || "bell";
}

function getSessions(): SessionRow[] {
  try {
    const raw = localStorage.getItem(LS.sessions);
    if (!raw) return [];
    const a = JSON.parse(raw) as unknown;
    return Array.isArray(a) ? (a as SessionRow[]) : [];
  } catch {
    return [];
  }
}

function setSessions(arr: SessionRow[]) {
  localStorage.setItem(LS.sessions, JSON.stringify(arr.slice(0, 50)));
}

/** Web Audio — dipicu setelah scraping selesai (dan preview saat ganti preset). */
function playCompletionSound() {
  if (!readSoundOn()) return;
  const preset = readSoundPreset();
  const w = window as Window &
    typeof globalThis & { webkitAudioContext?: typeof AudioContext };
  const Ctx = window.AudioContext || w.webkitAudioContext;
  if (!Ctx) return;
  try {
    const ctx = new Ctx();
    const master = ctx.createGain();
    master.connect(ctx.destination);
    master.gain.value = 0.14;

    const beep = (
      freq: number,
      t0: number,
      dur: number,
      type: OscillatorType = "sine",
    ) => {
      const o = ctx.createOscillator();
      const g = ctx.createGain();
      o.type = type;
      o.frequency.setValueAtTime(freq, t0);
      g.gain.setValueAtTime(0.0001, t0);
      g.gain.exponentialRampToValueAtTime(1, t0 + 0.01);
      g.gain.exponentialRampToValueAtTime(0.0001, t0 + dur);
      o.connect(g);
      g.connect(master);
      o.start(t0);
      o.stop(t0 + dur + 0.05);
    };

    const now = ctx.currentTime;
    if (preset === "chime") {
      beep(440, now, 0.1);
      beep(554.37, now + 0.12, 0.1);
      beep(659.25, now + 0.24, 0.14);
    } else if (preset === "notification") {
      beep(880, now, 0.06);
      beep(1174.66, now + 0.08, 0.08);
    } else if (preset === "success") {
      beep(392, now, 0.1);
      beep(523.25, now + 0.11, 0.1);
      beep(659.25, now + 0.22, 0.18);
    } else if (preset === "alert") {
      beep(220, now, 0.14);
      beep(220, now + 0.22, 0.14);
    } else {
      beep(523.25, now, 0.12);
      beep(659.25, now + 0.18, 0.16);
    }

    window.setTimeout(() => {
      try {
        void ctx.close();
      } catch {
        /* ignore */
      }
    }, 1400);
  } catch {
    /* ignore */
  }
}

function formatCount(n: unknown): string {
  if (n == null || typeof n !== "number" || !Number.isFinite(n)) return "—";
  return n.toLocaleString("id-ID");
}

function setResultsTable(stores: ScrapeStoreRow[]) {
  const tbody = document.getElementById("results-tbody");
  const heading = document.getElementById("results-heading");
  const pag = document.getElementById("results-pagination-text");
  const totalMap = document.getElementById("dash-total-scraped");
  const totalFooter = document.getElementById("dash-footer-total");
  if (!tbody) return;
  const n = Array.isArray(stores) ? stores.length : 0;
  if (heading) heading.textContent = `Results (${n})`;
  if (pag) {
    pag.textContent = n
      ? `Menampilkan 1–${n} dari ${n}`
      : "Belum ada data — jalankan scraping dari sidebar";
  }
  const formatted = formatCount(n);
  if (totalMap) totalMap.textContent = formatted;
  if (totalFooter) totalFooter.textContent = formatted;
  if (!n) {
    tbody.innerHTML =
      '<tr><td colspan="4" class="px-4 py-10 text-center text-sm text-gray-500 leading-relaxed">Belum ada hasil. Isi <strong class="text-gray-400">keyword</strong>, <strong class="text-gray-400">lokasi</strong>, dan <strong class="text-gray-400">target</strong>, lalu klik <span class="text-[#0066cc] font-medium">Start Scraping</span>.<br/><span class="text-xs mt-2 block text-gray-600">Backend: <code class="text-gray-500">cd be && go run . server</code></span></td></tr>';
    return;
  }
  tbody.innerHTML = "";
  stores.forEach((s, i) => {
    const name = (s.name || "").trim() || "—";
    const addr = (s.address || "").trim() || "—";
    const phone = (s.phone || "").trim() || "—";
    const sub = addr.length > 48 ? addr.slice(0, 48) + "…" : addr;
    const tr = document.createElement("tr");
    tr.className =
      "hover:bg-[#1f1f1f] group transition-colors cursor-pointer result-row" +
      (i === 0 ? " bg-[#0066cc]/5 border-l-2 border-[#0066cc]" : "");
    tr.dataset.name = name;
    tr.dataset.sub = sub;
    tr.dataset.address = addr;
    tr.dataset.phone = phone;
    tr.innerHTML = `
				<td class="px-4 py-3 text-sm">
					<span class="font-medium block">${escapeHtml(name)}</span>
					<span class="text-xs text-gray-500">${escapeHtml(sub)}</span>
				</td>
				<td class="px-4 py-3">
					<span class="text-sm font-mono text-gray-600">—</span>
				</td>
				<td class="px-4 py-3 text-xs text-gray-400 font-mono">${escapeHtml(phone)}</td>
				<td class="px-4 py-3 text-right">
					<button type="button" class="opacity-0 group-hover:opacity-100 p-1 text-gray-400 hover:text-[#0066cc] transition-all" aria-label="Detail">
						<iconify-icon icon="lucide:external-link"></iconify-icon>
					</button>
				</td>`;
    tbody.appendChild(tr);
  });
}

function escapeHtml(s: string) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function persistSoundFromUI() {
  const toggle = document.getElementById(
    "notification-sound-toggle",
  ) as HTMLInputElement | null;
  const sel = document.getElementById(
    "notification-sound-select",
  ) as HTMLSelectElement | null;
  const state = document.getElementById("notification-sound-state");
  if (toggle) {
    localStorage.setItem(LS.soundOn, toggle.checked ? "1" : "0");
    if (state) state.textContent = toggle.checked ? "On" : "Off";
  }
  if (sel) localStorage.setItem(LS.soundPreset, sel.value);
}

function loadSoundIntoUI() {
  const toggle = document.getElementById(
    "notification-sound-toggle",
  ) as HTMLInputElement | null;
  const sel = document.getElementById(
    "notification-sound-select",
  ) as HTMLSelectElement | null;
  const state = document.getElementById("notification-sound-state");
  if (toggle) {
    toggle.checked = readSoundOn();
    if (state) state.textContent = toggle.checked ? "On" : "Off";
  }
  if (sel) {
    const p = readSoundPreset();
    sel.value = ["bell", "chime", "notification", "success", "alert"].includes(
      p,
    )
      ? p
      : "bell";
  }
}

function renderSessionsTable() {
  const tbody = document.getElementById("settings-sessions-tbody");
  const selAll = document.getElementById(
    "settings-sessions-select-all",
  ) as HTMLInputElement | null;
  if (!tbody) return;
  if (selAll) selAll.checked = false;
  const rows = getSessions();
  if (!rows.length) {
    tbody.innerHTML =
      '<tr><td colspan="4" class="px-4 py-8 text-center text-sm text-gray-500">Belum ada riwayat scraping di browser ini.</td></tr>';
    return;
  }
  tbody.innerHTML = "";
  rows.forEach((s, i) => {
    const tr = document.createElement("tr");
    tr.className = "hover:bg-[#252525] transition-colors group cursor-pointer";
    const dt = s.at
      ? new Date(s.at).toLocaleString("id-ID", {
          dateStyle: "short",
          timeStyle: "short",
        })
      : "—";
    const badge =
      s.status === "completed"
        ? '<span class="text-[9px] text-green-500 font-bold bg-green-500/10 px-1.5 py-0.5 rounded uppercase">Completed</span>'
        : '<span class="text-[9px] text-gray-500 font-bold bg-gray-500/10 px-1.5 py-0.5 rounded uppercase">Archived</span>';
    tr.innerHTML = `
				<td class="px-4 py-4">
					<input type="checkbox" class="session-row-cb rounded border-[#2d2d2d] bg-[#1a1a1a] text-[#0066cc] focus:ring-0" data-session-index="${i}" />
				</td>
				<td class="px-4 py-4">
					<p class="text-xs font-bold text-white">${escapeHtml(dt)}</p>
					${badge}
					<p class="text-[10px] text-gray-500 mt-1 truncate max-w-[160px]">${escapeHtml(s.keyword || "")}</p>
				</td>
				<td class="px-4 py-4 text-xs text-gray-400">${escapeHtml(s.location || "—")}</td>
				<td class="px-4 py-4 text-right font-mono text-xs text-white font-bold">${typeof s.results === "number" ? s.results : "—"}</td>`;
    tbody.appendChild(tr);
  });
}

function appendSession(entry: SessionRow) {
  const arr = getSessions();
  arr.unshift(entry);
  setSessions(arr);
  renderSessionsTable();
}

const notificationEntries: NotificationEntry[] = [
  {
    id: "seed-welcome",
    title: "Selamat datang",
    body: "Isi keyword, lokasi, dan target di sidebar lalu klik Start Scraping. Progres tampil di Live tracking dan log.",
    at: Date.now(),
    kind: "info",
  },
];

function notificationKindBorder(kind: NotificationKind | string) {
  if (kind === "success") return "border-l-[#22c55e]";
  if (kind === "error") return "border-l-red-500";
  return "border-l-[#0066cc]";
}

function renderNotificationList() {
  const list = document.getElementById("notification-list");
  if (!list) return;
  if (!notificationEntries.length) {
    list.innerHTML =
      '<p class="text-xs text-gray-500 text-center py-8 px-4 m-0">Belum ada notifikasi.</p>';
    return;
  }
  list.innerHTML = "";
  notificationEntries.forEach((n) => {
    const time = new Date(n.at).toLocaleString("id-ID", {
      dateStyle: "short",
      timeStyle: "short",
    });
    const border = notificationKindBorder(n.kind);
    const el = document.createElement("div");
    el.className = `rounded-lg border border-[#2d2d2d] border-l-4 ${border} bg-[#1f1f1f] p-3 mb-2 last:mb-0`;
    el.innerHTML = `
				<p class="text-xs font-bold text-white m-0 mb-1">${escapeHtml(n.title)}</p>
				<p class="text-[11px] text-gray-400 m-0 leading-relaxed">${escapeHtml(n.body)}</p>
				<p class="text-[10px] text-gray-600 m-0 mt-2 font-mono">${escapeHtml(time)}</p>`;
    list.appendChild(el);
  });
}

function setNotificationPanelOpen(open: boolean) {
  const panel = document.getElementById("notification-panel");
  const btn = document.getElementById("notification-trigger");
  const dot = document.getElementById("notification-dot");
  if (!panel || !btn) return;
  if (open) {
    panel.classList.remove("hidden");
    panel.classList.add("flex", "flex-col");
    btn.setAttribute("aria-expanded", "true");
    if (dot) dot.classList.add("hidden");
  } else {
    panel.classList.add("hidden");
    panel.classList.remove("flex", "flex-col");
    btn.setAttribute("aria-expanded", "false");
  }
}

function isNotificationPanelOpen() {
  const panel = document.getElementById("notification-panel");
  return !!(panel && !panel.classList.contains("hidden"));
}

function appendDashboardNotification(
  title: string,
  body: string,
  kind: NotificationKind = "info",
) {
  notificationEntries.unshift({
    id: `n-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    title,
    body,
    at: Date.now(),
    kind,
  });
  while (notificationEntries.length > 40) notificationEntries.pop();
  renderNotificationList();
  if (!isNotificationPanelOpen()) {
    document.getElementById("notification-dot")?.classList.remove("hidden");
  }
}

function wireNotifications() {
  const trigger = document.getElementById("notification-trigger");
  const clearBtn = document.getElementById("notification-clear-all");

  trigger?.addEventListener("click", (e) => {
    e.stopPropagation();
    setNotificationPanelOpen(!isNotificationPanelOpen());
  });

  clearBtn?.addEventListener("click", (e) => {
    e.stopPropagation();
    notificationEntries.length = 0;
    renderNotificationList();
  });

  document.addEventListener("mousedown", (e) => {
    const anchor = document.getElementById("notification-anchor");
    if (!anchor || !isNotificationPanelOpen()) return;
    const t = e.target;
    if (t instanceof Node && anchor.contains(t)) return;
    setNotificationPanelOpen(false);
  });
}

function setLog(text: string) {
  const el = document.getElementById("scrape-log");
  if (el) {
    el.textContent = text;
    el.scrollTop = el.scrollHeight;
  }
}

/** phase: idle | running | done — mengikuti data progress dari GET /api/scrape/status */
function updateLiveProgress(
  saved: unknown,
  target: unknown,
  phase: "idle" | "running" | "done",
) {
  const fill = document.getElementById("live-progress-fill");
  const label = document.getElementById("live-progress-label");
  const dot = document.getElementById("live-progress-dot");
  if (!fill || !label) return;
  const t = Math.max(0, Number(target) || 0);
  const s = Math.max(0, Number(saved) || 0);
  let pct = 0;
  if (t > 0) pct = Math.min(100, (s / t) * 100);
  fill.style.width = `${pct}%`;
  label.textContent = t > 0 ? `${s} / ${t}` : "—";
  if (dot) {
    dot.className = "w-2 h-2 rounded-full shrink-0 ";
    if (phase === "running") dot.className += "bg-green-500 animate-pulse";
    else if (phase === "done") dot.className += "bg-green-500";
    else dot.className += "bg-gray-500";
  }
}

function setScrapeBusy(busy: boolean) {
  const btn = document.getElementById(
    "start-scrape-btn",
  ) as HTMLButtonElement | null;
  if (!btn) return;
  btn.disabled = busy;
  btn.classList.toggle("opacity-60", busy);
  btn.classList.toggle("cursor-not-allowed", busy);
}

function setViewMode(mode: "map" | "list") {
  const mapStage = document.getElementById("map-stage");
  const results = document.getElementById("results-panel");
  const split = document.getElementById("dashboard-split");
  const btnMap = document.getElementById("map-view-toggle");
  const btnList = document.getElementById("list-view-toggle");
  if (!mapStage || !results || !split || !btnMap || !btnList) return;

  const active = "bg-[#0066cc] text-white";
  const inactive = "text-gray-500 hover:text-white bg-transparent";

  if (mode === "list") {
    mapStage.classList.add("hidden");
    results.classList.remove("lg:w-[500px]");
    results.classList.add("lg:flex-1");
    split.classList.remove("lg:flex-row");
    split.classList.add("lg:flex-col");
    btnList.className =
      "inline-flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-bold transition-all " +
      active;
    btnMap.className =
      "inline-flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-bold transition-all " +
      inactive;
  } else {
    mapStage.classList.remove("hidden");
    results.classList.add("lg:w-[500px]");
    results.classList.remove("lg:flex-1");
    split.classList.add("lg:flex-row");
    split.classList.remove("lg:flex-col");
    btnMap.className =
      "inline-flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-bold transition-all " +
      active;
    btnList.className =
      "inline-flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-bold transition-all " +
      inactive;
    void getMapMod()
      .then((m) => m.invalidateMapSize())
      .catch(() => {});
  }
}

function collectRows(): { name: string; address: string; phone: string }[] {
  return Array.from(document.querySelectorAll("tr.result-row")).map((row) => ({
    name: row.getAttribute("data-name") || "",
    address:
      row.getAttribute("data-address") || row.getAttribute("data-sub") || "",
    phone: row.getAttribute("data-phone") || "",
  }));
}

function downloadText(filename: string, text: string, mime: string) {
  const blob = new Blob([text], { type: mime });
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
  URL.revokeObjectURL(a.href);
}

function wireExports() {
  document.getElementById("export-csv-btn")?.addEventListener("click", () => {
    const rows = collectRows();
    if (!rows.length) {
      window.alert("Belum ada baris untuk diekspor.");
      return;
    }
    const esc = (s: string) => '"' + String(s).replace(/"/g, '""') + '"';
    const lines = [
      ["name", "address", "phone"].join(","),
      ...rows.map((r) => [esc(r.name), esc(r.address), esc(r.phone)].join(",")),
    ];
    downloadText(
      "maps-scraper-export.csv",
      lines.join("\n"),
      "text/csv;charset=utf-8",
    );
  });
  document.getElementById("export-json-btn")?.addEventListener("click", () => {
    const rows = collectRows();
    if (!rows.length) {
      window.alert("Belum ada baris untuk diekspor.");
      return;
    }
    downloadText(
      "maps-scraper-export.json",
      JSON.stringify(rows, null, 2),
      "application/json;charset=utf-8",
    );
  });
  document.getElementById("export-xlsx-btn")?.addEventListener("click", () => {
    window.alert(
      "Export Excel membutuhkan library tambahan (mis. SheetJS). Gunakan CSV atau JSON untuk saat ini.",
    );
  });
}

function wireRadius() {
  const r = document.getElementById("input-radius") as HTMLInputElement | null;
  const label = document.getElementById("radius-label");
  r?.addEventListener("input", () => {
    if (label) label.textContent = r.value;
  });
}

function wireModalsAndSettings() {
  /** Capture + composedPath: klik pada `<iconify-icon>` (shadow DOM) tetap terdeteksi. */
  document.addEventListener(
    "click",
    (e) => {
      for (const node of e.composedPath()) {
        if (!(node instanceof Element)) continue;
        if (node.id === "download-app-trigger") {
          openModal("download-modal");
          return;
        }
        if (node.id === "settings-modal-trigger") {
          loadSoundIntoUI();
          renderSessionsTable();
          openModal("settings-modal");
          return;
        }
        if (node.id === "social-modal-trigger") {
          openModal("social-modal");
          return;
        }
      }
    },
    true,
  );

  document.querySelectorAll("[data-modal-close]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const id = btn.getAttribute("data-modal-close");
      if (id) closeModal(id);
    });
  });
  document.querySelectorAll("[data-modal-backdrop]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const id = btn.getAttribute("data-modal-backdrop");
      if (id) closeModal(id);
    });
  });

  document
    .getElementById("settings-save-btn")
    ?.addEventListener("click", () => {
      persistSoundFromUI();
      closeModal("settings-modal");
    });

  document
    .getElementById("notification-sound-toggle")
    ?.addEventListener("change", persistSoundFromUI);
  document
    .getElementById("notification-sound-select")
    ?.addEventListener("change", () => {
      persistSoundFromUI();
      playCompletionSound();
    });

  document.querySelectorAll(".download-platform-btn").forEach((b) => {
    b.addEventListener("click", () => {
      window.alert(
        "Paket desktop belum dipublikasikan. Gunakan web dashboard dan backend untuk saat ini.",
      );
    });
  });

  document
    .getElementById("all-releases-link")
    ?.addEventListener("click", (e) => {
      e.preventDefault();
      window.alert(
        "Halaman rilis akan ditautkan ketika build desktop tersedia.",
      );
    });

  document
    .getElementById("settings-sessions-select-all")
    ?.addEventListener("change", (e) => {
      const t = e.target;
      if (!(t instanceof HTMLInputElement)) return;
      document.querySelectorAll(".session-row-cb").forEach((cb) => {
        if (cb instanceof HTMLInputElement) cb.checked = t.checked;
      });
    });

  document
    .getElementById("settings-sessions-delete")
    ?.addEventListener("click", () => {
      const idxs = Array.from(
        document.querySelectorAll(".session-row-cb:checked"),
      )
        .map((cb) =>
          parseInt(
            cb instanceof HTMLInputElement
              ? cb.getAttribute("data-session-index") || "-1"
              : "-1",
            10,
          ),
        )
        .filter((i) => i >= 0)
        .sort((a, b) => b - a);
      if (!idxs.length) {
        window.alert("Pilih satu atau lebih baris.");
        return;
      }
      const arr = getSessions();
      idxs.forEach((i) => {
        if (i >= 0 && i < arr.length) arr.splice(i, 1);
      });
      setSessions(arr);
      renderSessionsTable();
    });

  document
    .getElementById("settings-sessions-clear")
    ?.addEventListener("click", () => {
      if (!getSessions().length) return;
      if (!window.confirm("Hapus semua riwayat sesi di browser ini?")) return;
      localStorage.removeItem(LS.sessions);
      renderSessionsTable();
    });

  document
    .getElementById("settings-sessions-restore")
    ?.addEventListener("click", () => {
      window.alert(
        "Riwayat hanya menyimpan ringkasan. Isi kembali keyword dan lokasi di sidebar, lalu jalankan scraping untuk memuat ulang data hasil.",
      );
    });

  window.addEventListener("keydown", (e) => {
    if (e.key !== "Escape") return;
    setNotificationPanelOpen(false);
    closeModal("settings-modal");
    closeModal("social-modal");
    closeModal("download-modal");
  });
}

function errMsg(v: unknown, fallback: string): string {
  if (v && typeof v === "object" && "error" in v) {
    const e = (v as { error?: unknown }).error;
    if (typeof e === "string") return e;
  }
  return fallback;
}

function createStartScrapeHandler(apiBase: string) {
  return async function startScrapeFromUI() {
    const kwEl = document.getElementById(
      "input-keyword",
    ) as HTMLInputElement | null;
    const locEl = document.getElementById(
      "input-location",
    ) as HTMLInputElement | null;
    const maxEl = document.getElementById(
      "input-max-results",
    ) as HTMLInputElement | null;
    const kw = kwEl?.value?.trim() || "";
    const loc = locEl?.value?.trim() || "";
    const maxRaw = maxEl?.value;
    let maxResults = parseInt(String(maxRaw), 10);
    if (!Number.isFinite(maxResults) || maxResults < 1) maxResults = 10;
    if (maxResults > 500) maxResults = 500;

    if (!kw || !loc) {
      window.alert("Isi keyword bisnis dan lokasi terlebih dahulu.");
      return;
    }

    setLog("Mengirim permintaan ke backend…");
    updateLiveProgress(0, maxResults, "running");
    setScrapeBusy(true);

    let res: Response;
    try {
      res = await fetch(`${apiBase}/api/scrape`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ keyword: kw, location: loc, maxResults }),
      });
    } catch {
      setScrapeBusy(false);
      updateLiveProgress(0, 0, "idle");
      void safeIdleMap();
      setLog(
        "Gagal menghubungi backend. Pastikan server jalan: go run . server",
      );
      appendDashboardNotification(
        "Backend tidak terjangkau",
        "Tidak bisa menghubungi API. Pastikan server jalan dan PUBLIC_API_URL benar.",
        "error",
      );
      window.alert(
        "Tidak bisa terhubung ke API. Cek URL di .env (PUBLIC_API_URL) dan backend.",
      );
      return;
    }

    let data: Record<string, unknown> = {};
    try {
      data = (await res.json()) as Record<string, unknown>;
    } catch {
      data = {};
    }

    if (res.status === 409) {
      setScrapeBusy(false);
      updateLiveProgress(0, 0, "idle");
      void safeIdleMap();
      setLog(errMsg(data, "Scraping lain masih berjalan."));
      appendDashboardNotification(
        "Scraping bentrok",
        errMsg(data, "Job lain masih berjalan di server."),
        "error",
      );
      window.alert(errMsg(data, "Scraping masih berjalan di server."));
      return;
    }
    if (!res.ok) {
      setScrapeBusy(false);
      updateLiveProgress(0, 0, "idle");
      void safeIdleMap();
      setLog(errMsg(data, `Error ${res.status}`));
      appendDashboardNotification(
        "Permintaan ditolak",
        errMsg(data, `HTTP ${res.status}`),
        "error",
      );
      window.alert(errMsg(data, `Permintaan ditolak (${res.status})`));
      return;
    }

    const jobId = typeof data.jobId === "string" ? data.jobId : "";
    if (!jobId) {
      setScrapeBusy(false);
      updateLiveProgress(0, 0, "idle");
      void safeIdleMap();
      setLog("Respons tidak berisi jobId.");
      appendDashboardNotification(
        "Respons tidak valid",
        "Server tidak mengembalikan jobId.",
        "error",
      );
      return;
    }

    setViewMode("map");
    try {
      const m = await getMapMod();
      const mc =
        data.mapCenter && typeof data.mapCenter === "object"
          ? (data.mapCenter as MapCenter)
          : null;
      m.focusSearchOnMap(mc);
      m.invalidateMapSize();
    } catch (e) {
      console.warn("Peta fokus:", e);
    }

    setLog(
      `Job dimulai (${jobId}). Browser headless di server — peta sudah diarahkan ke lokasi; pantau Live tracking + log.`,
    );
    const jobShort = jobId.length > 12 ? `${jobId.slice(0, 10)}…` : jobId;
    appendDashboardNotification(
      "Scraping dimulai",
      `“${kw}” di ${loc} · target ${maxResults} · job ${jobShort}`,
      "info",
    );

    let pollTimer: ReturnType<typeof setInterval>;
    const tick = async () => {
      let st: Response;
      try {
        st = await fetch(
          `${apiBase}/api/scrape/status?jobId=${encodeURIComponent(jobId)}`,
        );
      } catch {
        clearInterval(pollTimer);
        setScrapeBusy(false);
        updateLiveProgress(0, 0, "idle");
        void safeIdleMap();
        setLog(
          "Koneksi ke /api/scrape/status putus. Cek backend dan jaringan.",
        );
        appendDashboardNotification(
          "Koneksi putus",
          "Tidak bisa mengambil status job. Cek backend dan jaringan.",
          "error",
        );
        return;
      }
      const j = (await st.json().catch(() => ({}))) as ScrapeStatusJson;
      if (!st.ok) {
        clearInterval(pollTimer);
        setScrapeBusy(false);
        updateLiveProgress(0, 0, "idle");
        void safeIdleMap();
        setLog(j.error || "Status job gagal diambil.");
        appendDashboardNotification(
          "Status job gagal",
          j.error || `HTTP ${st.status}`,
          "error",
        );
        return;
      }
      if (Array.isArray(j.logs)) setLog(j.logs.join("\n"));

      const pr = j.progress;
      if (pr && typeof pr === "object" && j.status === "running") {
        updateLiveProgress(pr.saved, pr.target, "running");
      }

      if (j.status === "running") return;

      clearInterval(pollTimer);
      setScrapeBusy(false);

      if (j.status === "error") {
        updateLiveProgress(0, 0, "idle");
        void safeIdleMap();
        appendDashboardNotification(
          "Scraping gagal",
          j.error || "Job berakhir dengan error.",
          "error",
        );
        window.alert(j.error || "Scraping gagal");
        return;
      }
      if (j.status === "done" && Array.isArray(j.stores)) {
        const stores = j.stores;
        const tgt =
          pr && typeof pr.target === "number" && pr.target > 0
            ? pr.target
            : stores.length;
        updateLiveProgress(stores.length, tgt, "done");
        setResultsTable(stores);
        try {
          const m = await getMapMod();
          m.plotStoresOnMap(stores, j.mapCenter ?? null);
          m.invalidateMapSize();
        } catch (e) {
          console.warn("Peta hasil:", e);
        }
        setLog((j.logs || []).join("\n") + "\n\nSelesai.");
        appendSession({
          at: new Date().toISOString(),
          keyword: kw,
          location: loc,
          results: stores.length,
          jobId,
          status: "completed",
        });
        playCompletionSound();
        appendDashboardNotification(
          "Scraping selesai",
          `${stores.length} listing tersimpan untuk “${kw}” di ${loc}.`,
          "success",
        );
        return;
      }
      updateLiveProgress(0, 0, "idle");
      void safeIdleMap();
      setResultsTable([]);
      appendDashboardNotification(
        "Job selesai tanpa data",
        "Status bukan error atau done dengan daftar toko kosong.",
        "error",
      );
    };

    pollTimer = setInterval(() => void tick(), 1200);
    await tick();
  };
}

export function initScrapMapsDashboard(
  apiBase: string = scrapMapsApiBase,
): void {
  document.getElementById("start-scrape-btn")?.addEventListener("click", () => {
    void createStartScrapeHandler(apiBase)();
  });

  document
    .getElementById("map-view-toggle")
    ?.addEventListener("click", () => setViewMode("map"));
  document
    .getElementById("list-view-toggle")
    ?.addEventListener("click", () => setViewMode("list"));

  setResultsTable([]);
  updateLiveProgress(0, 0, "idle");
  loadSoundIntoUI();
  renderSessionsTable();
  renderNotificationList();
  wireNotifications();
  wireModalsAndSettings();
  wireExports();
  wireRadius();
  void (async () => {
    try {
      const m = await getMapMod();
      m.ensureMap();
      m.showIdlePreviewMap();
    } catch (e) {
      console.error("Peta preview gagal dimuat:", e);
    }
  })();
}
