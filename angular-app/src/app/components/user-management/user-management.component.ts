import { Component, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subscription } from 'rxjs';
import { AuthService } from '../../services/auth.service';
import { UserManagementService, User } from '../../services/user-management.service';

@Component({
    selector: 'app-user-management',
    imports: [CommonModule, FormsModule],
    template: `
    <div class="user-management-overlay" *ngIf="isOpen" (click)="onOverlayClick($event)">
      <div class="user-management-modal" (click)="$event.stopPropagation()">
        <!-- Header -->
        <div class="modal-header">
          <h2>User Management</h2>
          <button class="close-btn" (click)="close()" title="Close">×</button>
        </div>

        <!-- Filter Section -->
        <div class="filter-section">
          <label>Filter by Status:</label>
          <select [(ngModel)]="selectedStatus" (change)="loadUsers()">
            <option value="">All Users</option>
            <option value="pending">Pending Approval</option>
            <option value="active">Active</option>
            <option value="disabled">Disabled</option>
            <option value="rejected">Rejected</option>
          </select>
        </div>

        <!-- Bulk Actions Bar -->
        <div class="bulk-actions-bar" *ngIf="isAdmin && users.length > 0 && !isLoading">
          <label class="select-all-label">
            <input type="checkbox" (change)="toggleSelectAll($event)" [checked]="isAllSelected()" />
            Select All
          </label>
          <span class="selected-count">{{ getSelectedCount() }} selected</span>
          
          <div class="bulk-buttons">
            <button 
              (click)="bulkApproveSelected()"
              [disabled]="getSelectedCount() === 0 || isBulkUpdating"
              class="btn btn-sm btn-success bulk-btn"
              title="Approve all selected users">
              ✓ Approve Selected
            </button>
            
            <button 
              (click)="bulkDisableSelected()"
              [disabled]="getSelectedCount() === 0 || isBulkUpdating"
              class="btn btn-sm btn-warning bulk-btn"
              title="Disable all selected users">
              ⏸ Disable Selected
            </button>
            
            <button 
              (click)="bulkDeleteSelected()"
              [disabled]="getSelectedCount() === 0 || isBulkUpdating"
              class="btn btn-sm btn-danger bulk-btn"
              title="Permanently delete all selected users">
              🗑 Delete Selected
            </button>
          </div>
        </div>

        <!-- Loading State -->
        <div class="loading-state" *ngIf="isLoading">
          <div class="spinner-border text-primary" role="status">
            <span class="visually-hidden">Loading...</span>
          </div>
          <p>Loading users...</p>
        </div>

        <!-- Error State -->
        <div class="error-state" *ngIf="errorMessage">
          <p class="text-danger">{{ errorMessage }}</p>
          <button (click)="loadUsers()" class="btn btn-sm btn-outline-primary mt-2">Retry</button>
        </div>

        <!-- User List -->
        <div class="user-list" *ngIf="!isLoading && !errorMessage">
          <div class="user-count">
            Showing {{ users.length }} user(s)
          </div>

          <div class="users-container">
            <div *ngFor="let user of users; let i = index" class="user-card" [class.selected]="selectedUsers[user.id]">
              <!-- Selection Checkbox -->
              <div class="selection-checkbox">
                <input 
                  type="checkbox" 
                  [checked]="selectedUsers[user.id]"
                  (change)="toggleUserSelection(user)"
                  *ngIf="isAdmin" />
              </div>

              <div class="user-info">
                <div class="user-avatar">{{ user.email[0].toUpperCase() }}</div>
                <div class="user-details">
                  <div class="user-email">{{ user.email }}</div>
                  <div class="user-meta">
                    <span class="badge" [ngClass]="getStatusBadgeClass(user.status)">
                      {{ formatStatus(user.status) }}
                    </span>
                    <span class="badge" [ngClass]="getRoleBadgeClass(user.role)">
                      {{ user.role || 'User' }}
                    </span>
                  </div>
                </div>
              </div>

              <!-- Actions -->
              <div class="user-actions">
                <select 
                  *ngIf="isAdmin" 
                  [(ngModel)]="pendingStatus[user.id]" 
                  (change)="onStatusChange(user)"
                  class="form-select form-select-sm status-dropdown"
                  [disabled]="isUpdating[user.id] || isBulkUpdating">
                  <option value="">Update Status</option>
                  <option value="active">Activate</option>
                  <option value="disabled">Disable</option>
                  <option value="pending">Pending</option>
                  <option value="rejected">Reject</option>
                </select>

                <select 
                  *ngIf="isAdmin" 
                  [(ngModel)]="pendingRole[user.id]" 
                  (change)="onRoleChange(user)"
                  class="form-select form-select-sm role-dropdown"
                  [disabled]="isUpdating[user.id] || isBulkUpdating">
                  <option value="">Update Role</option>
                  <option value="admin">Make Admin</option>
                  <option value="user">Make User</option>
                </select>

                <button 
                  *ngIf="isAdmin && user.status !== 'disabled'" 
                  (click)="deleteUser(user)"
                  class="btn btn-sm btn-danger"
                  [disabled]="isUpdating[user.id] || isBulkUpdating">
                  Delete
                </button>

                <!-- Update in Progress -->
                <span *ngIf="isUpdating[user.id]" class="updating-indicator">
                  <div class="spinner-border spinner-border-sm text-primary" role="status"></div>
                </span>
              </div>
            </div>
          </div>

          <!-- Empty State -->
          <div *ngIf="users.length === 0" class="empty-state">
            <p>No users found matching the selected filter.</p>
          </div>
        </div>
      </div>
    </div>
  `,
    styles: [`
    .user-management-overlay {
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background-color: rgba(0, 0, 0, 0.7);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 1050;
    }

    .user-management-modal {
      background-color: #1e293b;
      border-radius: 12px;
      width: 90%;
      max-width: 800px;
      max-height: 80vh;
      overflow-y: auto;
      box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
    }

    .modal-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 20px 24px;
      border-bottom: 1px solid #374151;
    }

    .modal-header h2 {
      margin: 0;
      color: #e5e7eb;
      font-size: 1.5rem;
    }

    .close-btn {
      background: transparent;
      border: none;
      color: #9ca3af;
      font-size: 28px;
      cursor: pointer;
      padding: 0;
      width: 40px;
      height: 40px;
      display: flex;
      align-items: center;
      justify-content: center;
      border-radius: 8px;
      transition: all 0.2s ease;
    }

    .close-btn:hover {
      background-color: rgba(99, 102, 241, 0.1);
      color: #e5e7eb;
    }

    .filter-section {
      padding: 16px 24px;
      border-bottom: 1px solid #374151;
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .filter-section label {
      color: #9ca3af;
      font-size: 14px;
      white-space: nowrap;
    }

    .filter-section select {
      flex: 1;
      padding: 8px 12px;
      background-color: #2d3748;
      border: 1px solid #374151;
      border-radius: 6px;
      color: #e5e7eb;
      font-size: 14px;
      outline: none;
    }

    .filter-section select:focus {
      border-color: #6366f1;
    }

    .bulk-actions-bar {
      padding: 12px 24px;
      border-bottom: 1px solid #374151;
      display: flex;
      align-items: center;
      gap: 16px;
      background-color: #1e293b;
    }

    .select-all-label {
      color: #9ca3af;
      font-size: 14px;
      display: flex;
      align-items: center;
      gap: 8px;
      cursor: pointer;
    }

    .select-all-label input[type="checkbox"] {
      width: 16px;
      height: 16px;
      cursor: pointer;
    }

    .selected-count {
      color: #6366f1;
      font-size: 14px;
      font-weight: 500;
    }

    .bulk-buttons {
      display: flex;
      gap: 8px;
      margin-left: auto;
    }

    .bulk-btn {
      padding: 6px 12px;
      font-size: 13px;
      white-space: nowrap;
    }

    .user-card.selected {
      background-color: #374151;
      border-left: 3px solid #6366f1;
    }

    .selection-checkbox {
      margin-right: 12px;
      display: flex;
      align-items: center;
    }

    .selection-checkbox input[type="checkbox"] {
      width: 18px;
      height: 18px;
      cursor: pointer;
      accent-color: #6366f1;
    }

    .loading-state, .error-state {
      padding: 40px 24px;
      text-align: center;
      color: #9ca3af;
    }

    .user-list {
      padding: 16px 24px 24px;
    }

    .user-count {
      color: #6b7280;
      font-size: 13px;
      margin-bottom: 12px;
    }

    .users-container {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .user-card {
      background-color: #2d3748;
      border-radius: 8px;
      padding: 16px;
      transition: all 0.2s ease;
    }

    .user-card:hover {
      background-color: #374151;
    }

    .user-info {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 12px;
    }

    .user-avatar {
      width: 40px;
      height: 40px;
      border-radius: 50%;
      background: linear-gradient(135deg, #6366f1, #8b5cf6);
      display: flex;
      align-items: center;
      justify-content: center;
      color: white;
      font-weight: bold;
      font-size: 16px;
      flex-shrink: 0;
    }

    .user-details {
      flex: 1;
      min-width: 0;
    }

    .user-email {
      color: #e5e7eb;
      font-weight: 500;
      margin-bottom: 4px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .user-meta {
      display: flex;
      gap: 8px;
      align-items: center;
    }

    .badge {
      padding: 4px 10px;
      border-radius: 12px;
      font-size: 12px;
      font-weight: 500;
    }

    .badge.bg-success {
      background-color: #10b981;
      color: white;
    }

    .badge.bg-warning {
      background-color: #f59e0b;
      color: white;
    }

    .badge.bg-danger {
      background-color: #ef4444;
      color: white;
    }

    .badge.bg-secondary {
      background-color: #6b7280;
      color: white;
    }

    .badge.bg-info {
      background-color: #3b82f6;
      color: white;
    }

    .user-actions {
      display: flex;
      gap: 8px;
      align-items: center;
      flex-wrap: wrap;
    }

    .status-dropdown, .role-dropdown {
      flex: 1;
      min-width: 120px;
    }

    .btn-danger {
      padding: 6px 12px;
      font-size: 13px;
    }

    .updating-indicator {
      margin-left: auto;
    }

    .empty-state {
      text-align: center;
      padding: 40px 0;
      color: #6b7280;
    }

    @media (max-width: 640px) {
      .user-management-modal {
        width: 95%;
        max-height: 90vh;
      }

      .user-actions {
        flex-direction: column;
        align-items: stretch;
      }

      .status-dropdown, .role-dropdown {
        min-width: auto;
      }
    }
  `]
})
export class UserManagementComponent implements OnInit, OnDestroy {
  isOpen = false;
  users: User[] = [];
  isLoading = false;
  errorMessage = '';
  selectedStatus = '';
  isAdmin = false;

