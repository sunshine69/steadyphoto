import { Component, ElementRef, ViewChild, OnInit, OnDestroy, inject, ChangeDetectionStrategy } from '@angular/core';

import { HttpEventType } from '@angular/common/http';

import { UploadService, UploadProgressEvent } from '../../services/upload.service';
import { UploadTriggerService } from '../../services/upload-trigger.service';
import { Subscription } from 'rxjs';

// Wrapper interface to safely store metadata alongside the original native File object  
interface SelectableFile { 
  name: string; size: number; type: string; // Convenience aliases for templates and progress display  
  previewUrl?: string;
  nativeFile: File; // Keeps actual Blob/File prototype intact - required by FormData.append()
}

@Component({
    selector: 'app-upload-modal',
    imports: [],
    template: `
    <!-- Backdrop -->
    @if (isVisible) {
      <div class="modal-backdrop" (click)="closeModal()"></div>
    }
    
    <!-- Modal Container -->
    @if (isVisible) {
      <div class="upload-modal-container">
        <!-- Mobile handle bar (only visible on bottom sheet) -->
        <div class="mobile-handle"></div>
        
        <!-- Header -->
        <div class="card-header">
          <h2 class="modal-title">Upload Media</h2>
          <button class="close-btn" (click)="closeModal()" aria-label="Close">✕</button>
        </div>
        
        <!-- Content Area: Changes based on state -->
        <!-- State 1: Drop Zone / File Selection -->
        @if (!isUploading && !uploadComplete) {
          <div
            class="drop-zone"
            [class.drag-over]="dragOver"
            (click)="triggerFilePicker()"
            (dragover)="onDragOver($event)"
            (dragleave)="onDragLeave()"
            (drop)="onDrop($event)">
            <!-- Hidden File Input -->
            <input type="file" #fileInput multiple accept="image/jpeg,image/png,video/mp4,video/quicktime,.jpg,.jpeg,.png,.mp4,.mov" class="hidden-input" (change)="handleFileSelection($event)">
            <div class="drop-icon">📁</div>
            <p class="drop-text-main">Tap to select files</p>
            <p class="drop-subtext">JPG, PNG, MP4, MOV — max 1GB each</p>
            
            <!-- Preview Grid -->
            @if (selectedFiles.length > 0) {
              <div class="preview-grid" #fileGrid>
                @for (file of selectedFiles; track file; let i = $index) {
                  <div class="preview-item">
                    <img [src]="getPreviewUrl(file)" alt="{{ file.name }}" class="thumb-img">
                    <button class="remove-btn" (click)="$event.stopPropagation(); removeFile(i)" aria-label="Remove file">✕</button>
                  </div>
                }
              </div>
              
              <!-- Upload Button -->
              <div class="upload-actions">
                <span>{{ selectedFiles.length }} file{{ selectedFiles.length === 1 ? '' : 's' }} ready</span>
                <button class="btn-upload" (click)="$event.stopPropagation(); startUpload()">Upload</button>
              </div>
            }
          </div>
        }
        
        <!-- State 2: Uploading Progress -->
        @if (isUploading) {
          <div class="progress-area">
            <p class="upload-status-text">{{ currentFileName }}</p>
            <div class="file-progress-list" #scrollContainer>
              @for (evt of progressEvents; track evt; let i = $index) {
                <div class="single-file-row">
                  <span class="fname-truncate">{{ evt.fileName.split(',')[0] }}...</span>
                  <div class="progress-bar-bg">
                    <div
                      class="progress-fill"
                      [style.width.%]="evt.progress"></div>
                  </div>
                  <span class="pct-text">{{ getPctText(evt) }}%</span>
                </div>
              }
            </div>
          </div>
        }
        
        <!-- State 3: Success / Error -->
        @if (uploadComplete) {
          <div class="result-area" [class.success]="!hasError" [class.error]="hasError">
            <span class="icon">{{ hasError ? '⚠️' : '✅' }}</span>
            <p>{{ uploadMessage }}</p>
            
            <!-- Show skipped duplicates if any -->
            @if (response?.skipped_duplicates && response.skipped_duplicates.length > 0) {
              <div class="dupes-box">
                <strong>Duplicates Skipped ({{ response.skipped_duplicates.length }}):</strong>
                <ul>
                  @for (dup of response.skipped_duplicates; track dup) {
                    <li>{{ dup.filename }}</li>
                  }
                </ul>
              </div>
            }
            
            <!-- Show uploaded IDs if any -->
            @if (response?.uploaded && response.uploaded.length > 0) {
              <div class="dupes-box">
                <strong>Newly Uploaded ({{ response.uploaded.length }}):</strong>
                <p>IDs: {{ getUploadedIds() }}</p>
              </div>
            }
            
            <button class="btn-close-result" (click)="closeModal()">Close</button>
          </div>
        }
      </div>
    }
    `,
    changeDetection: ChangeDetectionStrategy.Eager,
    styles: [`
    /* Backdrop */
    .modal-backdrop {
      position: fixed; inset: 0; background-color: rgba(0,0,0,0.6); z-index: 998; backdrop-filter: blur(2px);
    }

    /* Modal Container - Desktop Default */
    .upload-modal-container {
      position: fixed;
      top: 50%;
      left: 50%;
      transform: translate(-50%, -50%);
      width: 640px;
      max-width: 95vw;
      z-index: 1000;
      background-color: #1e293b;
      border-radius: 16px;
      box-shadow: 0 20px 40px rgba(0,0,0,0.5);
      display: flex;
      flex-direction: column;
      max-height: 85vh;
      overflow-y: auto;
      -webkit-overflow-scrolling: touch;
    }

    /* Mobile handle bar */
    .mobile-handle {
      display: none;
      align-self: center;
      width: 40px;
      height: 4px;
      background-color: #4b5563;
      border-radius: 2px;
      margin-top: 12px;
      margin-bottom: 8px;
      flex-shrink: 0;
    }

    /* Header */
    .card-header {
      padding: 16px 20px;
      border-bottom: 1px solid #374151;
      display: flex;
      justify-content: space-between;
      align-items: center;
      position: sticky;
      top: 0;
      background-color: #1e293b;
      z-index: 1;
      min-height: 48px;
    }
    .modal-title {
      margin: 0;
      font-size: 18px;
      color: #f3f4f6;
      font-weight: 600;
    }
    .close-btn {
      background: none;
      border: none;
      color: #9ca3af;
      cursor: pointer;
      font-size: 20px;
      padding: 8px;
      line-height: 1;
      transition: color 0.2s;
      min-width: 44px;
      min-height: 44px;
      display: flex;
      align-items: center;
      justify-content: center;
      border-radius: 8px;
    }
    .close-btn:hover {
      color: white;
      background-color: rgba(255,255,255,0.05);
    }

    /* Drop Zone */
    .drop-zone {
      border: 2px dashed #374151;
      margin: 20px;
      padding: 40px 20px;
      text-align: center;
      transition: all 0.2s ease;
      cursor: pointer;
      background-color: rgba(99, 102, 241, 0.03);
      border-radius: 8px;
    }
    .drop-zone.drag-over {
      border-color: #6366f1;
      background-color: rgba(99, 102, 241, 0.15);
      transform: scale(1.01);
      box-shadow: inset 0 0 20px rgba(99, 102, 241, 0.1);
    }
    .drop-icon {
      font-size: 36px;
      margin-bottom: 8px;
      opacity: 0.7;
      pointer-events: none;
    }
    .drop-text-main {
      color: #e5e7eb;
      font-weight: 500;
      margin: 4px 0;
      pointer-events: none;
      font-size: 15px;
    }
    .drop-subtext {
      color: #6b7280;
      font-size: 13px;
      margin-bottom: 16px;
      pointer-events: none;
    }
    
    /* Preview Grid */
    .preview-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(90px, 1fr));
      gap: 12px;
      padding-top: 16px;
      border-top: 1px solid #374151;
    }
    .preview-item {
      position: relative;
      aspect-ratio: 1/1;
      background-color: #0f172a;
      border-radius: 8px;
      overflow: hidden;
      cursor: pointer;
      transition: transform 0.2s, box-shadow 0.2s;
      display: flex;
      align-items: center;
      justify-content: center;
    }
    .preview-item:hover {
      transform: translateY(-3px);
      box-shadow: 0 4px 12px rgba(0,0,0,0.5);
    }
    .thumb-img {
      width: 100%;
      height: 100%;
      object-fit: cover;
      pointer-events: none;
    }
    
    /* Remove Button - Always visible on mobile */
    .remove-btn {
      position: absolute;
      top: 4px;
      right: 4px;
      background-color: rgba(239,68,68,0.9);
      color: white;
      border: none;
      width: 26px;
      height: 26px;
      line-height: 26px;
      text-align: center;
      font-size: 14px;
      cursor: pointer;
      border-radius: 50%;
      opacity: 1; /* Always visible */
      transition: background-color 0.2s, transform 0.2s;
      z-index: 2;
      -webkit-tap-highlight-color: transparent;
    }
    .remove-btn:active {
      background-color: #dc2626;
      transform: scale(0.9);
    }

    /* Actions */
    .upload-actions {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-top: 24px;
      padding: 0 8px;
      color: #9ca3af;
      font-size: 14px;
      flex-wrap: wrap;
      gap: 8px;
    }
    .btn-upload {
      background-color: #6366f1;
      color: white;
      border: none;
      padding: 10px 24px;
      border-radius: 8px;
      cursor: pointer;
      font-weight: 500;
      transition: all 0.2s ease;
      min-height: 44px;
      min-width: 44px;
    }
    .btn-upload:active {
      background-color: #4f46e5;
      transform: scale(0.98);
    }

    /* Progress Area */
    .progress-area {
      padding: 24px 20px;
      display: flex;
      flex-direction: column;
      gap: 8px;
      max-height: 50vh;
      overflow-y: auto;
      -webkit-overflow-scrolling: touch;
    }
    .upload-status-text {
      color: #9ca3af;
      font-size: 14px;
      margin-bottom: 8px;
      text-align: center;
    }
    
    /* File Row */
    .single-file-row {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 6px 0;
      border-bottom: 1px solid #374151;
      flex-wrap: wrap;
    }
    .fname-truncate {
      color: #e5e7eb;
      font-size: 13px;
      max-width: 180px;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      flex-shrink: 0;
    }
    
    /* Progress Bar - Responsive */
    .progress-bar-bg {
      flex: 1 1 150px;
      height: 6px;
      background-color: #374151;
      border-radius: 99px;
      overflow: hidden;
      position: relative;
      min-width: 100px;
    }
    .progress-fill {
      height: 100%;
      background: linear-gradient(90deg, #8b5cf6, #ec4899);
      width: 0%;
      transition: width 0.3s ease-out;
      border-radius: 99px;
    }

    /* Result Area */
    .result-area {
      padding: 24px 20px;
      text-align: center;
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 16px;
    }
    .icon {
      font-size: 32px;
      margin-bottom: 8px;
    }
    
    /* Success / Error Colors */
    .result-area.success p, .dupes-box strong {
      color: #4ade80;
    } 
    .result-area.error p {
      color: #f87171;
    }

    .btn-close-result {
      background-color: #374151;
      border: none;
      padding: 10px 24px;
      border-radius: 8px;
      cursor: pointer;
      margin-top: 12px;
      min-height: 44px;
      min-width: 44px;
    }

    /* Duplicates Box */
    .dupes-box {
      width: 90%;
      background-color: rgba(30,41,59,0.7);
      padding: 10px 14px;
      border-radius: 8px;
      text-align: left;
      font-size: 13px;
    }
    
    /* Utilities */
    .hidden-input {
      display: none;
    }

    /* ==============================
       RESPONSIVE BREAKPOINTS
       ============================== */

    /* Tablet (601px - 768px) - Smaller centered modal */
    @media (max-width: 768px) {
      .upload-modal-container {
        width: 90vw;
        max-width: 600px;
      }
    }

    /* Phone (≤ 600px) - Top Aligned Dialog */
    @media (max-width: 600px) {
      .upload-modal-container {
        position: fixed;
        top: 40px;
        left: 0;
        right: 0;
        width: 100%;
        max-width: 100vw;
        border-radius: 16px 16px 0 0;
        max-height: calc(100vh - 80px);
        transform: none;
      }
      .mobile-handle {
        display: block;
      }
      
      /* Compact padding for mobile */
      .drop-zone {
        margin: 12px;
        padding: 24px 16px;
      }
      .drop-text-main {
        font-size: 14px;
      }
      .drop-subtext {
        font-size: 12px;
        margin-bottom: 12px;
      }
      
      /* Smaller preview cards */
      .preview-grid {
        grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
        gap: 8px;
      }
      
      /* Remove button slightly smaller */
      .remove-btn {
        width: 24px;
        height: 24px;
        line-height: 24px;
        font-size: 12px;
        top: 2px;
        right: 2px;
      }
      
      /* Full-width upload button */
      .upload-actions {
        flex-direction: column;
        align-items: stretch;
        text-align: center;
        gap: 12px;
      }
      .btn-upload {
        width: 100%;
      }
      
      /* Compact progress area */
      .progress-area {
        padding: 16px 12px;
        gap: 6px;
      }
      .single-file-row {
        gap: 8px;
      }
      .fname-truncate {
        max-width: 120px;
        font-size: 12px;
      }
      .progress-bar-bg {
        flex: 1 1 100px;
        min-width: 80px;
      }
      
      /* Compact result area */
      .result-area {
        padding: 20px 16px;
        gap: 12px;
      }
      .dupes-box {
        width: 100%;
        padding: 8px 12px;
      }
    }

    /* Small phone (≤ 400px) - Extra compact */
    @media (max-width: 400px) {
      .upload-modal-container {
        max-height: 95vh;
      }
      .card-header {
        padding: 10px 12px;
        min-height: 44px;
      }
      .modal-title {
        font-size: 16px;
      }
      .drop-zone {
        margin: 8px;
        padding: 20px 12px;
      }
      .drop-icon {
        font-size: 28px;
      }
      .drop-text-main {
        font-size: 14px;
      }
      .drop-subtext {
        font-size: 11px;
        margin-bottom: 10px;
      }
      .preview-grid {
        grid-template-columns: repeat(auto-fill, minmax(64px, 1fr));
        gap: 6px;
      }
      .remove-btn {
        width: 22px;
        height: 22px;
        line-height: 22px;
        font-size: 11px;
      }
      .progress-area {
        padding: 12px 10px;
      }
      .fname-truncate {
        max-width: 90px;
      }
    }

    /* Very small phone (≤ 340px) - Minimal */
    @media (max-width: 340px) {
      .upload-modal-container {
        max-height: 98vh;
        border-radius: 12px 12px 0 0;
      }
      .drop-zone {
        padding: 16px 8px;
      }
      .preview-grid {
        grid-template-columns: repeat(3, 1fr);
        gap: 4px;
      }
    }

    /* Landscape mode on phones - Full screen overlay */
    @media (max-height: 500px) and (orientation: landscape) {
      .upload-modal-container {
        top: 0;
        bottom: 0;
        left: 0;
        right: 0;
        border-radius: 0;
        max-height: 100vh;
      }
      .mobile-handle {
        display: none;
      }
      .drop-zone {
        padding: 16px;
        margin: 8px;
      }
      .drop-icon {
        font-size: 24px;
        margin-bottom: 4px;
      }
      .drop-text-main {
        font-size: 13px;
        margin: 2px 0;
      }
      .drop-subtext {
        margin-bottom: 8px;
      }
      .preview-grid {
        grid-template-columns: repeat(auto-fill, minmax(60px, 1fr));
        gap: 4px;
      }
      .upload-actions {
        margin-top: 12px;
        flex-direction: row;
      }
      .btn-upload {
        width: auto;
      }
    }

    /* Touch device optimizations */
    @media (pointer: coarse) {
      .drop-zone {
        touch-action: manipulation;
      }
      .preview-item {
        touch-action: manipulation;
      }
      .preview-item:active {
        transform: scale(0.95);
      }
    }
  `]
})

