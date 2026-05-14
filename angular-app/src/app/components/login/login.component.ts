import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './login.component.html',
  styles: [`
    .login-container {
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
export class LoginComponent {
  private authService = inject(AuthService);
  private router = inject(Router);

  email = '';
  password = '';
  errorMessage = '';
  isLoading = false;

  async onSubmit() {
    if (!this.email || !this.password) {
      this.errorMessage = 'Please enter both email and password.';
      return;
    }

    this.isLoading = true;
    this.errorMessage = '';

    this.authService.login(this.email, this.password).subscribe({
      next: () => {
        console.log('Login successful');
        this.router.navigate(['/']); // Redirect to home (photo list) after login
      },
      error: (err) => {
        console.error('Login error', err);
        this.errorMessage = 'Invalid email or password. Please try again.';
        this.isLoading = false;
      }
    });
  }
}
