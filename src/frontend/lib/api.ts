const BACKEND_URL = process.env.BACKEND_URL || "http://localhost:8080";

export interface Transaction {
  id: string;
  stock_id: string;
  ticker: string;
  stock_name: string;
  transaction_type: "buy" | "sell";
  shares: number;
  price_per_share: number;
  transaction_date: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTransactionInput {
  ticker: string;
  name?: string;
  transaction_type: "buy" | "sell";
  shares: number;
  price_per_share: number;
  transaction_date: string;
}

export interface UpdateTransactionInput {
  transaction_type: "buy" | "sell";
  shares: number;
  price_per_share: number;
  transaction_date: string;
}

export interface Holding {
  ticker: string;
  name: string;
  total_shares: number;
  avg_cost: number;
  current_price: number;
  market_value: number;
  gain_loss: number;
  gain_loss_pct: number;
}

export interface Portfolio {
  holdings: Holding[];
  total_value: number;
  total_cost: number;
  total_gain_loss: number;
}

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BACKEND_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
    cache: "no-store",
  });
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(error.error || res.statusText);
  }
  return res.json();
}

export interface HealthStatus {
  status: string;
  service: string;
  db_target: string;
}

export async function getHealth(): Promise<HealthStatus> {
  return apiFetch<HealthStatus>("/health");
}

export async function getPortfolio(): Promise<Portfolio> {
  return apiFetch<Portfolio>("/api/portfolio");
}

export async function getTransactions(ticker?: string): Promise<Transaction[]> {
  const query = ticker ? `?ticker=${ticker}` : "";
  return apiFetch<Transaction[]>(`/api/transactions${query}`);
}

export async function getTransaction(id: string): Promise<Transaction> {
  return apiFetch<Transaction>(`/api/transactions/${id}`);
}

export async function createTransaction(
  input: CreateTransactionInput,
): Promise<Transaction> {
  return apiFetch<Transaction>("/api/transactions", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateTransaction(
  id: string,
  input: UpdateTransactionInput,
): Promise<Transaction> {
  return apiFetch<Transaction>(`/api/transactions/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function deleteTransaction(id: string): Promise<void> {
  await fetch(`${BACKEND_URL}/api/transactions/${id}`, {
    method: "DELETE",
  });
}

export interface HistoricalPricePoint {
  date: string;
  open: number | null;
  high: number | null;
  low: number | null;
  close: number | null;
  adj_close: number | null;
  volume: number | null;
}

export interface HistoricalPriceResponse {
  ticker: string;
  currency: string;
  interval: string;
  prices: HistoricalPricePoint[];
}

export interface ImportResult {
  created: number;
  updated: number;
  total: number;
}

export async function importAlpaca(): Promise<ImportResult> {
  return apiFetch<ImportResult>("/api/import/alpaca", { method: "POST" });
}

export interface StagingRow {
  id: string;
  import_id: string;
  trade_date: string | null;
  symbol: string | null;
  side: string | null;
  quantity: number | null;
  price_per_share: number | null;
  currency: string;
  fees: number;
  account: string | null;
  source_row: string | null;
  warnings: string[];
  status: string;
}

export interface ImportDetail {
  import: {
    id: string;
    filename: string;
    file_type: string;
    status: string;
  };
  rows: StagingRow[];
}

export interface ConfirmResult {
  inserted: number;
  duplicates: number;
}

export async function uploadFile(file: File): Promise<ImportDetail> {
  const formData = new FormData();
  formData.append("file", file);

  const res = await fetch(`${BACKEND_URL}/api/imports/upload`, {
    method: "POST",
    body: formData,
    // Do NOT set Content-Type — browser sets it with boundary for multipart
  });
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(error.error || res.statusText);
  }
  return res.json();
}

export async function getImport(importId: string): Promise<ImportDetail> {
  return apiFetch<ImportDetail>(`/api/imports/${importId}`);
}

export async function patchStagingRow(
  importId: string,
  rowId: string,
  updates: Partial<
    Pick<
      StagingRow,
      "trade_date" | "symbol" | "side" | "quantity" | "price_per_share"
    >
  >,
): Promise<void> {
  const res = await fetch(
    `${BACKEND_URL}/api/imports/${importId}/rows/${rowId}`,
    {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(updates),
    },
  );
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(error.error || res.statusText);
  }
}

export async function confirmImport(importId: string): Promise<ConfirmResult> {
  return apiFetch<ConfirmResult>(`/api/imports/${importId}/confirm`, {
    method: "POST",
  });
}

export async function getHistoricalPrices(
  ticker: string,
  start: string,
  end: string,
): Promise<HistoricalPriceResponse> {
  const baseUrl =
    typeof window === "undefined"
      ? BACKEND_URL
      : process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
  const res = await fetch(
    `${baseUrl}/api/prices/${ticker}/history?start=${start}&end=${end}`,
    { cache: "no-store" },
  );
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(error.error || res.statusText);
  }
  return res.json();
}
