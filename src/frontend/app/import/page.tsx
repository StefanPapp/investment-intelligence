"use client";

import { useState } from "react";
import { importAlpacaAction } from "@/app/actions/import";

export default function ImportPage() {
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{
    success: boolean;
    created?: number;
    updated?: number;
    total?: number;
    error?: string;
  } | null>(null);

  async function handleImport() {
    setLoading(true);
    setResult(null);
    const res = await importAlpacaAction();
    setResult(res);
    setLoading(false);
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Import Transactions</h1>

      <div className="bg-white rounded-lg shadow-sm border p-6">
        <h2 className="text-lg font-semibold mb-2">Alpaca</h2>
        <p className="text-gray-600 text-sm mb-4">
          Import your filled orders from Alpaca. Existing orders are matched by
          order ID and updated if changed.
        </p>

        <button
          onClick={handleImport}
          disabled={loading}
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? "Importing..." : "Import from Alpaca"}
        </button>

        {result && (
          <div
            className={`mt-4 p-4 rounded-md text-sm ${
              result.success
                ? "bg-green-50 text-green-800 border border-green-200"
                : "bg-red-50 text-red-800 border border-red-200"
            }`}
          >
            {result.success ? (
              <p>
                Imported {result.total} order{result.total !== 1 ? "s" : ""}
                {" \u2014 "}
                {result.created} created, {result.updated} updated.
              </p>
            ) : (
              <p>Import failed: {result.error}</p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