export class UploadModalComponent implements OnInit, OnDestroy {
  private uploadService = inject(UploadService);
  private uploadTrigger = inject(UploadTriggerService);
  
  isVisible = false;
  dragOver = false;
  isUploading = false;
  uploadComplete = false;
  hasError = false;
  currentFileName = 'Waiting for files...';
  response: any = null;
  
  selectedFiles: SelectableFile[] = [];
  progressEvents: UploadProgressEvent[] = [];

  @ViewChild('fileInput') fileInput!: ElementRef<HTMLInputElement>;

  private subscription?: Subscription;

  ngOnInit(): void {
    // Subscribe to upload trigger service for open/close events
    this.subscription = this.uploadTrigger.isUploadModalOpen$.subscribe((isOpen: boolean) => {
      if (isOpen && !this.isVisible) {
        this.open();
      } else if (!isOpen && this.isVisible) {
        this.closeModal();
      }
    });

    window.addEventListener('keydown', this.handleEscapeKey);
  }

  ngOnDestroy(): void {
    if(this.subscription) this.subscription.unsubscribe();
    window.removeEventListener('keydown', this.handleEscapeKey);
    
    // Cleanup object URLs to prevent memory leaks
    for (const file of this.selectedFiles) {
      URL.revokeObjectURL(file.previewUrl as string); 
    }
  }

