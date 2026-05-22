import { Component, ElementRef, ViewChild, OnInit, OnDestroy, inject } from '@angular/core';

import { HttpEventType } from '@angular/common/http';


import { CommonModule } from '@angular/common';
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
  standalone: true,
  imports: [CommonModule],
  template: `
    <!-- Backdrop -->
    <div class="modal-backdrop" *ngIf="isVisible" (click)="closeModal()"></div>

    <!-- Modal Container -->
    <div class="upload-modal-container" *ngIf="isVisible">
      <div class="upload-card">
        <!-- Header -->
        <div class="card-header">
          <h2 class="modal-title">Upload Media</h2>
          <button class="close-btn" (click)="closeModal()">✕</button>
        </div>

        <!-- Content Area: Changes based on state -->
        
        <!-- State 1: Drop Zone / File Selection -->
        <ng-container *ngIf="!isUploading && !uploadComplete">
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
            <p class="drop-text-main">Drag & drop files here or click to browse</p>
            <p class="drop-subtext">Supports JPG, PNG, MP4, MOV (Max 1GB)</p>

            <!-- Preview Grid -->
            <ng-container *ngIf="selectedFiles.length > 0; else emptyState">
              <div class="preview-grid" #fileGrid>
                <div *ngFor="let file of selectedFiles; let i = index" class="preview-item">
                  <img [src]="getPreviewUrl(file)" alt="{{ file.name }}" class="thumb-img" (click)="removeFile(i)">
                  <button class="remove-btn" (click)="$event.stopPropagation(); removeFile(i)">✕</button>
                </div>
              </div>

              <!-- Upload Button -->
              <div class="upload-actions">
                <span>{{ selectedFiles.length }} files ready to upload</span>
                <button class="btn-upload" (click)="startUpload()">Start Upload</button>
              </div>
            </ng-container>

            <ng-template #emptyState></ng-template>
          </div>
        </ng-container>

        <!-- State 2: Uploading Progress -->
        <ng-container *ngIf="isUploading">
          <div class="progress-area">
             <p class="upload-status-text">{{ currentFileName }}</p>
             
             <div class="file-progress-list" #scrollContainer>
               <div *ngFor="let evt of progressEvents; let i = index" class="single-file-row">
                 <span class="fname-truncate">{{ evt.fileName.split(',')[0] }}...</span>
                 
                 <!-- Individual file bar (simplified for batch) -->
                 <div class="progress-bar-bg">
                   <div 
                     class="progress-fill" 
                     [style.width.%]="evt.progress"></div>
                 </div>

                 <span class="pct-text">{{ evt.status === 'completed' ? 100 : Math.round(evt.progress) }}%</span>
               </div>
             </div>
          </div>
        </ng-container>

        <!-- State 3: Success / Error -->
        <ng-container *ngIf="uploadComplete">
           <div class="result-area" [class.success]="!hasError" [class.error]="hasError">
              <span class="icon">{{ hasError ? '⚠️' : '✅' }}</span>
              <p>{{ uploadMessage }}</p>

              <!-- Show skipped duplicates if any -->
              <div *ngIf="response?.skipped_duplicates && response.skipped_duplicates.length > 0" class="dupes-box">
                <strong>Duplicates Skipped ({{ response.skipped_duplicates.length }}):</strong>
                <ul>
                  <li *ngFor="let dup of response.skipped_duplicates">{{ dup.filename }}</li>
                </ul>
              </div>

              <!-- Show uploaded IDs if any -->
               <div *ngIf="response?.uploaded && response.uploaded.length > 0" class="dupes-box">
                <strong>Newly Uploaded ({{ response.uploaded.length }}):</strong>
                 <p>IDs: {{ getUploadedIds() }}</p>
              </div>

             <button class="btn-close-result" (click)="closeModal()">Close</button>
           </div>
        </ng-container>

      </div>
    </div>
  `,
  styles: [`
    /* Backdrop */
    .modal-backdrop {
      position: fixed; inset: 0; background-color: rgba(0,0,0,0.6); z-index: 998; backdrop-filter: blur(2px);
    }

    /* Modal Container */
    .upload-modal-container {
      position: fixed; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 640px; max-width: 95vw; z-index: 1000;
      background-color: #1e293b; border-radius: 16px; box-shadow: 0 20px 40px rgba(0,0,0,0.5); display: flex; flex-direction: column; max-height: 85vh; overflow-y: auto;
    }

    /* Header */
    .card-header { padding: 16px 20px; border-bottom: 1px solid #374151; display: flex; justify-content: space-between; align-items: center; position: sticky; top: 0; background-color: #1e293b; z-index: 1; }
    .modal-title { margin: 0; font-size: 18px; color: #f3f4f6; font-weight: 600; }
    .close-btn { background: none; border: none; color: #9ca3af; cursor: pointer; font-size: 20px; padding: 4px; line-height: 1; transition: color 0.2s; }
    .close-btn:hover { color: white; }

    /* Drop Zone */
    .drop-zone { border: 2px dashed #374151; margin: 20px; padding: 40px 20px; text-align: center; transition: all 0.2s ease; cursor: pointer; background-color: rgba(99, 102, 241, 0.03); border-radius: 8px; }
    .drop-zone.drag-over { border-color: #6366f1; background-color: rgba(99, 102, 241, 0.15); transform: scale(1.01); box-shadow: inset 0 0 20px rgba(99, 102, 241, 0.1); }
    .drop-icon { font-size: 36px; margin-bottom: 8px; opacity: 0.7; pointer-events: none; }
    .drop-text-main { color: #e5e7eb; font-weight: 500; margin: 4px 0; pointer-events: none; }
    .drop-subtext { color: #6b7280; font-size: 13px; margin-bottom: 16px; pointer-events: none; }
    
    /* Preview Grid */
    .preview-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(90px, 1fr)); gap: 12px; padding-top: 16px; border-top: 1px solid #374151; }
    .preview-item { position: relative; aspect-ratio: 1/1; background-color: #0f172a; border-radius: 8px; overflow: hidden; cursor: pointer; transition: transform 0.2s, box-shadow 0.2s; display: flex; align-items: center; justify-content: center;}
    .preview-item:hover { transform: translateY(-3px); box-shadow: 0 4px 12px rgba(0,0,0,0.5); }
    .thumb-img { width: 100%; height: 100%; object-fit: cover; pointer-events: none; }
    
    /* Remove Button */
    .remove-btn { position: absolute; top: 4px; right: 4px; background-color: rgba(239,68,68,0.9); color: white; border: none; width: 22px; height: 22px; line-height: 18px; text-align: center; font-size: 14px; cursor: pointer; border-radius: 50%; opacity: 0; transition: opacity 0.2s, transform 0.2s; }
    .preview-item:hover .remove-btn { opacity: 1; }

    /* Actions */
    .upload-actions { display: flex; justify-content: space-between; align-items: center; margin-top: 24px; padding: 0 8px; color: #9ca3af; font-size: 14px; }
    .btn-upload { background-color: #6366f1; color: white; border: none; padding: 10px 24px; border-radius: 8px; cursor: pointer; font-weight: 500; transition: all 0.2s ease; }
    .btn-upload:hover { background-color: #4f46e5; transform: translateY(-1px); box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3); }

    /* Progress Area */
    .progress-area { padding: 24px; display: flex; flex-direction: column; gap: 8px; max-height: 50vh; overflow-y: auto; }
    .upload-status-text { color: #9ca3af; font-size: 14px; margin-bottom: 8px; text-align: center;}
    
    /* File Row */
    .single-file-row { display: flex; align-items: center; gap: 12px; padding: 6px 0; border-bottom: 1px solid #374151; }
    .fname-truncate { color: #e5e7eb; font-size: 13px; max-width: 180px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; flex-shrink: 0;}
    
    /* Progress Bar */
    .progress-bar-bg { width: 240px; height: 6px; background-color: #374151; border-radius: 99px; overflow: hidden; position: relative; }
    .progress-fill { height: 100%; background: linear-gradient(90deg, #8b5cf6, #ec4899); width: 0%; transition: width 0.3s ease-out; border-radius: 99px;}

    /* Result Area */
    .result-area { padding: 24px; text-align: center; display: flex; flex-direction: column; align-items: center; gap: 16px; }
    .icon { font-size: 32px; margin-bottom: 8px;}
    
    /* Success / Error Colors */
    .result-area.success p, .dupes-box strong { color: #4ade80; } 
    .result-area.error p { color: #f87171; }

    .btn-close-result { background-color: #374151; border: none; padding: 8px 20px; border-radius: 6px; cursor: pointer; margin-top: 12px;}
    
    /* Duplicates Box */
    .dupes-box { width: 90%; background-color: rgba(30,41,59,0.7); padding: 8px 12px; border-radius: 6px; text-align: left; font-size: 13px;}
    
    /* Utilities */
    .hidden-input { display: none; }

    @media (max-width: 480px) { 
      .upload-modal-container { width: 95vw; top: auto; bottom: 0; transform: translateY(0); border-radius: 16px 16px 0 0; max-height: 80vh;}
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
  response: any = null; // Stores backend JSON payload
  
  selectedFiles: SelectableFile[] = [];
  progressEvents: UploadProgressEvent[] = [];

  @ViewChild('fileInput') fileInput!: ElementRef<HTMLInputElement>;

  private subscription?: Subscription;

  ngOnInit(): void {
    
    // Subscribe to upload trigger service for open/close events (like PresentationMode)
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
      this.fileInput.nativeElement.value = ''; // Clear previous selection so same file can be selected again
    }


    this.isVisible = true;
  }

  closeModal(): void {
     this.selectedFiles.forEach(f => URL.revokeObjectURL((f as any).previewUrl)); 
     this.selectedFiles = [];
     this.progressEvents = [];
     
     setTimeout(() => { // Delay hiding to allow backdrop transition if we added one, or just clean up state
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
          this.selectedFiles.push({ name: file.name, size: file.size, type: file.type, nativeFile: file, previewUrl: URL.createObjectURL(file) }); 
        } else {
           alert(`Skipping unsupported type: ${file.name}`);
        }
      }
    }
  }

  onDragOver(e: DragEvent): void { e.preventDefault(); this.dragOver = true; }

  onDragLeave(): void { this.dragOver = false; }
  
  onDrop(e: DragEvent): void { 
     e.preventDefault(); 
     this.dragOver = false; 
     
     if (e.dataTransfer?.files) {
       const droppedFiles = Array.from(e.dataTransfer.files);
       
       // Validate types and add to selected files
       for(const file of droppedFiles) {
         if(UploadService.isValidMediaType(file)) {
           this.selectedFiles.push({ name: file.name, size: file.size, type: file.type, nativeFile: file, previewUrl: URL.createObjectURL(file) }); 
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
    
    // If no files left, reset progress events to keep UI clean
     if(this.progressEvents.length > index + 1 || !this.isUploading) { /* optional cleanup */ }
  }

  getPreviewUrl(file: SelectableFile): string { 
      return file.previewUrl || ''; 
  }

  // Helper method for template (avoids inline TS type assertions in interpolation)
  getUploadedIds(): string {
    if (!this.response?.uploaded) return '';
    return this.response.uploaded.map((item: any) => item.id).join(', ');
  }

  startUpload(): void {
     if(this.selectedFiles.length === 0) return;

     this.isUploading = true;
     this.uploadComplete = false;
     
     const filesToUpload: SelectableFile[] = [...this.selectedFiles]; // Snapshot
     
     // Initialize progress events for each file so they show up immediately in the list 
     this.progressEvents = [];
      for (const f of filesToUpload) {
        this.progressEvents.push({ fileName: f.name, status: 'uploading', progress: 0 });
      }

    const totalFiles = filesToUpload.length;
    
    // We upload them in a batch via FormData. 
    // The backend returns one JSON response for the whole batch.
     this.uploadService.uploadFiles(filesToUpload as any).subscribe({
       next: (event) => {
         if (event.type === HttpEventType.UploadProgress && event.total) {
           const percent = Math.round((event.loaded / event.total) * 100);

            // Update progress using immutable updates to prevent Angular change detection freezes 
             this.progressEvents = this.progressEvents.map(p => ({ ...p, status: 'uploading', progress: Math.min(100, (percent / totalFiles)) }));
             
              if(totalFiles > 0) {
                 const names = filesToUpload.slice(0, 2).map(f => f.name).join(', ');
                  this.currentFileName = `Uploading: ${names}${totalFiles > 2 ? '...' : ''}`;
               }

         } else if (event.type === HttpEventType.Response && event.body) {
            // Upload completed successfully - process the response data 
             const resData = event.body as any;
             
             this.progressEvents = this.progressEvents.map(p => ({ ...p, status: 'completed', progress: 100 }));

             this.response = resData; 

             if (this.response && Object.keys(this.response).length > 0) {
               console.log('Upload successful:', this.response.uploaded?.length || 0, 'uploaded,', 
                           (this.response.skipped_duplicates as any[])?.length || 0, 'skipped.');
                 window.dispatchEvent(new CustomEvent('media-upload-complete'));
             } else {
               console.warn('Upload returned empty response');
              this.hasError = true;
              this.uploadMessage = 'Server did not return expected data.';
            }

         } else if (event.type === HttpEventType.Response && !event.body) { 
           // Empty success response from server  
             this.progressEvents = this.progressEvents.map(p => ({ ...p, status: 'completed', progress: 100 }));
              console.log('Upload completed with empty body');

         } else if (event.type === HttpEventType.UploadProgress && !event.total) { 
           // Fallback for edge cases where total isn't available yet  
             this.progressEvents = this.progressEvents.map(p => ({ ...p, status: 'uploading', progress: 0 }));
           }
       },

       
       error: (err) => { 
          console.error('Unexpected upload stream error:', err);
            // This catches network drops or HTTP errors that bypass the service's internal try/catch if any.
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

   // Changed from getter to property so it can be assigned dynamically in error/success states.
   uploadMessage = 'Media uploaded successfully!'; 
}
