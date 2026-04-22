"use client";

import { use, useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import ReviewTable from "@/components/review-table";
import { confirmImportAction } from "@/app/actions/import";
import type { StagingRow } from "@/lib/api";

export default function ImportReviewPage({
  params,
}: {
  params: Promise<{ importId: string }>;
}) {
  const { importId } = use(params);
  const router = useRouter();
  const [rows, setRows] = useState<StagingRow[]>([]);
  const [filename, setFilename] = useState("");
  const [loading, setLoading] = useState(true);
  const [confirming, setConfirming] = useState(false);
  const [confirmResult, setConfirmResult] = useState<{
    inserted: number;
    duplicates: number;
  } | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const baseUrl =
      process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
    fetch(`${baseUrl}/api/imports/${importId}`, { cache: "no-store" })
      .then((res) => {
        if (!res.ok) throw new Error("Failed to load import");
        return res.json();
      })
      .then((detail) => {
        setRows(detail.rows);
        setFilename(detail.import.filename);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [importId]);

  const handleRowUpdated = useCallback(
    (rowId: string, updates: Partial<StagingRow>) => {
      setRows((prev) =>
        prev.map((r) => (r.id === rowId ? { ...r, ...updates } : r)),
      );
    },
    [],
  );

  async function handleConfirm() {
    setConfirming(true);
    setError(null);
    const result = await confirmImportAction(importId);
    if (result.success) {
      setConfirmResult({
        inserted: result.inserted ?? 0,
        duplicates: result.duplicates ?? 0,
      });
    } else {
      setError(result.error ?? "Confirm failed");
    }
    setConfirming(false);
  }

  if (loading) {
    return <p className="text-gray-500 py-8 text-center">Loading...</p>;
  }

  if (error && !rows.length) {
    return <p className="text-red-500 py-8 text-center">{error}</p>;
  }

  const readyCount = rows.filter((r) => r.status === "ready").length;
  const needsAttentionCount = rows.filter(
    (r) => r.status === "needs_attention",
  ).length;

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold">Review Import</h1>
          <p className="text-gray-500 text-sm mt-1">{filename}</p>
        </div>
        <div className="flex gap-3">
          <button
            onClick={() => router.push("/import")}
            className="px-4 py-2 text-sm text-gray-600 border rounded-md hover:bg-gray-50"
          >
            Cancel
          </button>
          {!confirmResult && (
            <button
              onClick={handleConfirm}
              disabled={confirming || readyCount === 0}
              className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {confirming
                ? "Importing..."
                : `Import ${readyCount} Transaction${readyCount !== 1 ? "s" : ""}`}
            </button>
          )}
        </div>
      </div>

      {needsAttentionCount > 0 && !confirmResult && (
        <div className="mb-4 p-3 rounded-md text-sm bg-amber-50 text-amber-800 border border-amber-200">
          {needsAttentionCount} row{needsAttentionCount !== 1 ? "s" : ""} need
          attention. Edit to resolve warnings before importing.
        </div>
      )}

      {confirmResult && (
        <div className="mb-4 p-4 rounded-md text-sm bg-green-50 text-green-800 border border-green-200">
          <p>
            Import complete: {confirmResult.inserted} inserted,{" "}
            {confirmResult.duplicates} duplicates skipped.
          </p>
          <button
            onClick={() => router.push("/transactions")}
            className="mt-2 text-blue-600 hover:underline text-sm"
          >
            View Transactions
          </button>
        </div>
      )}

      {error && (
        <div className="mb-4 p-4 rounded-md text-sm bg-red-50 text-red-800 border border-red-200">
          <p>{error}</p>
        </div>
      )}

      <ReviewTable
        importId={importId}
        rows={rows}
        onRowUpdated={handleRowUpdated}
      />
    </div>
  );
}
