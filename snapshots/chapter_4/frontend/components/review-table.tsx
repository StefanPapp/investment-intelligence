"use client";

import { useState } from "react";
import type { StagingRow } from "@/lib/api";
import { patchRowAction } from "@/app/actions/import";

interface ReviewTableProps {
  importId: string;
  rows: StagingRow[];
  onRowUpdated: (rowId: string, updates: Partial<StagingRow>) => void;
}

export default function ReviewTable({
  importId,
  rows,
  onRowUpdated,
}: ReviewTableProps) {
  const ready = rows.filter((r) => r.status === "ready");
  const needsAttention = rows.filter((r) => r.status === "needs_attention");
  const skipped = rows.filter((r) => r.status === "skipped");

  return (
    <div className="space-y-6">
      {needsAttention.length > 0 && (
        <RowGroup
          title="Needs Attention"
          rows={needsAttention}
          importId={importId}
          onRowUpdated={onRowUpdated}
          bgColor="bg-amber-50"
          borderColor="border-amber-200"
        />
      )}
      {ready.length > 0 && (
        <RowGroup
          title="Ready"
          rows={ready}
          importId={importId}
          onRowUpdated={onRowUpdated}
          bgColor="bg-green-50"
          borderColor="border-green-200"
        />
      )}
      {skipped.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-gray-500 mb-2">
            Skipped ({skipped.length})
          </h3>
          <div className="bg-gray-50 rounded-lg border border-gray-200 p-4">
            {skipped.map((row) => (
              <p key={row.id} className="text-xs text-gray-500">
                {row.source_row}
              </p>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

interface RowGroupProps {
  title: string;
  rows: StagingRow[];
  importId: string;
  onRowUpdated: (rowId: string, updates: Partial<StagingRow>) => void;
  bgColor: string;
  borderColor: string;
}

function RowGroup({
  title,
  rows,
  importId,
  onRowUpdated,
  bgColor,
  borderColor,
}: RowGroupProps) {
  return (
    <div>
      <h3 className="text-sm font-semibold text-gray-700 mb-2">
        {title} ({rows.length})
      </h3>
      <div
        className={`rounded-lg border ${borderColor} ${bgColor} overflow-hidden`}
      >
        <table className="w-full text-sm">
          <thead>
            <tr className={`border-b ${borderColor} text-left text-gray-500`}>
              <th className="py-2 px-3">Date</th>
              <th className="py-2 px-3">Symbol</th>
              <th className="py-2 px-3">Side</th>
              <th className="py-2 px-3 text-right">Qty</th>
              <th className="py-2 px-3 text-right">Price</th>
              <th className="py-2 px-3">Currency</th>
              <th className="py-2 px-3">Warnings</th>
              <th className="py-2 px-3"></th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <EditableRow
                key={row.id}
                row={row}
                importId={importId}
                onRowUpdated={onRowUpdated}
                borderColor={borderColor}
              />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

interface EditableRowProps {
  row: StagingRow;
  importId: string;
  onRowUpdated: (rowId: string, updates: Partial<StagingRow>) => void;
  borderColor: string;
}

function EditableRow({
  row,
  importId,
  onRowUpdated,
  borderColor,
}: EditableRowProps) {
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const [tradeDate, setTradeDate] = useState(row.trade_date ?? "");
  const [symbol, setSymbol] = useState(row.symbol ?? "");
  const [side, setSide] = useState(row.side ?? "");
  const [quantity, setQuantity] = useState(
    row.quantity !== null ? String(row.quantity) : "",
  );
  const [pricePerShare, setPricePerShare] = useState(
    row.price_per_share !== null ? String(row.price_per_share) : "",
  );

  function handleEdit() {
    // Reset local state to current row values when opening edit mode
    setTradeDate(row.trade_date ?? "");
    setSymbol(row.symbol ?? "");
    setSide(row.side ?? "");
    setQuantity(row.quantity !== null ? String(row.quantity) : "");
    setPricePerShare(
      row.price_per_share !== null ? String(row.price_per_share) : "",
    );
    setSaveError(null);
    setEditing(true);
  }

  function handleCancel() {
    setEditing(false);
    setSaveError(null);
  }

  async function handleSave() {
    setSaving(true);
    setSaveError(null);

    const updates: Partial<StagingRow> = {
      trade_date: tradeDate !== "" ? tradeDate : null,
      symbol: symbol !== "" ? symbol : null,
      side: side !== "" ? side : null,
      quantity: quantity !== "" ? Number(quantity) : null,
      price_per_share: pricePerShare !== "" ? Number(pricePerShare) : null,
    };

    const result = await patchRowAction(importId, row.id, updates);

    if (!result.success) {
      setSaveError(result.error ?? "Save failed");
      setSaving(false);
      return;
    }

    onRowUpdated(row.id, updates);
    setEditing(false);
    setSaving(false);
  }

  if (editing) {
    return (
      <>
        <tr className={`border-b ${borderColor}`}>
          <td className="py-2 px-3">
            <input
              type="date"
              value={tradeDate}
              onChange={(e) => setTradeDate(e.target.value)}
              className="border border-gray-300 rounded px-2 py-1 text-xs w-full"
            />
          </td>
          <td className="py-2 px-3">
            <input
              type="text"
              value={symbol}
              onChange={(e) => setSymbol(e.target.value.toUpperCase())}
              placeholder="AAPL"
              className="border border-gray-300 rounded px-2 py-1 text-xs w-24"
            />
          </td>
          <td className="py-2 px-3">
            <select
              value={side}
              onChange={(e) => setSide(e.target.value)}
              className="border border-gray-300 rounded px-2 py-1 text-xs"
            >
              <option value="">--</option>
              <option value="buy">buy</option>
              <option value="sell">sell</option>
            </select>
          </td>
          <td className="py-2 px-3 text-right">
            <input
              type="number"
              value={quantity}
              onChange={(e) => setQuantity(e.target.value)}
              min="0"
              step="any"
              className="border border-gray-300 rounded px-2 py-1 text-xs w-24 text-right"
            />
          </td>
          <td className="py-2 px-3 text-right">
            <input
              type="number"
              value={pricePerShare}
              onChange={(e) => setPricePerShare(e.target.value)}
              min="0"
              step="any"
              className="border border-gray-300 rounded px-2 py-1 text-xs w-28 text-right"
            />
          </td>
          <td className="py-2 px-3 text-gray-600">{row.currency}</td>
          <td className="py-2 px-3">
            {row.warnings.length > 0 && (
              <span className="text-xs text-amber-700">
                {row.warnings.join("; ")}
              </span>
            )}
          </td>
          <td className="py-2 px-3">
            <div className="flex gap-2">
              <button
                onClick={handleSave}
                disabled={saving}
                className="text-xs bg-blue-600 text-white px-2 py-1 rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {saving ? "Saving..." : "Save"}
              </button>
              <button
                onClick={handleCancel}
                disabled={saving}
                className="text-xs text-gray-600 hover:underline disabled:opacity-50"
              >
                Cancel
              </button>
            </div>
          </td>
        </tr>
        {saveError && (
          <tr>
            <td colSpan={8} className="px-3 pb-2">
              <p className="text-xs text-red-600">{saveError}</p>
            </td>
          </tr>
        )}
      </>
    );
  }

  return (
    <tr className={`border-b ${borderColor} hover:bg-white/50`}>
      <td className="py-2 px-3">
        {row.trade_date !== null ? (
          row.trade_date
        ) : (
          <span className="text-red-600 text-xs font-medium">missing</span>
        )}
      </td>
      <td className="py-2 px-3 font-medium">
        {row.symbol !== null ? (
          row.symbol
        ) : (
          <span className="text-red-600 text-xs font-medium">missing</span>
        )}
      </td>
      <td className="py-2 px-3">
        {row.side !== null ? (
          <span
            className={`px-2 py-1 rounded text-xs ${
              row.side === "buy"
                ? "bg-green-100 text-green-800"
                : "bg-red-100 text-red-800"
            }`}
          >
            {row.side.toUpperCase()}
          </span>
        ) : (
          <span className="text-red-600 text-xs font-medium">missing</span>
        )}
      </td>
      <td className="py-2 px-3 text-right">
        {row.quantity !== null ? (
          row.quantity
        ) : (
          <span className="text-red-600 text-xs font-medium">missing</span>
        )}
      </td>
      <td className="py-2 px-3 text-right">
        {row.price_per_share !== null ? (
          row.price_per_share
        ) : (
          <span className="text-red-600 text-xs font-medium">missing</span>
        )}
      </td>
      <td className="py-2 px-3 text-gray-600">{row.currency}</td>
      <td className="py-2 px-3">
        {row.warnings.length > 0 && (
          <span className="text-xs text-amber-700">
            {row.warnings.join("; ")}
          </span>
        )}
      </td>
      <td className="py-2 px-3">
        <button
          onClick={handleEdit}
          className="text-xs text-blue-600 hover:underline"
        >
          Edit
        </button>
      </td>
    </tr>
  );
}
