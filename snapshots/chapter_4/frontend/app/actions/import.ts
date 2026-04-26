"use server";

import { revalidatePath } from "next/cache";
import {
  importAlpaca,
  uploadFile,
  patchStagingRow,
  confirmImport,
} from "@/lib/api";
import type { StagingRow } from "@/lib/api";

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

export async function uploadFileAction(formData: FormData): Promise<{
  success: boolean;
  importId?: string;
  rows?: StagingRow[];
  error?: string;
}> {
  const file = formData.get("file") as File;
  if (!file || file.size === 0) {
    return { success: false, error: "No file selected" };
  }

  try {
    const detail = await uploadFile(file);
    revalidatePath("/import");
    return {
      success: true,
      importId: detail.import.id,
      rows: detail.rows,
    };
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : "Upload failed",
    };
  }
}

export async function patchRowAction(
  importId: string,
  rowId: string,
  updates: Record<string, unknown>,
): Promise<{ success: boolean; error?: string }> {
  try {
    await patchStagingRow(importId, rowId, updates);
    return { success: true };
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : "Update failed",
    };
  }
}

export async function confirmImportAction(importId: string): Promise<{
  success: boolean;
  inserted?: number;
  duplicates?: number;
  error?: string;
}> {
  try {
    const result = await confirmImport(importId);
    revalidatePath("/");
    revalidatePath("/transactions");
    return {
      success: true,
      inserted: result.inserted,
      duplicates: result.duplicates,
    };
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : "Confirm failed",
    };
  }
}
