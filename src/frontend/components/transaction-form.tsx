"use client";

import { Transaction } from "@/lib/api";

interface Props {
  action: (formData: FormData) => Promise<void>;
  transaction?: Transaction;
  showTickerField?: boolean;
}

export function TransactionForm({ action, transaction, showTickerField = true }: Props) {
  const today = new Date().toISOString().split("T")[0];

  return (
    <form action={action} className="space-y-4 max-w-md">
      {showTickerField && (
        <>
          <div>
            <label htmlFor="ticker" className="block text-sm font-medium text-gray-700 mb-1">
              Ticker Symbol
            </label>
            <input
              type="text"
              id="ticker"
              name="ticker"
              required
              placeholder="AAPL"
              defaultValue={transaction?.ticker}
              className="w-full border rounded-md px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-1">
              Company Name (optional)
            </label>
            <input
              type="text"
              id="name"
              name="name"
              placeholder="Apple Inc."
              defaultValue={transaction?.stock_name}
              className="w-full border rounded-md px-3 py-2 text-sm"
            />
          </div>
        </>
      )}

      <div>
        <label htmlFor="transaction_type" className="block text-sm font-medium text-gray-700 mb-1">
          Type
        </label>
        <select
          id="transaction_type"
          name="transaction_type"
          defaultValue={transaction?.transaction_type || "buy"}
          className="w-full border rounded-md px-3 py-2 text-sm"
        >
          <option value="buy">Buy</option>
          <option value="sell">Sell</option>
        </select>
      </div>

      <div>
        <label htmlFor="shares" className="block text-sm font-medium text-gray-700 mb-1">
          Shares
        </label>
        <input
          type="number"
          id="shares"
          name="shares"
          required
          step="0.0001"
          min="0.0001"
          defaultValue={transaction?.shares}
          className="w-full border rounded-md px-3 py-2 text-sm"
        />
      </div>

      <div>
        <label htmlFor="price_per_share" className="block text-sm font-medium text-gray-700 mb-1">
          Price per Share
        </label>
        <input
          type="number"
          id="price_per_share"
          name="price_per_share"
          required
          step="0.01"
          min="0.01"
          defaultValue={transaction?.price_per_share}
          className="w-full border rounded-md px-3 py-2 text-sm"
        />
      </div>

      <div>
        <label htmlFor="transaction_date" className="block text-sm font-medium text-gray-700 mb-1">
          Date
        </label>
        <input
          type="date"
          id="transaction_date"
          name="transaction_date"
          required
          defaultValue={transaction?.transaction_date || today}
          className="w-full border rounded-md px-3 py-2 text-sm"
        />
      </div>

      <button
        type="submit"
        className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700"
      >
        {transaction ? "Update Transaction" : "Add Transaction"}
      </button>
    </form>
  );
}
