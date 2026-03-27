import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "Stock Portfolio Manager",
  description: "Track your stock investments",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="bg-gray-50 min-h-screen">
        <nav className="bg-white shadow-sm border-b">
          <div className="max-w-6xl mx-auto px-4 py-3 flex items-center gap-6">
            <Link href="/" className="text-lg font-semibold text-gray-900">
              Portfolio
            </Link>
            <Link href="/transactions" className="text-gray-600 hover:text-gray-900">
              Transactions
            </Link>
            <Link href="/add" className="text-gray-600 hover:text-gray-900">
              Add Transaction
            </Link>
          </div>
        </nav>
        <main className="max-w-6xl mx-auto px-4 py-6">{children}</main>
      </body>
    </html>
  );
}
