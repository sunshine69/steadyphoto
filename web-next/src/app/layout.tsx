"use client";

import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Sidebar from '@/components/Sidebar';
import Header from '@/components/Header';
import './globals.css'; // <-- Added this import

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 60 * 1000, // 1 minute
      retry: 1,
    },
  },
});

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <QueryClientProvider client={queryClient}>
      <html lang="en">
        <body className="bg-zinc-950 text-zinc-100 antialiased flex h-screen overflow-hidden selection:bg-blue-500/30">
          {/* Sidebar - Fixed width, never scrolls */}
          <Sidebar />

          {/* Main Content Area */}
          <div className="flex-1 flex flex-col min-w-0 relative">
            {/* Top Bar for Search and User Profile */}
            <Header />
            
            {/* Scrollable Photo Viewport */}
            <main className="flex-1 overflow-y-auto p-4 scroll-smooth custom-scrollbar">
              {children}
            </main>
          </div>

          {/* Global Lightbox Portal will be rendered via portal in context or root */}
        </body>
      </html>
    </QueryClientProvider>
  );
}
