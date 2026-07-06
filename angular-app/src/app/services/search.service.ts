import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, BehaviorSubject, Subject } from 'rxjs';
import { map } from 'rxjs/operators';
import { environment } from '../../environments/environment';
import { Photo } from '../models/photo.model';
import { PhotoService } from './photo.service';

export type SearchScope = 'all' | 'name' | 'tags' | 'date' | 'location';

export interface SearchResponse {
  results: Photo[];
  total: number;
  limit: number;
  offset: number;
}

@Injectable({
  providedIn: 'root'
})
export class SearchService {
  private http = inject(HttpClient);
  private photoService = inject(PhotoService);
  private API_BASE_URL = environment.apiBaseUrl;

  private searchTermSource = new BehaviorSubject<string>('');
  private searchScopeSource = new BehaviorSubject<SearchScope>('all');
  private searchDateSource = new BehaviorSubject<string>('');

  searchTerm$ = this.searchTermSource.asObservable();
  searchScope$ = this.searchScopeSource.asObservable();
  searchDate$ = this.searchDateSource.asObservable();

  setSearchTerm(term: string) {
    this.searchTermSource.next(term);
    console.log('[SEARCH-SERVICE] setSearchTerm:', term);
  }

  setSearchScope(scope: SearchScope) {
    this.searchScopeSource.next(scope);
    console.log('[SEARCH-SERVICE] setSearchScope:', scope);
  }

  setSearchDate(dateRange: string) {
    this.searchDateSource.next(dateRange);
    console.log('[SEARCH-SERVICE] setSearchDate:', dateRange);
  }

  /**
   * Triggers an actual search call to the backend.
   */
  triggerSearch(query: string, scope: SearchScope = 'all', dateRange?: string): void {
    console.log('[SEARCH-SERVICE] triggerSearch() called with:', { query, scope, dateRange });
    
    // Update streams IMMEDIATELY so photo-list can react and call loadPhotos()
    // BEFORE the HTTP response arrives
    this.searchTermSource.next(query);
    this.searchScopeSource.next(scope);
    this.searchDateSource.next(dateRange || '');
    console.log('[SEARCH-SERVICE]   streams updated:', { query, scope, dateRange: dateRange || '' });
    
    this.searchMedia(query, scope, 20, 0, dateRange).subscribe({
      next: (response) => {
        console.log('[SEARCH-SERVICE] triggerSearch() results:', response.results.length, 'items, total:', response.total);
        // Update streams again with final values (redundant but safe)
        this.searchTermSource.next(query);
        this.searchScopeSource.next(scope);
        this.searchDateSource.next(dateRange || '');
      },
      error: (err) => {
        console.error('[SEARCH-SERVICE] triggerSearch() error:', err);
      }
    });
  }

  /**
   * Normalizes raw backend media data to include thumbnailUrl and proper field mapping.
   * The backend Media struct doesn't include thumbnailUrl, so we construct it from the ID.
   */
  private normalizeMedia(p: any): Photo {
    const id = p.ID ?? p.id ?? '';
    let mediaType: 'photo' | 'video' | undefined;
    const rawMediaType = p.MediaType || p.mediaType || p.media_type;
    if (rawMediaType) {
      const mt = String(rawMediaType).toLowerCase();
      mediaType = mt === 'video' ? 'video' : 'photo';
    }

    return {
      id: id,
      path: id ? `${this.API_BASE_URL}/media/${id}/original` : '',
      thumbnailUrl: id ? `${this.API_BASE_URL}/media/${id}/thumb` : '',
      filename: p.Filename ?? p.filename ?? '',
      captured_at: p.capturedAt ?? p.capturedAt ?? '',
      width: p.Width ?? p.width,
      height: p.Height ?? p.height,
      size: p.SizeBytes ?? p.sizeBytes ?? p.size,
      type: p.Type ?? p.type,
      mediaType: mediaType,
      tags: p.Tags ?? p.tags ?? '',
      metadata: p.Metadata ? this.photoService.normalizeMetadata(p.Metadata) : undefined,
      videoMetadata: p.VideoMetadata ? {
        duration: p.VideoMetadata.Duration,
        bitrate: p.VideoMetadata.Bitrate,
        video_codec: p.VideoMetadata.VideoCodec,
        audio_codec: p.VideoMetadata.AudioCodec,
        frame_rate: p.VideoMetadata.FrameRate,
      } : undefined,
    };
  }

  /**
   * Searches media by text across filename, tags, and metadata.
   * @param query Search text
   * @param scope Search scope: 'all' (default), 'name', 'tags', 'date'
   * @param limit Number of results per page
   * @param offset Pagination offset
   * @param dateRange Date range string in format: "dd/mm/yyyy", "yyyy/mm/dd", "dd/mm/yyyy - dd/mm/yyyy", etc.
   */
  searchMedia(query: string, scope: SearchScope = 'all', limit: number = 20, offset: number = 0, dateRange?: string): Observable<SearchResponse> {
    console.log('[SEARCH-SERVICE] searchMedia called with:', { query, scope, limit, offset, dateRange });
    const params = new URLSearchParams({
      query: query,
      scope: scope,
      limit: limit.toString(),
      offset: offset.toString()
    });

    if (dateRange) {
      params.append('dateRange', dateRange);
    }

    const url = `${this.API_BASE_URL}/media/search?${params}`;
    console.log('[SEARCH-SERVICE] Request URL:', url);

    return this.http.get<any>(url).pipe(
      map(response => {
        console.log('[SEARCH-SERVICE] Response received:', response);
        return {
          results: (response.results || []).map((p: any) => this.normalizeMedia(p)),
          total: response.total ?? 0,
          limit: response.limit ?? limit,
          offset: response.offset ?? offset,
        };
      })
    );
  }
}
