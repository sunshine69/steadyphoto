"use client";

import { useState } from 'react';
import { 
  Images, // Replaced 'Photos' with 'Images' as it is the correct export name in lucide-react
  Calendar, 
  Users, 
  Folder, 
  Settings, 
  Menu, 
  X,
  Plus,
  Search
} from 'lucide-react';
import Link from 'next/link';

export default function Sidebar() {
  const [isOpen, setIsOpen] = useState(false);

  const navItems = [
    { icon: Images, label: 'Photos', href: '/' }, // Also changed the menu item to use the correct icon if needed, or keep a generic one. Here I'll use Images for consistency with logo.
    { icon: Calendar, label: 'Timeline', href: '/timeline' },
    { icon: Users, label: 'People', href: '/people' },
    { icon: Folder, label: 'Albums', href: '/albums' },
  ];

  const bottomItems = [
    { icon: Settings, label: 'Settings', href: '/settings' },
  ];

  return (
    <>
      {/* Mobile Menu Button */}
      <button 
        className="lg:hidden fixed top-4 left-4 z-50 p-2 bg-zinc-900/80 backdrop-blur-sm rounded-md border border-zinc-800"
        onClick={() => setIsOpen(!isOpen)}
      >
        {isOpen ? <X size={20} /> : <Menu size={20} />}
      </button>

      {/* Overlay for mobile */}
      {isOpen && (
        <div 
          className="lg:hidden fixed inset-0 bg-black/50 z-40"
          onClick={() => setIsOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside className={`
        fixed lg:static inset-y-0 left-0 z-40 w-64 bg-zinc-950 border-r border-zinc-800/50 
        transform transition-transform duration-300 ease-in-out flex flex-col
        ${isOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
      `}>
        {/* Logo */}
        <div className="p-6 border-b border-zinc-800/50">
          <h1 className="text-xl font-bold tracking-tight text-white flex items-center gap-2">
            <Images className="w-6 h-6 text-blue-500" />
            SteadyPhoto
          </h1>
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-4 space-y-1 overflow-y-auto">
          {navItems.map((item) => (
            <Link
              key={item.label}
              href={item.href}
              className="flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium 
                         text-zinc-400 hover:text-white hover:bg-zinc-900 transition-colors"
            >
              <item.icon size={18} />
              {item.label}
            </Link>
          ))}
        </nav>

        {/* Bottom Actions */}
        <div className="p-4 border-t border-zinc-800/50 space-y-1">
          {bottomItems.map((item) => (
            <Link
              key={item.label}
              href={item.href}
              className="flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium 
                         text-zinc-400 hover:text-white hover:bg-zinc-900 transition-colors"
            >
              <item.icon size={18} />
              {item.label}
            </Link>
          ))}
          
          <button className="w-full flex items-center justify-center gap-2 px-3 py-2.5 rounded-md 
                             text-sm font-medium bg-blue-600 hover:bg-blue-700 text-white transition-colors mt-4">
            <Plus size={18} />
            Import Photos
          </button>
        </div>
      </aside>
    </>
  );
}
