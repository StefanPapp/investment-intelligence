import { getTransaction } from "@/lib/api";
import { TransactionForm } from "@/components/transaction-form";
import { editTransactionAction } from "@/app/actions/transactions";
import { notFound } from "next/navigation";

export default async function EditTransactionPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let transaction;
  try {
    transaction = await getTransaction(id);
  } catch {
    notFound();
  }

  const boundAction = editTransactionAction.bind(null, id);

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Edit Transaction</h1>
      <div className="bg-white rounded-lg shadow-sm border p-6">
        <TransactionForm
          action={boundAction}
          transaction={transaction}
          showTickerField={false}
        />
      </div>
    </div>
  );
}
