"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import {
  createTransaction,
  updateTransaction,
  deleteTransaction,
} from "@/lib/api";

export async function addTransactionAction(formData: FormData) {
  await createTransaction({
    ticker: (formData.get("ticker") as string).toUpperCase(),
    name: (formData.get("name") as string) || undefined,
    transaction_type: formData.get("transaction_type") as "buy" | "sell",
    shares: parseFloat(formData.get("shares") as string),
    price_per_share: parseFloat(formData.get("price_per_share") as string),
    transaction_date: formData.get("transaction_date") as string,
  });
  revalidatePath("/");
  revalidatePath("/transactions");
  redirect("/transactions");
}

export async function editTransactionAction(id: string, formData: FormData) {
  await updateTransaction(id, {
    transaction_type: formData.get("transaction_type") as "buy" | "sell",
    shares: parseFloat(formData.get("shares") as string),
    price_per_share: parseFloat(formData.get("price_per_share") as string),
    transaction_date: formData.get("transaction_date") as string,
  });
  revalidatePath("/");
  revalidatePath("/transactions");
  redirect("/transactions");
}

export async function deleteTransactionAction(id: string) {
  await deleteTransaction(id);
  revalidatePath("/");
  revalidatePath("/transactions");
  redirect("/transactions");
}
