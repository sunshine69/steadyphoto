import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, throwError } from 'rxjs';
import { catchError } from 'rxjs/operators';
import { environment } from '../../environments/environment';

export interface User {
  id: string;
  email: string;
  status: string; // pending, active, disabled, rejected
  role: string; // admin, user
  created_at?: string;
  updated_at?: string;
}

export interface AdminUpdateUserRequest {
  status?: string;
  role?: string;
}

export interface BulkOperationRequest {
  user_ids: string[];
}

export interface BulkOperationResponse {
  message: string;
  approved_count?: number;
  disabled_count?: number;
  deleted_count?: number;
}

@Injectable({
  providedIn: 'root'
})
export class UserManagementService {
  private http = inject(HttpClient);
  private API_BASE_URL = environment.apiBaseUrl;

  /**
   * Get all users (admin only)
   */
  getUsers(status?: string): Observable<User[]> {
    let url = `${this.API_BASE_URL}/admin/users`;
    if (status) {
      url += `?status=${status}`;
    }
    return this.http.get<User[]>(url).pipe(
      catchError(err => {
        console.error('UserManagementService: Failed to fetch users', err);
        return throwError(() => err);
      })
    );
  }

  /**
   * Get a specific user by ID (admin only)
   */
  getUserById(id: string): Observable<User> {
    const url = `${this.API_BASE_URL}/admin/users/${id}`;
    return this.http.get<User>(url).pipe(
      catchError(err => {
        console.error('UserManagementService: Failed to fetch user', err);
        return throwError(() => err);
      })
    );
  }

  /**
   * Update a user's status or role (admin only)
   */
  updateUser(id: string, updates: AdminUpdateUserRequest): Observable<User> {
    const url = `${this.API_BASE_URL}/admin/users/${id}`;
    return this.http.patch<User>(url, updates).pipe(
      catchError(err => {
        console.error('UserManagementService: Failed to update user', err);
        return throwError(() => err);
      })
    );
  }

  /**
   * Delete (disable) a user (admin only)
   */
  deleteUser(id: string): Observable<void> {
    const url = `${this.API_BASE_URL}/admin/users/${id}`;
    return this.http.delete<void>(url).pipe(
      catchError(err => {
        console.error('UserManagementService: Failed to delete user', err);
        return throwError(() => err);
      })
    );
  }

  /**
   * Register a new user (admin can create users directly)
   */
  registerUser(email: string, password: string): Observable<any> {
    const url = `${this.API_BASE_URL}/auth/register`;
    return this.http.post(url, { email, password }).pipe(
      catchError(err => {
        console.error('UserManagementService: Failed to register user', err);
        return throwError(() => err);
      })
    );
  }

  /**
   * Bulk approve users (set status to active) - admin only
   */
  bulkApproveUsers(userIds: string[]): Observable<BulkOperationResponse> {
    const url = `${this.API_BASE_URL}/admin/users/bulk-approve`;
    return this.http.post<BulkOperationResponse>(url, { user_ids: userIds }).pipe(
      catchError(err => {
        console.error('UserManagementService: Failed to bulk approve users', err);
        return throwError(() => err);
      })
    );
  }

  /**
   * Bulk disable users (set status to disabled) - admin only
   */
  bulkDisableUsers(userIds: string[]): Observable<BulkOperationResponse> {
    const url = `${this.API_BASE_URL}/admin/users/bulk-disable`;
    return this.http.post<BulkOperationResponse>(url, { user_ids: userIds }).pipe(
      catchError(err => {
        console.error('UserManagementService: Failed to bulk disable users', err);
        return throwError(() => err);
      })
    );
  }

  /**
   * Bulk delete users (permanent deletion) - admin only
   */
  bulkDeleteUsers(userIds: string[]): Observable<BulkOperationResponse> {
    const url = `${this.API_BASE_URL}/admin/users/bulk-delete`;
    return this.http.delete<BulkOperationResponse>(url, { body: { user_ids: userIds } }).pipe(
      catchError(err => {
        console.error('UserManagementService: Failed to bulk delete users', err);
        return throwError(() => err);
      })
    );
  }
}
