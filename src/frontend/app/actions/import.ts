"use server";

import { revalidatePath } from "next/cache";
import { importAlpaca } from "@/lib/api";

export async function importAlpacaAction(): Promise<{
  success: boolean;
  created?: number;
  updated?: number;
  total?: number;
  error?: string;
}> {
  try {
    const result = await importAlpaca();
    revalidatePath("/");
    revalidatePath("/transactions");
    return {
      success: true,
      created: result.created,
      updated: result.updated,
      total: result.total,
    };
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : "Import failed",
    };
  }
}
