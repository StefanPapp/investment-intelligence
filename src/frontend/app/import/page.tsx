"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { importAlpacaAction, uploadFileAction } from "@/app/actions/import";
import FileDropzone from "@/components/file-dropzone";

export default function ImportPage() {
  const router = useRouter();

  // Alpaca state
  const [alpacaLoading, setAlpacaLoading] = useState(false);
  const [alpacaResult, setAlpacaResult] = useState<{
    success: boolean;
    created?: number;
    updated?: number;
    total?: number;
    error?: string;
  } | null>(null);

  // File upload state
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);

  async function handleAlpacaImport() {
    setAlpacaLoading(true);
    setAlpacaResult(null);
    const res = await importAlpacaAction();
    setAlpacaResult(res);
    setAlpacaLoading(false);
  }

  async function handleFileUpload() {
    if (!selectedFile) return;
    setUploading(true);
    setUploadError(null);

    const formData = new FormData();
    formData.append("file", selectedFile);

    const result = await uploadFileAction(formData);
    if (result.success && result.importId) {
      router.push(`/import/${result.importId}`);
    } else {
      setUploadError(result.error ?? "Upload failed");
    }
    setUploading(false);
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Import Transactions</h1>

      {/* Alpaca section */}
      <div className="bg-white rounded-lg shadow-sm border p-6 mb-6">
        <h2 className="text-lg font-semibold mb-2">Alpaca</h2>
        <p className="text-gray-600 text-sm mb-4">
          Import your filled orders from Alpaca. Existing orders are matched by
          order ID and updated if changed.
        </p>
        <button
          onClick={handleAlpacaImport}
          disabled={alpacaLoading}
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {alpacaLoading ? "Importing..." : "Import from Alpaca"}
        </button>
        {alpacaResult && (
          <div
            className={`mt-4 p-4 rounded-md text-sm ${
              alpacaResult.success
                ? "bg-green-50 text-green-800 border border-green-200"
                : "bg-red-50 text-red-800 border border-red-200"
            }`}
          >
            {alpacaResult.success ? (
              <p>
                Imported {alpacaResult.total} order
                {alpacaResult.total !== 1 ? "s" : ""}
                {" \u2014 "}
                {alpacaResult.created} created, {alpacaResult.updated} updated.
              </p>
            ) : (
              <p>Import failed: {alpacaResult.error}</p>
            )}
          </div>
        )}
      </div>

      {/* File upload section */}
      <div className="bg-white rounded-lg shadow-sm border p-6">
        <h2 className="text-lg font-semibold mb-2">File Import</h2>
        <p className="text-gray-600 text-sm mb-4">
          Upload a broker statement (CSV or screenshot). Transactions will be
          extracted using AI and presented for review before import.
        </p>
        <FileDropzone
          onFileSelected={(file) => {
            setSelectedFile(file);
            setUploadError(null);
          }}
          disabled={uploading}
        />
        {selectedFile && (
          <div className="mt-4 flex items-center gap-3">
            <span className="text-sm text-gray-700">{selectedFile.name}</span>
            <button
              onClick={handleFileUpload}
              disabled={uploading}
              className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {uploading ? "Extracting..." : "Upload & Extract"}
            </button>
          </div>
        )}
        {uploadError && (
          <div className="mt-4 p-4 rounded-md text-sm bg-red-50 text-red-800 border border-red-200">
            <p>{uploadError}</p>
          </div>
        )}
      </div>
    </div>
  );
}
