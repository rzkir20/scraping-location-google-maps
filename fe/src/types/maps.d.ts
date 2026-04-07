type NotificationKind = "info" | "success" | "error";

type MapCenter = { lat: string; lng: string };

type Store = { name?: string; phone?: string; address?: string };

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
  rating?: string;
  address?: string;
  phone?: string;
}

interface LiveCardRow {
  name?: string;
  rating?: string;
  category?: string;
  address?: string;
  phone?: string;
  openingStatus?: string;
}

interface ScrapeStatusJson {
  status?: string;
  error?: string;
  logs?: string[];
  stores?: ScrapeStoreRow[];
  mapCenter?: MapCenter | null;
  progress?: { saved?: number; target?: number };
  currentCard?: LiveCardRow;
}
