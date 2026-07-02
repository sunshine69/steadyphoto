import { Injectable } from '@angular/core';
import { BehaviorSubject, Subject, Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class SelectionService {
  private _selectedIds$ = new BehaviorSubject<Set<string>>(new Set());
  private _selectAllTrigger$ = new Subject<void>();

  public selectedIds$ = this._selectedIds$.asObservable();
  public selectAllTrigger$ = this._selectAllTrigger$.asObservable();

  get count(): number {
    return this._selectedIds$.value.size;
  }

  isSelected(id: string): boolean {
    return this._selectedIds$.value.has(id);
  }

  add(id: string): void {
    const ids = new Set(this._selectedIds$.value);
    ids.add(id);
    this._selectedIds$.next(ids);
  }

  remove(id: string): void {
    const ids = new Set(this._selectedIds$.value);
    ids.delete(id);
    this._selectedIds$.next(ids);
  }

  toggle(id: string): void {
    if (this.isSelected(id)) {
      this.remove(id);
    } else {
      this.add(id);
    }
  }

  /**
   * Add all the given IDs to the selection, preserving any previously
   * selected items.  This is the correct behaviour for a "Select All on
   * current page" action — it should accumulate rather than replace.
   */
  selectAll(ids: string[]): void {
    const newSet = new Set(this._selectedIds$.value);
    ids.forEach(id => newSet.add(id));
    this._selectedIds$.next(newSet);
  }

  clear(): void {
    this._selectedIds$.next(new Set());
  }

  triggerSelectAll(): void {
    this._selectAllTrigger$.next();
  }

  isAllSelected(ids: string[]): boolean {
    return ids.every(id => this.isSelected(id));
  }
}
