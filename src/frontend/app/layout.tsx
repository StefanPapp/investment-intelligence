import type { Metadata } from "next";
import Link from "next/link";
import { getHealth } from "@/lib/api";
import "./globals.css";

export const metadata: Metadata = {
  title: "Stock Portfolio Manager",
  description: "Track your stock investments",
};

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const health = await getHealth().catch(() => null);
  const isTest = health?.db_target === "test";

  return (
    <html lang="en">
      <body className="bg-gray-50 min-h-screen">
        <nav className="bg-white shadow-sm border-b">
          <div className="max-w-6xl mx-auto px-4 py-3 flex items-center gap-6">
            <Link href="/" className="text-lg font-semibold text-gray-900">
              Portfolio
            </Link>
            <Link
              href="/transactions"
              className="text-gray-600 hover:text-gray-900"
            >
              Transactions
            </Link>
            <Link href="/charts" className="text-gray-600 hover:text-gray-900">
              Charts
            </Link>
            <Link href="/add" className="text-gray-600 hover:text-gray-900">
              Add Transaction
            </Link>
            {isTest && (
              <span className="ml-auto px-2 py-0.5 text-xs font-semibold rounded bg-amber-100 text-amber-800 border border-amber-300">
                TEST
              </span>
            )}
          </div>
        </nav>
        <main className="max-w-6xl mx-auto px-4 py-6">{children}</main>
      </body>
    </html>
  );
}
