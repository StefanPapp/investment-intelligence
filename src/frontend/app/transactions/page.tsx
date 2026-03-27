import Link from "next/link";
import { getTransactions } from "@/lib/api";
import { deleteTransactionAction } from "@/app/actions/transactions";

export const dynamic = "force-dynamic";

export default async function TransactionsPage() {
  let transactions;
  try {
    transactions = await getTransactions();
  } catch {
    return (
      <div className="text-center py-8">
        <p className="text-red-500">Failed to load transactions. Is the backend running?</p>
      </div>
    );
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Transaction History</h1>
        <Link
          href="/add"
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700"
        >
          Add Transaction
        </Link>
      </div>
      <div className="bg-white rounded-lg shadow-sm border">
        {transactions.length === 0 ? (
          <p className="text-gray-500 text-center py-8">No transactions yet.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-gray-500">
                <th className="py-3 px-4">Date</th>
                <th className="py-3 px-4">Ticker</th>
                <th className="py-3 px-4">Type</th>
                <th className="py-3 px-4 text-right">Shares</th>
                <th className="py-3 px-4 text-right">Price</th>
                <th className="py-3 px-4 text-right">Total</th>
                <th className="py-3 px-4">Actions</th>
              </tr>
            </thead>
            <tbody>
              {transactions.map((t) => (
                <tr key={t.id} className="border-b hover:bg-gray-50">
                  <td className="py-3 px-4">{t.transaction_date}</td>
                  <td className="py-3 px-4 font-medium">{t.ticker}</td>
                  <td className="py-3 px-4">
                    <span
                      className={`px-2 py-1 rounded text-xs ${
                        t.transaction_type === "buy"
                          ? "bg-green-100 text-green-800"
                          : "bg-red-100 text-red-800"
                      }`}
                    >
                      {t.transaction_type.toUpperCase()}
                    </span>
                  </td>
                  <td className="py-3 px-4 text-right">{t.shares}</td>
                  <td className="py-3 px-4 text-right">
                    {new Intl.NumberFormat("en-US", {
                      style: "currency",
                      currency: "USD",
                    }).format(t.price_per_share)}
                  </td>
                  <td className="py-3 px-4 text-right">
                    {new Intl.NumberFormat("en-US", {
                      style: "currency",
                      currency: "USD",
                    }).format(t.shares * t.price_per_share)}
                  </td>
                  <td className="py-3 px-4">
                    <div className="flex gap-2">
                      <Link
                        href={`/transactions/${t.id}/edit`}
                        className="text-blue-600 hover:underline text-xs"
                      >
                        Edit
                      </Link>
                      <form
                        action={deleteTransactionAction.bind(null, t.id)}
                      >
                        <button
                          type="submit"
                          className="text-red-600 hover:underline text-xs"
                        >
                          Delete
                        </button>
                      </form>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