  private isAdminSubscription?: Subscription;

  // Track pending changes for each user
  pendingStatus: { [key: string]: string } = {};
  pendingRole: { [key: string]: string } = {};
  isUpdating: { [key: string]: boolean } = {};
  
  // Bulk operation state
  selectedUsers: { [key: string]: boolean } = {};
  isBulkUpdating = false;

  private userManagementService = inject(UserManagementService);
  private authService = inject(AuthService);

  ngOnInit(): void {
    // Subscribe to admin status changes — ensures isAdmin updates dynamically on login/logout
    this.isAdminSubscription = this.authService.isAdmin$.subscribe(isAdmin => {
      this.isAdmin = isAdmin;
    });
  }

  open(): void {
    this.isOpen = true;
    this.loadUsers();
  }

  close(): void {
    this.isOpen = false;
    this.selectedStatus = '';
    this.users = [];
    this.errorMessage = '';
    this.pendingStatus = {};
    this.pendingRole = {};
    this.isUpdating = {};
  }

  onOverlayClick(event: MouseEvent): void {
    if (event.target === event.currentTarget) {
      this.close();
    }
  }

  loadUsers(): void {
    this.isLoading = true;
    this.errorMessage = '';
    
    this.userManagementService.getUsers(this.selectedStatus || undefined).subscribe({
      next: (users) => {
        this.users = users;
        this.pendingStatus = {};
        this.pendingRole = {};
        this.isUpdating = {};
        this.isLoading = false;
      },
      error: (err) => {
        console.error('Failed to load users', err);
        this.errorMessage = 'Failed to load users. Please try again.';
        this.isLoading = false;
      }
    });
  }

