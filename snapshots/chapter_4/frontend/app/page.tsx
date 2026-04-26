import { getPortfolio } from "@/lib/api";
import { PortfolioTable } from "@/components/portfolio-table";

export const dynamic = "force-dynamic";

export default async function PortfolioPage() {
  let portfolio;
  try {
    portfolio = await getPortfolio();
  } catch {
    return (
      <div className="text-center py-8">
        <p className="text-red-500">
          Failed to load portfolio. Is the backend running?
        </p>
      </div>
    );
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Portfolio Overview</h1>
        <div className="flex gap-4 text-sm">
          <div>
            <span className="text-gray-500">Total Value: </span>
            <span className="font-semibold">
              {new Intl.NumberFormat("en-US", {
                style: "currency",
                currency: "USD",
              }).format(portfolio.total_value)}
            </span>
          </div>
          <div
            className={
              portfolio.total_gain_loss >= 0 ? "text-green-600" : "text-red-600"
            }
          >
            <span className="text-gray-500">P&L: </span>
            <span className="font-semibold">
              {new Intl.NumberFormat("en-US", {
                style: "currency",
                currency: "USD",
              }).format(portfolio.total_gain_loss)}
            </span>
          </div>
        </div>
      </div>
      <div className="bg-white rounded-lg shadow-sm border p-4">
        <PortfolioTable holdings={portfolio.holdings ?? []} />
      </div>
    </div>
  );
}