  private handleEscapeKey = (e: KeyboardEvent): void => {
    if(e.key === 'Escape' && this.isVisible) this.closeModal();
  };

  open(): void {
    // Reset state for new upload session
    this.selectedFiles = [];
    this.progressEvents = [];
    this.isUploading = false; 
    this.uploadComplete = false;
    this.hasError = false;
    this.response = null;
    
    if (this.fileInput?.nativeElement) {
      this.fileInput.nativeElement.value = '';
    }

    this.isVisible = true;
  }

  closeModal(): void {
    this.selectedFiles.forEach(f => URL.revokeObjectURL((f as any).previewUrl)); 
    this.selectedFiles = [];
    this.progressEvents = [];
    
    setTimeout(() => {
      this.isVisible = false;
      this.isUploading = false;
      this.uploadComplete = false;
    }, 100); 
  }

  triggerFilePicker(): void {
    if (this.fileInput?.nativeElement) {
      this.fileInput.nativeElement.click();
    }
  }

  handleFileSelection(event: Event): void { 
    const input = event.target as HTMLInputElement;
    if (input?.files && input.files.length > 0) {
      for(const file of Array.from(input.files)) {
        if(UploadService.isValidMediaType(file)) {
          this.selectedFiles.push({ 
            name: file.name, 
            size: file.size, 
            type: file.type, 
            nativeFile: file, 
            previewUrl: URL.createObjectURL(file) 
          }); 
        } else {
          alert(`Skipping unsupported type: ${file.name}`);
        }
      }
    }
  }