  onStatusChange(user: User): void {
    const newStatus = this.pendingStatus[user.id];
    if (!newStatus) return;

    this.isUpdating[user.id] = true;
    
    this.userManagementService.updateUser(user.id, { status: newStatus }).subscribe({
      next: (updatedUser) => {
        // Update the user in the list
        const index = this.users.findIndex(u => u.id === user.id);
        if (index !== -1) {
          this.users[index] = updatedUser;
        }
        delete this.pendingStatus[user.id];
      },
      error: (err) => {
        console.error('Failed to update user status', err);
        alert('Failed to update user status. Please try again.');
        delete this.pendingStatus[user.id];
      },
      complete: () => {
        this.isUpdating[user.id] = false;
      }
    });
  }

  onRoleChange(user: User): void {
    const newRole = this.pendingRole[user.id];
    if (!newRole) return;

    this.isUpdating[user.id] = true;
    
    this.userManagementService.updateUser(user.id, { role: newRole }).subscribe({
      next: (updatedUser) => {
        // Update the user in the list
        const index = this.users.findIndex(u => u.id === user.id);
        if (index !== -1) {
          this.users[index] = updatedUser;
        }
        delete this.pendingRole[user.id];
      },
      error: (err) => {
        console.error('Failed to update user role', err);
        alert('Failed to update user role. Please try again.');
        delete this.pendingRole[user.id];
      },
      complete: () => {
        this.isUpdating[user.id] = false;
      }
    });
  }

