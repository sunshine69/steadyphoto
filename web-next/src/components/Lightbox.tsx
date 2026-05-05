"use client";

import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Photo } from '@/types/photo';
import { getPhotoOriginalUrl } from '@/lib/api';
import { X } from 'lucide-react'; // Need to check if lucide-react is installed, or use a simple SVG

export function Lightbox({ 
  photo, 
  isOpen, 
  onClose 
}: { 
  photo: Photo | null; 
  isOpen: boolean; 
  onClose: () => void; 
}) {
  return (
    <AnimatePresence>
      {isOpen && photo && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/95 p-4 md:p-10"
          onClick={onClose}
        >
          {/* Close Button */}
          <button
            className="absolute right-6 top-6 z-[60] rounded-full bg-white/10 p-2 text-white transition-colors hover:bg-white/20"
            onClick={(e) => {
              e.stopPropagation();
              onClose();
            }}
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
          </button>

          {/* Image Container */}
          <motion.div
            initial={{ scale: 0.9, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ scale: 0.9, opacity: 0 }}
            className="relative max-h-full max-w-full overflow-hidden rounded-lg"
            onClick={(e) => e.stopPropagation()}
          >
            <img
              src={getPhotoOriginalUrl(photo.id)}
              alt={photo.filename}
              className="max-h-[85vh] max-w-full object-contain"
            />
            <div className="mt-4 text-center text-sm text-zinc-400">
              {photo.filename}
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