  onDragOver(e: DragEvent): void { 
    e.preventDefault(); 
    this.dragOver = true; 
  }

  onDragLeave(): void { 
    this.dragOver = false; 
  }
  
  onDrop(e: DragEvent): void { 
    e.preventDefault(); 
    this.dragOver = false; 
    
    if (e.dataTransfer?.files) {
      const droppedFiles = Array.from(e.dataTransfer.files);
      
      for(const file of droppedFiles) {
        if(UploadService.isValidMediaType(file)) {
          this.selectedFiles.push({ 
            name: file.name, 
            size: file.size, 
            type: file.type, 
            nativeFile: file, 
            previewUrl: URL.createObjectURL(file) 
          }); 
        } else {
          alert(`Skipping unsupported type: ${file.name}`);
        }
      }
    }
  }

  removeFile(index: number): void {
    const file = this.selectedFiles[index];
    if (file && file.previewUrl) URL.revokeObjectURL(file.previewUrl as string); 
    this.selectedFiles.splice(index, 1);
  }

  getPreviewUrl(file: SelectableFile): string { 
    return file.previewUrl || ''; 
  }

  getUploadedIds(): string {
    if (!this.response?.uploaded) return '';
    return this.response.uploaded.map((item: any) => item.id).join(', ');
  }

