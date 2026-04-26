"use client";

import { useCallback, useState } from "react";

interface FileDropzoneProps {
  onFileSelected: (file: File) => void;
  disabled?: boolean;
}

const ACCEPTED_TYPES = [".csv", ".png", ".jpg", ".jpeg", ".pdf"];
const MAX_SIZE = 10 * 1024 * 1024; // 10 MB

export default function FileDropzone({
  onFileSelected,
  disabled = false,
}: FileDropzoneProps) {
  const [dragActive, setDragActive] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const validateAndSelect = useCallback(
    (file: File) => {
      setError(null);
      const ext = file.name.substring(file.name.lastIndexOf(".")).toLowerCase();
      if (!ACCEPTED_TYPES.includes(ext)) {
        setError(
          `Unsupported file type: ${ext}. Accepted: ${ACCEPTED_TYPES.join(", ")}`,
        );
        return;
      }
      if (file.size > MAX_SIZE) {
        setError("File too large (max 10 MB)");
        return;
      }
      onFileSelected(file);
    },
    [onFileSelected],
  );

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    setDragActive(false);
    if (disabled) return;
    const file = e.dataTransfer.files[0];
    if (file) validateAndSelect(file);
  }

  function handleDragOver(e: React.DragEvent) {
    e.preventDefault();
    if (!disabled) setDragActive(true);
  }

  function handleDragLeave() {
    setDragActive(false);
  }

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (file) validateAndSelect(file);
  }

  return (
    <div>
      <div
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
          disabled
            ? "border-gray-200 bg-gray-50 cursor-not-allowed"
            : dragActive
              ? "border-blue-500 bg-blue-50"
              : "border-gray-300 hover:border-gray-400 cursor-pointer"
        }`}
      >
        <p className="text-gray-600 text-sm mb-2">
          Drag and drop a file here, or click to browse
        </p>
        <p className="text-gray-400 text-xs">
          CSV, PNG, JPG, or PDF (max 10 MB)
        </p>
        <input
          type="file"
          accept={ACCEPTED_TYPES.join(",")}
          onChange={handleChange}
          disabled={disabled}
          className="hidden"
          id="file-upload"
        />
        <label
          htmlFor="file-upload"
          className={`inline-block mt-3 px-4 py-2 text-sm rounded-md ${
            disabled
              ? "bg-gray-300 text-gray-500 cursor-not-allowed"
              : "bg-blue-600 text-white hover:bg-blue-700 cursor-pointer"
          }`}
        >
          Choose File
        </label>
      </div>
      {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
    </div>
  );
}
