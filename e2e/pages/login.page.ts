import { Page, expect } from '@playwright/test';

// Selectors verified against the real markup:
// go/secscan/ui/web/l8ui/login/index.html (form fields, error banner)
// go/secscan/ui/web/js/app.js (post-login redirect target, .username fill-in)
export class LoginPage {
  constructor(private readonly page: Page) {}

  async goto(): Promise<void> {
    await this.page.goto('/l8ui/login/index.html');
  }

  async login(username: string, password: string): Promise<void> {
    await this.page.fill('#username', username);
    await this.page.fill('#password', password);
    await this.page.click('#login-btn');
  }

  async expectLoggedIn(): Promise<void> {
    await expect(this.page).toHaveURL(/\/app\.html$/, { timeout: 10000 });
    // js/app.js sets .username from sessionStorage on load; 'User' is the
    // fallback shown only when no session exists.
    await expect(this.page.locator('.username')).not.toHaveText('User');
  }

  /**
   * A user with no L8User.customer scope (e.g. opsadmin) is met with the
   * "Select Customer" popup (secscan/customer-picker.js) before the app is
   * usable. No-ops if the picker doesn't appear (already-scoped user).
   */
  async resolveCustomerPickerIfPresent(displayName: string): Promise<void> {
    const pickerInput = this.page.locator('#secscan-customer-picker-input');
    const appeared = await pickerInput
      .waitFor({ state: 'visible', timeout: 5000 })
      .then(() => true)
      .catch(() => false);
    if (!appeared) return;

    // Layer8DReferencePicker.open(input) is called from the popup's onShow,
    // so the picker overlay is already open when the input appears -- no
    // click needed (and one would be blocked by the overlay covering it).
    await this.page.locator(`.layer8d-refpicker-item[data-display="${displayName}"]`).click();
    await this.page.locator('.layer8d-refpicker-select-btn').click();
    await expect(pickerInput).not.toBeVisible();
  }

  async expectError(messageSubstring: string): Promise<void> {
    const banner = this.page.locator('#error-message');
    await expect(banner).toHaveClass(/visible/);
    await expect(this.page.locator('#error-text')).toContainText(messageSubstring);
  }
}
