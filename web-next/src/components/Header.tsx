"use client";

import { Search, Bell, User } from 'lucide-react';

export default function Header() {
  return (
    <header className="h-16 border-b border-zinc-800/50 flex items-center justify-between px-4 lg:px-8 bg-zinc-950/50 backdrop-blur-sm sticky top-0 z-30">
      {/* Search Bar */}
      <div className="flex-1 max-w-xl mx-auto">
        <div className="relative group">
          <Search 
            className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500 group-focus-within:text-blue-500 transition-colors" 
            size={18} 
          />
          <input 
            type="text" 
            placeholder="Search photos, dates, or places..." 
            className="w-full bg-zinc-900/50 border border-zinc-800 rounded-lg pl-10 pr-4 py-2 text-sm text-zinc-200 focus:outline-none focus:border-blue-500/50 focus:ring-1 focus:ring-blue-500/50 transition-all placeholder:text-zinc-600"
          />
        </div>
      </div>

      {/* Right Actions */}
      <div className="flex items-center gap-4">
        <button className="p-2 text-zinc-400 hover:text-white hover:bg-zinc-900 rounded-full transition-colors relative">
          <Bell size={20} />
          <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-blue-500 rounded-full border border-zinc-950"></span>
        </button>
        
        <div className="w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-xs font-bold cursor-pointer hover:ring-2 ring-white/20 transition-all">
          JD
        </div>
      </div>
    </header>
  );
}
