import type { Metadata } from "next";
import "./globals.css";
import { PreferencesProvider } from "@/lib/preferences/preferences-context";
import { AuthProvider } from "@/lib/auth/auth-context";

export const metadata: Metadata = {
  title: "Hamburger Gateway Dashboard",
  description: "Hamburger 网关管理面板",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <body className="antialiased">
        <PreferencesProvider><AuthProvider>{children}</AuthProvider></PreferencesProvider>
      </body>
    </html>
  );
}