  getPctText(evt: UploadProgressEvent): number {
    return evt.status === 'completed' ? 100 : Math.round(evt.progress);
  }

  startUpload(): void {
    if(this.selectedFiles.length === 0) return;

    this.isUploading = true;
    this.uploadComplete = false;
    
    const filesToUpload: SelectableFile[] = [...this.selectedFiles];
    
    // Initialize progress events for each file
    this.progressEvents = [];
    for (const f of filesToUpload) {
      this.progressEvents.push({ fileName: f.name, status: 'uploading' as const, progress: 0 });
    }

    const totalFiles = filesToUpload.length;
    
    // Upload files in SERIAL (one at a time) — critical for mobile stability.
    // uploadFiles emits UploadProgressEvent objects directly (not HttpEventType).
    this.uploadService.uploadFiles(filesToUpload as any).subscribe({
      next: (event: any) => {
        // Check if this is a progress event (from our serial uploadFiles)
        if (event.fileName && event.status) {
          const progressEvt = event as UploadProgressEvent;
          
          // Update the progress event for this specific file
          this.progressEvents = this.progressEvents.map(p => 
            p.fileName === progressEvt.fileName 
              ? { ...p, status: progressEvt.status as 'uploading' | 'completed' | 'error', progress: progressEvt.progress } 
              : p
          );
          
          // Update current file name being uploaded
          this.currentFileName = progressEvt.fileName;
          
          // Check if any file had an error
          if (progressEvt.status === 'error') {
            this.hasError = true;
          }
        }
        // Also handle raw HttpEventType events (in case uploadFiles still emits them internally)
        else if (event.type === HttpEventType.UploadProgress && event.total) {
          const percent = Math.round((event.loaded / event.total) * 100);
          
          this.progressEvents = this.progressEvents.map(p => 
            p.status === 'uploading' 
              ? { ...p, progress: percent } 
              : p
          );
          
          if(totalFiles > 0) {
            const names = filesToUpload.slice(0, 2).map(f => f.name).join(', ');
            this.currentFileName = `Uploading: ${names}${totalFiles > 2 ? '...' : ''}`;
          }
        } else if (event.type === HttpEventType.Response && event.body) {
          const resData = event.body as any;
          
          // Mark current uploading file as completed
          this.progressEvents = this.progressEvents.map(p => 
            p.status === 'uploading' 
              ? { ...p, status: 'completed' as const, progress: 100 } 
              : p
          );

          this.response = resData; 

          if (this.response && Object.keys(this.response).length > 0) {
            const dupCount = (this.response.skipped_duplicates as any[])?.length || 0;
            if (dupCount > 0) {
              this.uploadMessage = `Upload complete. ${dupCount} duplicate${dupCount > 1 ? 's' : ''} skipped.`;
            } else {
              this.uploadMessage = 'Media uploaded successfully!';
            }
            window.dispatchEvent(new CustomEvent('media-upload-complete'));
          } else {
            console.warn('Upload returned empty response');
            this.hasError = true;
            this.uploadMessage = 'Server did not return expected data.';
          }
        } else if (event.type === HttpEventType.Response && !event.body) { 
          this.progressEvents = this.progressEvents.map(p => 
            p.status === 'uploading' 
              ? { ...p, status: 'completed' as const, progress: 100 } 
              : p
          );
          console.log('Upload completed with empty body');
        }
      },
      
      error: (err) => { 
        console.error('Unexpected upload stream error:', err);
        this.hasError = true;
        this.uploadMessage = `Network failure during transfer.`;
      },

      complete: () => { 
        console.log('Upload sequence completed.');
        this.isUploading = false;
        this.uploadComplete = true;
      }
    });
  }

  uploadMessage = 'Media uploaded successfully!'; 
}