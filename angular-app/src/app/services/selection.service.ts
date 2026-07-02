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

  selectAll(ids: string[]): void {
    const newSet = new Set<string>();
    ids.forEach(id => newSet.add(id));
    this._selectedIds$.next(newSet);
  }

  clear(): void {
    this._selectedIds$.next(new Set());
  }

  triggerSelectAll(): void {
    console.log('[DEBUG] SelectionService.triggerSelectAll() called');
    this._selectAllTrigger$.next();
    console.log('[DEBUG] Next emitted to _selectAllTrigger$');
  }

  isAllSelected(ids: string[]): boolean {
    return ids.every(id => this.isSelected(id));
  }
}
