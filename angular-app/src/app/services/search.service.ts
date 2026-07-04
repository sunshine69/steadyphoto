import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, BehaviorSubject, Subject } from 'rxjs';
import { map } from 'rxjs/operators';
import { environment } from '../../environments/environment';
import { Photo } from '../models/photo.model';
import { PhotoService } from './photo.service';

export type SearchScope = 'all' | 'name' | 'tags';

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

  searchTerm$ = this.searchTermSource.asObservable();
  searchScope$ = this.searchScopeSource.asObservable();

  setSearchTerm(term: string) {
    this.searchTermSource.next(term);
  }

  setSearchScope(scope: SearchScope) {
    this.searchScopeSource.next(scope);
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
   * @param scope Search scope: 'all' (default), 'name', 'tags'
   * @param limit Number of results per page
   * @param offset Pagination offset
   */
  searchMedia(query: string, scope: SearchScope = 'all', limit: number = 20, offset: number = 0): Observable<SearchResponse> {
    const params = new URLSearchParams({
      query: query,
      scope: scope,
      limit: limit.toString(),
      offset: offset.toString()
    });

    return this.http.get<any>(`${this.API_BASE_URL}/media/search?${params}`).pipe(
      map(response => ({
        results: (response.results || []).map((p: any) => this.normalizeMedia(p)),
        total: response.total ?? 0,
        limit: response.limit ?? limit,
        offset: response.offset ?? offset,
      }))
    );
  }
}
