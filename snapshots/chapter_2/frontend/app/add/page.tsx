import { TransactionForm } from "@/components/transaction-form";
import { addTransactionAction } from "@/app/actions/transactions";

export default function AddTransactionPage() {
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Add Transaction</h1>
      <div className="bg-white rounded-lg shadow-sm border p-6">
        <TransactionForm action={addTransactionAction} />
      </div>
    </div>
  );
}
