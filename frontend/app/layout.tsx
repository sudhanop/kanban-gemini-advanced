import type { Metadata } from "next";
import "./globals.css";
import { ThemeProvider } from "@/components/providers/ThemeProvider";
import { QueryProvider } from "@/components/providers/QueryProvider";
import { AuthProvider } from "@/components/providers/AuthProvider";
import { WebSocketProvider } from "@/components/providers/WebSocketProvider";
import { Toaster } from "react-hot-toast";

export const metadata: Metadata = {
  title: {
    default: "FlowBoard — AI-Powered Workflow Platform",
    template: "%s | FlowBoard",
  },
  description:
    "Enterprise-grade workflow platform combining Kanban, Timeline, Sprint, and Analytics views with AI assistance, real-time collaboration, and smart reminders.",
  keywords: [
    "kanban",
    "project management",
    "AI workflow",
    "sprint planning",
    "team collaboration",
    "productivity",
  ],
  openGraph: {
    title: "FlowBoard — AI-Powered Workflow Platform",
    description: "Enterprise workflow platform with AI assistance and real-time collaboration.",
    type: "website",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <link rel="icon" href="/favicon.svg" type="image/svg+xml" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </head>
      <body className="min-h-screen bg-background font-sans antialiased">
        <ThemeProvider attribute="class" defaultTheme="dark" enableSystem>
          <QueryProvider>
            <AuthProvider>
              <WebSocketProvider>
                {children}
                <Toaster
                  position="bottom-right"
                  toastOptions={{
                    style: {
                      background: "hsl(240 10% 7%)",
                      color: "hsl(0 0% 95%)",
                      border: "1px solid hsl(240 10% 14%)",
                      borderRadius: "12px",
                    },
                    success: {
                      iconTheme: { primary: "#22c55e", secondary: "#fff" },
                    },
                    error: {
                      iconTheme: { primary: "#ef4444", secondary: "#fff" },
                    },
                  }}
                />
              </WebSocketProvider>
            </AuthProvider>
          </QueryProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
