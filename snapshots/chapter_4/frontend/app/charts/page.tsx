import { getPortfolio } from "@/lib/api";
import { StockChart } from "@/components/stock-chart";

export const dynamic = "force-dynamic";

export default async function ChartsPage() {
  let holdings;
  try {
    const portfolio = await getPortfolio();
    holdings = portfolio.holdings ?? [];
  } catch {
    return (
      <div className="text-center py-8">
        <p className="text-red-500">
          Failed to load portfolio. Is the backend running?
        </p>
      </div>
    );
  }

  if (holdings.length === 0) {
    return (
      <div className="text-center py-8">
        <p className="text-gray-500">
          No holdings yet. Add a transaction to get started, then come back to
          see charts.
        </p>
      </div>
    );
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Price Charts</h1>
      <StockChart holdings={holdings} />
    </div>
  );
}
