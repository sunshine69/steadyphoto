import { Component, inject, ChangeDetectionStrategy } from '@angular/core';

import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../../services/auth.service';

@Component({
    selector: 'app-settings',
    imports: [FormsModule],
    templateUrl: './settings.component.html',
    changeDetection: ChangeDetectionStrategy.Eager,
    styles: [`
    .settings-wrapper {
      min-height: calc(100vh - 64px); /* Subtract sidebar height */
      padding: 2rem;
      background-color: #0f172a;
    }

    .settings-container {
      max-width: 800px;
      margin: 0 auto;
    }

    .page-title {
      font-size: 2rem;
      font-weight: 700;
      color: #f1f5f9;
      margin-bottom: 2rem;
    }

    /* Tab Navigation */
    .tab-nav {
      display: flex;
      gap: 0.5rem;
      margin-bottom: 2rem;
      border-bottom: 2px solid #334155;
    }

    .tab-button {
      padding: 0.75rem 1.5rem;
      background-color: transparent;
      color: #94a3b8;
      border: none;
      border-bottom: 2px solid transparent;
      font-size: 1rem;
      font-weight: 500;
      cursor: pointer;
      transition: all 0.2s ease;
    }

    .tab-button:hover {
      color: #e2e8f0;
    }

    .tab-button.active {
      color: #6366f1;
      border-bottom-color: #6366f1;
    }

    /* Tab Content */
    .tab-content {
      display: none;
    }

    .tab-content.active {
      display: block;
    }

    /* Section Styles */
    .section {
      background-color: #1e293b;
      border-radius: 16px;
      padding: 2rem;
      margin-bottom: 2rem;
      box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
      border: 1px solid #334155;
    }

    .section-title {
      font-size: 1.25rem;
      font-weight: 600;
      color: #f1f5f9;
      margin-bottom: 1rem;
    }

    .section-description {
      color: #94a3b8;
      font-size: 0.875rem;
      margin-bottom: 1.5rem;
    }

    /* Form Styles */
    .form-group {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
      margin-bottom: 1.25rem;
    }

    .form-label {
      font-size: 0.875rem;
      font-weight: 500;
      color: #cbd5e1;
      margin: 0;
    }

    .form-control {
      width: 100%;
      padding: 0.75rem 1rem;
      background-color: #0f172a;
      border: 1px solid #334155;
      border-radius: 8px;
      color: #f1f5f9;
      font-size: 0.9375rem;
      transition: all 0.2s ease;
      outline: none;
    }

    .form-control::placeholder {
      color: #64748b;
    }

    .form-control:focus {
      border-color: #6366f1;
      box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.15);
    }

    /* Error Message */
    .error-message {
      background-color: rgba(239, 68, 68, 0.1);
      border: 1px solid rgba(239, 68, 68, 0.3);
      color: #fca5a5;
      padding: 0.75rem 1rem;
      border-radius: 8px;
      font-size: 0.875rem;
    }

    /* Success Message */
    .success-message {
      background-color: rgba(34, 197, 94, 0.1);
      border: 1px solid rgba(34, 197, 94, 0.3);
      color: #86efac;
      padding: 0.75rem 1rem;
      border-radius: 8px;
      font-size: 0.875rem;
    }

    /* Action Buttons */
    .form-actions {
      display: flex;
      gap: 0.75rem;
      margin-top: 1.5rem;
    }

    .btn-save {
      padding: 0.875rem 2rem;
      background-color: #6366f1;
      color: white;
      border: none;
      border-radius: 8px;
      font-size: 0.9375rem;
      font-weight: 600;
      cursor: pointer;
      transition: all 0.2s ease;
    }

    .btn-save:hover:not(:disabled) {
      background-color: #4f46e5;
      transform: translateY(-1px);
    }

    .btn-save:active:not(:disabled) {
      transform: translateY(0);
    }

    .btn-save:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }

    /* Info Badge */
    .info-badge {
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.375rem 0.75rem;
      background-color: rgba(99, 102, 241, 0.1);
      border-radius: 6px;
      font-size: 0.875rem;
      color: #a5b4fc;
    }

    /* Password Strength Indicator */
    .password-strength {
      margin-top: 0.5rem;
      display: flex;
      gap: 0.25rem;
    }

    .strength-bar {
      height: 4px;
      width: 100%;
      background-color: #334155;
      border-radius: 2px;
      transition: all 0.3s ease;
    }

    .strength-bar.active {
      background-color: #6366f1;
    }

    /* Danger Zone */
    .danger-zone {
      margin-top: 2rem;
      padding-top: 2rem;
      border-top: 1px solid #334155;
    }

    .btn-danger {
      padding: 0.875rem 2rem;
      background-color: rgba(239, 68, 68, 0.1);
      color: #fca5a5;
      border: 1px solid rgba(239, 68, 68, 0.3);
      border-radius: 8px;
      font-size: 0.9375rem;
      font-weight: 600;
      cursor: pointer;
      transition: all 0.2s ease;
    }

    .btn-danger:hover:not(:disabled) {
      background-color: rgba(239, 68, 68, 0.2);
      border-color: rgba(239, 68, 68, 0.5);
    }

    .btn-danger:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }

    /* Required field indicator */
    .required-indicator {
      color: #ef4444;
      margin-left: 2px;
    }

    /* Validation states for form fields */
    .form-control.invalid {
      border-color: rgba(239, 68, 68, 0.5);
      box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.1);
    }

    .form-control.valid {
      border-color: rgba(34, 197, 94, 0.5);
      box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.1);
    }

    .field-error {
      color: #fca5a5;
      font-size: 0.8rem;
      margin-top: 0.25rem;
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }

    .field-error svg {
      width: 14px;
      height: 14px;
      flex-shrink: 0;
    }
  `]
})
export class SettingsComponent {
  private authService = inject(AuthService);
  private router = inject(Router);

