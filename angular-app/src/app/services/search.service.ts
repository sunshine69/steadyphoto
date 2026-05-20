import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';

export type SearchScope = 'all' | 'name' | 'tags';

@Injectable({
  providedIn: 'root'
})
export class SearchService {
  private searchTermSubject = new BehaviorSubject<string>('');
  searchTerm$ = this.searchTermSubject.asObservable();

  private searchScopeSubject = new BehaviorSubject<SearchScope>('all');
  searchScope$ = this.searchScopeSubject.asObservable();

  setSearchTerm(term: string): void {
    this.searchTermSubject.next(term);
  }

  getSearchTerm(): string {
    return this.searchTermSubject.value;
  }

  setSearchScope(scope: SearchScope): void {
    this.searchScopeSubject.next(scope);
  }

  getSearchScope(): SearchScope {
    return this.searchScopeSubject.value;
  }
}
