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