  activeTab = 'profile';

  // Profile tab data
  profileEmail = '';
  profileLoading = false;
  profileMessage = '';
  profileMessageType: 'error' | 'success' = 'error';

  // Password change tab data
  currentPassword = '';
  newPassword = '';
  confirmPassword = '';
  passwordLoading = false;
  passwordMessage = '';
  passwordMessageType: 'error' | 'success' = 'error';
  
  // Validation states for form fields
  showCurrentPasswordError = false;
  showNewPasswordError = false;
  showConfirmPasswordError = false;

  ngOnInit(): void {
    this.loadProfile();
  }

  // Load current user profile
  loadProfile(): void {
    this.profileLoading = true;
    this.authService.getProfile().subscribe({
      next: (profile) => {
        this.profileEmail = profile.email || '';
        this.profileLoading = false;
      },
      error: (err) => {
        console.error('Failed to load profile', err);
        this.profileMessage = 'Failed to load your profile information.';
        this.profileMessageType = 'error';
        this.profileLoading = false;
      }
    });
  }

  // Update email address
  updateEmail(): void {
    if (!this.profileEmail || !this.profileEmail.includes('@')) {
      this.profileMessage = 'Please enter a valid email address.';
      this.profileMessageType = 'error';
      return;
    }

    this.profileLoading = true;
    this.profileMessage = '';

    this.authService.updateEmail(this.profileEmail).subscribe({
      next: () => {
        // Update local storage with new email
        localStorage.setItem('email', this.profileEmail);
        this.profileMessage = 'Your email address has been updated successfully.';
        this.profileMessageType = 'success';
        this.profileLoading = false;
        
        // Clear message after 3 seconds
        setTimeout(() => {
          if (this.profileMessageType === 'success') {
            this.profileMessage = '';
          }
        }, 3000);
      },
      error: (err) => {
        console.error('Failed to update email', err);
        if (err.status === 409) {
          this.profileMessage = 'This email is already in use by another account.';
        } else {
          this.profileMessage = 'Failed to update your email address. Please try again.';
        }
        this.profileMessageType = 'error';
        this.profileLoading = false;
      }
    });
  }

  // Calculate password strength (1-4)
  calculatePasswordStrength(password: string): number {
    let strength = 0;
    
    if (password.length >= 8) strength++;
    if (/[A-Z]/.test(password)) strength++;
    if (/[a-z]/.test(password)) strength++;
    if (/[0-9]/.test(password)) strength++;
    if (/[^A-Za-z0-9]/.test(password)) strength++; // Special character
    
    return Math.min(strength, 4);
  }

  // Validate form before submission
  validatePasswordForm(): boolean {
    let isValid = true;
    
    this.showCurrentPasswordError = false;
    this.showNewPasswordError = false;
    this.showConfirmPasswordError = false;

    if (!this.currentPassword) {
      this.passwordMessage = 'Please enter your current password.';
      this.passwordMessageType = 'error';
      this.showCurrentPasswordError = true;
      isValid = false;
    } else if (this.newPassword.length < 8) {
      this.passwordMessage = 'New password must be at least 8 characters long.';
      this.passwordMessageType = 'error';
      this.showNewPasswordError = true;
      isValid = false;
    } else if (this.newPassword !== this.confirmPassword) {
      this.passwordMessage = 'Passwords do not match.';
      this.passwordMessageType = 'error';
      this.showConfirmPasswordError = true;
      isValid = false;
    }

    return isValid;
  }

  // Handle password change
  changePassword(): void {
    this.passwordMessage = '';
    
    // Validate first before attempting to submit
    if (!this.validatePasswordForm()) {
      return;
    }

    this.passwordLoading = true;

    this.authService.changePassword(this.currentPassword, this.newPassword).subscribe({
      next: () => {
        // Clear password fields
        this.currentPassword = '';
        this.newPassword = '';
        this.confirmPassword = '';
        
        // Clear validation states
        this.showCurrentPasswordError = false;
        this.showNewPasswordError = false;
        this.showConfirmPasswordError = false;
        
        this.passwordMessage = 'Your password has been changed successfully.';
        this.passwordMessageType = 'success';
        this.passwordLoading = false;
        
        // Clear message after 3 seconds
        setTimeout(() => {
          if (this.passwordMessageType === 'success') {
            this.passwordMessage = '';
          }
        }, 3000);
      },
      error: (err) => {
        console.error('Failed to change password', err);
        // Clear validation states on server error
        this.showCurrentPasswordError = false;
        this.showNewPasswordError = false;
        this.showConfirmPasswordError = false;
        
        if (err.status === 401) {
          this.passwordMessage = 'Current password is incorrect.';
          this.passwordMessageType = 'error';
        } else {
          this.passwordMessage = 'Failed to change your password. Please try again.';
        }
        this.passwordLoading = false;
      }
    });
  }

  // Handle tab selection
  selectTab(tab: string): void {
    this.activeTab = tab;
  }

  // Navigate back to home page
  goHome(): void {
    this.router.navigate(['/']);
  }
}
