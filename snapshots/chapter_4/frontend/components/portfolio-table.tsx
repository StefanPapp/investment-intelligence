import { Holding } from "@/lib/api";

function formatCurrency(value: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(value);
}

function formatPercent(value: number): string {
  return `${value >= 0 ? "+" : ""}${value.toFixed(2)}%`;
}

export function PortfolioTable({ holdings }: { holdings: Holding[] }) {
  if (holdings.length === 0) {
    return (
      <p className="text-gray-500 text-center py-8">
        No holdings yet. Add a transaction to get started.
      </p>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b text-left text-gray-500">
            <th className="py-3 px-2">Ticker</th>
            <th className="py-3 px-2">Name</th>
            <th className="py-3 px-2 text-right">Shares</th>
            <th className="py-3 px-2 text-right">Avg Cost</th>
            <th className="py-3 px-2 text-right">Price</th>
            <th className="py-3 px-2 text-right">Value</th>
            <th className="py-3 px-2 text-right">Gain/Loss</th>
          </tr>
        </thead>
        <tbody>
          {holdings.map((h) => (
            <tr key={h.ticker} className="border-b hover:bg-gray-50">
              <td className="py-3 px-2 font-medium">{h.ticker}</td>
              <td className="py-3 px-2 text-gray-600">{h.name}</td>
              <td className="py-3 px-2 text-right">{h.total_shares}</td>
              <td className="py-3 px-2 text-right">
                {formatCurrency(h.avg_cost)}
              </td>
              <td className="py-3 px-2 text-right">
                {h.current_price > 0 ? formatCurrency(h.current_price) : "N/A"}
              </td>
              <td className="py-3 px-2 text-right">
                {formatCurrency(h.market_value)}
              </td>
              <td
                className={`py-3 px-2 text-right ${
                  h.gain_loss >= 0 ? "text-green-600" : "text-red-600"
                }`}
              >
                {formatCurrency(h.gain_loss)} ({formatPercent(h.gain_loss_pct)})
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