  deleteUser(user: User): void {
    if (!confirm(`Are you sure you want to disable user "${user.email}"?`)) {
      return;
    }

    this.isUpdating[user.id] = true;
    
    this.userManagementService.deleteUser(user.id).subscribe({
      next: () => {
        // Remove the user from the list
        this.users = this.users.filter(u => u.id !== user.id);
        delete this.pendingStatus[user.id];
        delete this.pendingRole[user.id];
      },
      error: (err) => {
        console.error('Failed to delete user', err);
        alert('Failed to disable user. Please try again.');
      },
      complete: () => {
        this.isUpdating[user.id] = false;
      }
    });
  }

  // Bulk operation methods
  getSelectedCount(): number {
    return Object.values(this.selectedUsers).filter(Boolean).length;
  }

  isAllSelected(): boolean {
    if (this.users.length === 0) return false;
    return this.users.every(user => this.selectedUsers[user.id]);
  }

  hasPendingUsers(): boolean {
    return this.users.some(u => u.status === 'pending');
  }

  hasActiveUsers(): boolean {
    return this.users.some(u => u.status === 'active');
  }

  hasActiveOrPendingUsers(): boolean {
    return this.hasActiveUsers() || this.hasPendingUsers();
  }

  toggleSelectAll(event: Event): void {
    const target = event.target as HTMLInputElement;
    const selectAll = target.checked;
    
    this.users.forEach(user => {
      if (user.status !== 'disabled') {
        this.selectedUsers[user.id] = selectAll;
      }
    });
  }

  toggleUserSelection(user: User): void {
    if (user.status === 'disabled') return;
    this.selectedUsers[user.id] = !this.selectedUsers[user.id];
  }

  bulkApproveSelected(): void {
    const selectedIds = Object.entries(this.selectedUsers)
      .filter(([_, selected]) => selected)
      .map(([id, _]) => id);
    
    if (selectedIds.length === 0) return;

    if (!confirm(`Are you sure you want to approve ${selectedIds.length} user(s)?`)) {
      return;
    }

    this.isBulkUpdating = true;
    
    this.userManagementService.bulkApproveUsers(selectedIds).subscribe({
      next: (response) => {
        
        alert(response.message || 'Users approved successfully');
        this.loadUsers(); // Refresh the list
      },
      error: (err) => {
        console.error('Failed to bulk approve users', err);
        alert('Failed to approve users. Please try again.');
      },
      complete: () => {
        this.isBulkUpdating = false;
      }
    });
  }

  bulkDisableSelected(): void {
    const selectedIds = Object.entries(this.selectedUsers)
      .filter(([_, selected]) => selected)
      .map(([id, _]) => id);
    
    if (selectedIds.length === 0) return;

    if (!confirm(`Are you sure you want to disable ${selectedIds.length} user(s)?`)) {
      return;
    }

    this.isBulkUpdating = true;
    
    this.userManagementService.bulkDisableUsers(selectedIds).subscribe({
      next: (response) => {
        
        alert(response.message || 'Users disabled successfully');
        this.loadUsers(); // Refresh the list
      },
      error: (err) => {
        console.error('Failed to bulk disable users', err);
        alert('Failed to disable users. Please try again.');
      },
      complete: () => {
        this.isBulkUpdating = false;
      }
    });
  }

  bulkDeleteSelected(): void {
    const selectedIds = Object.entries(this.selectedUsers)
      .filter(([_, selected]) => selected)
      .map(([id, _]) => id);
    
    if (selectedIds.length === 0) return;

    if (!confirm(`⚠️ WARNING: Are you sure you want to permanently delete ${selectedIds.length} user(s)? This action cannot be undone!`)) {
      return;
    }

    this.isBulkUpdating = true;
    
    this.userManagementService.bulkDeleteUsers(selectedIds).subscribe({
      next: (response) => {
        
        alert(response.message || 'Users deleted successfully');
        this.loadUsers(); // Refresh the list
      },
      error: (err) => {
        console.error('Failed to bulk delete users', err);
        alert('Failed to delete users. Please try again.');
      },
      complete: () => {
        this.isBulkUpdating = false;
      }
    });
  }

  formatStatus(status: string): string {
    switch (status) {
      case 'pending': return 'Pending Approval';
      case 'active': return 'Active';
      case 'disabled': return 'Disabled';
      case 'rejected': return 'Rejected';
      default: return status;
    }
  }

  getStatusBadgeClass(status: string): string {
    switch (status) {
      case 'pending': return 'bg-warning';
      case 'active': return 'bg-success';
      case 'disabled': return 'bg-secondary';
      case 'rejected': return 'bg-danger';
      default: return 'bg-secondary';
    }
  }

  getRoleBadgeClass(role: string): string {
    switch (role) {
      case 'admin': return 'bg-info';
      default: return 'bg-secondary';
    }
  }

  ngOnDestroy(): void {
    this.close();
    this.isAdminSubscription?.unsubscribe();
  }
}
