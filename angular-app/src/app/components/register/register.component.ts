import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './register.component.html',
  styles: [`
    .register-container {
      max-width: 400px;
      margin: 5rem auto;
      padding: 2rem;
      background: #ffffff10;
      border-radius: 8px;
      box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
    }
    .error-msg { color: #ff6b6b; margin-bottom: 1rem; }
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

    this.authService.register(this.email, this.password).subscribe({
      next: () => {
        console.log('Registration successful');
        this.router.navigate(['/login']); // Redirect to login after registration
      },
      error: (err) => {
        console.error('Registration error', err);
        this.errorMessage = 'Failed to create account. The email might already be in use.';
        this.isLoading = false;
      }
    });
  }
}
