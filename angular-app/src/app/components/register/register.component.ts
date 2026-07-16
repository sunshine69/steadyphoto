import { Component, inject, ChangeDetectionStrategy } from '@angular/core';

import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../services/auth.service';

@Component({
    selector: 'app-register',
    imports: [FormsModule, RouterLink],
    templateUrl: './register.component.html',
    changeDetection: ChangeDetectionStrategy.Eager,
    styles: [`
    .register-wrapper {
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      background-color: #0f172a;
      padding: 2rem;
    }

    .register-card {
      width: 100%;
      max-width: 420px;
      background-color: #1e293b;
      border-radius: 16px;
      padding: 2.5rem;
      box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
      border: 1px solid #334155;
    }

    /* Logo Section */
    .logo-section {
      text-align: center;
      margin-bottom: 2rem;
    }

    .logo-icon {
      margin-bottom: 0.75rem;
    }

    .app-title {
      font-size: 1.875rem;
      font-weight: 700;
      color: #f1f5f9;
      margin: 0 0 0.25rem 0;
    }

    .app-subtitle {
      font-size: 0.875rem;
      color: #94a3b8;
      margin: 0;
    }

    /* Form Styles */
    .register-form {
      display: flex;
      flex-direction: column;
      gap: 1.25rem;
    }

    .form-group {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
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

    /* Action Buttons */
    .form-actions {
      display: flex;
      gap: 0.75rem;
      margin-top: 0.5rem;
    }

    .btn-register {
      flex: 1;
      padding: 0.875rem 1.5rem;
      background-color: #6366f1;
      color: white;
      border: none;
      border-radius: 8px;
      font-size: 0.9375rem;
      font-weight: 600;
      cursor: pointer;
      transition: all 0.2s ease;
    }

    .btn-register:hover:not(:disabled) {
      background-color: #4f46e5;
      transform: translateY(-1px);
    }

    .btn-register:active:not(:disabled) {
      transform: translateY(0);
    }

    .btn-register:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }

    .btn-reset {
      padding: 0.875rem 1.5rem;
      background-color: transparent;
      color: #94a3b8;
      border: 1px solid #334155;
      border-radius: 8px;
      font-size: 0.9375rem;
      font-weight: 500;
      cursor: pointer;
      transition: all 0.2s ease;
    }

    .btn-reset:hover:not(:disabled) {
      background-color: rgba(148, 163, 184, 0.1);
      color: #e2e8f0;
      border-color: #475569;
    }

    .btn-reset:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }

    /* Login Link */
    .login-link {
      text-align: center;
      margin-top: 1.5rem;
      padding-top: 1.5rem;
      border-top: 1px solid #334155;
      font-size: 0.875rem;
      color: #94a3b8;
    }

    .text-primary {
      color: #818cf8;
    }

    .text-decoration-none {
      text-decoration: none;
    }

    .text-primary:hover {
      color: #a5b4fc;
    }
  `]
})
export class RegisterComponent {
  private authService = inject(AuthService);
  private router = inject(Router);

  email = '';
  password = '';
  confirmPassword = '';
  errorMessage = '';
  isLoading = false;

  async onSubmit() {
    if (!this.email || !this.password) {
      this.errorMessage = 'Please enter both email and password.';
      return;
    }

    if (this.password !== this.confirmPassword) {
      this.errorMessage = 'Passwords do not match.';
      return;
    }

    
    this.isLoading = true;
    this.errorMessage = '';

    // Use a flag to prevent multiple submissions
    let isSubscribed = true;

    this.authService.register(this.email, this.password).subscribe({
      next: (response) => {
        if (!isSubscribed) return;
        
        
        this.isLoading = false;
        
        // Small delay to ensure UI updates before navigation
        setTimeout(() => {
          if (isSubscribed) {
            this.router.navigate(['/login']);
          }
        }, 100);
      },
      error: (err) => {
        if (!isSubscribed) return;
        
        console.error('RegisterComponent: Registration error', err);
        this.errorMessage = 'Failed to create account. The email might already be in use.';
        this.isLoading = false;
      },
      complete: () => {
        
        if (isSubscribed) {
          this.isLoading = false;
        }
      }
    });

    // Cleanup on component destroy to prevent state corruption
    const originalDestroy = this.ngOnDestroy.bind(this);
    this.ngOnDestroy = () => {
      isSubscribed = false;
      
      originalDestroy();
    };
  }

  ngOnDestroy(): void {
    // This will be overridden above to clean up subscriptions
  }

  onReset(): void {
    this.email = '';
    this.password = '';
    this.confirmPassword = '';
    this.errorMessage = '';
  }
}
