"use client";

import { motion, AnimatePresence } from "framer-motion";
// Replaced 'Iso' with 'Settings2' as it is more universally supported across lucide-react versions
import { X, Calendar, Camera, Aperture, Settings2, Maximize2 } from "lucide-react";
import type { Photo } from "@/types/photo";

interface LightboxProps {
  isOpen: boolean;
  onClose: () => void;
  photoUrl: string | null;
  photoData?: Photo | null;
}

export default function Lightbox({ isOpen, onClose, photoUrl, photoData }: LightboxProps) {
  if (!photoUrl || !isOpen) return null;

  const imageVariants = {
    hidden: { opacity: 0, scale: 0.9 },
    visible: { 
      opacity: 1, 
      scale: 1,
      transition: { type: "spring" as const, damping: 25, stiffness: 300 }
    },
    exit: { 
      opacity: 0, 
      scale: 0.9,
      transition: { duration: 0.2 }
    }
  };

  return (
    <AnimatePresence>
      {/* Backdrop */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        className="fixed inset-0 z-[60] bg-black/95 flex items-center justify-center"
        onClick={onClose}
      >
        {/* Close Button */}
        <button 
          className="absolute top-4 right-4 p-2 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors z-[61]"
          onClick={(e) => { e.stopPropagation(); onClose(); }}
        >
          <X size={24} />
        </button>

        {/* Main Image Container */}
        <div className="relative w-full h-full flex items-center justify-center p-4">
          <motion.img
            layoutId={`photo-${photoData?.id}`}
            src={photoUrl}
            alt={photoData?.filename || "Photo"}
            variants={imageVariants}
            initial="hidden"
            animate="visible"
            exit="exit"
            className="max-w-full max-h-[85vh] object-contain rounded-sm shadow-2xl"
          />

          {/* Metadata Panel - Bottom */}
          {photoData && (
            <motion.div 
              initial={{ y: 100, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              exit={{ y: 100, opacity: 0 }}
              transition={{ delay: 0.2 }}
              className="absolute bottom-8 left-1/2 -translate-x-1/2 w-full max-w-2xl px-4"
            >
              <div className="bg-black/60 backdrop-blur-md rounded-lg p-4 border border-white/10">
                {/* Date and Filename */}
                <div className="flex items-center justify-between mb-3 pb-3 border-b border-white/10">
                  <div>
                    <h3 className="text-white font-medium text-sm">{photoData.filename}</h3>
                    <div className="flex items-center gap-2 text-zinc-400 text-xs mt-1">
                      <Calendar size={12} />
                      {new Date(photoData.captured_at).toLocaleString()}
                    </div>
                  </div>
                  <button 
                    className="p-2 rounded-md hover:bg-white/10 text-zinc-400 hover:text-white transition-colors"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <Maximize2 size={16} />
                  </button>
                </div>

                {/* EXIF Data Grid */}
                {photoData.metadata && (
                  <div className="grid grid-cols-3 gap-4 text-xs">
                    {photoData.metadata.camera && (
                      <div className="flex items-center gap-2 text-zinc-300">
                        <Camera size={14} className="text-blue-500" />
                        <span>{photoData.metadata.camera}</span>
                      </div>
                    )}
                    {photoData.metadata.iso && (
                      <div className="flex items-center gap-2 text-zinc-300">
                        {/* Replaced Iso icon with Settings2 for better compatibility */}
                        <Settings2 size={14} className="text-purple-500" />
                        <span>ISO {photoData.metadata.iso}</span>
                      </div>
                    )}
                    {photoData.metadata.aperture && (
                      <div className="flex items-center gap-2 text-zinc-300">
                        <Aperture size={14} className="text-green-500" />
                        <span>{photoData.metadata.aperture}</span>
                      </div>
                    )}
                  </div>
                )}

                {/* Dimensions */}
                {photoData.width && photoData.height && (
                  <div className="mt-3 pt-3 border-t border-white/10 text-xs text-zinc-500">
                    {photoData.width} × {photoData.height}px • {(photoData.size ? photoData.size / 1024 / 1024 : 0).toFixed(2)} MB
                  </div>
                )}
              </div>
            </motion.div>
          )}

          {/* Keyboard hint */}
          <div className="absolute bottom-4 right-4 text-xs text-zinc-600">
            Press ESC to close
          </div>
        </div>
      </motion.div>
    </AnimatePresence>
  );
}
