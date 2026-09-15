import { Page, expect } from '@playwright/test';

// Selectors verified against go/secscan/ui/web/app.html (sidebar nav links,
// user menu) per this project's own AppHtmlBodyFromL8erp DOM contract.
export class NavPage {
  constructor(private readonly page: Page) {}

  async goToSection(section: string): Promise<void> {
    await this.page.click(`.nav-link[data-section="${section}"]`);
  }

  async expectUsername(username: string): Promise<void> {
    await expect(this.page.locator('.username')).toHaveText(username);
  }

  async logout(): Promise<void> {
    await this.page.click('.logout-btn');
    // The server normalizes /l8ui/login/index.html to /l8ui/login/.
    await expect(this.page).toHaveURL(/\/l8ui\/login\/(index\.html)?$/, { timeout: 10000 });
  }
}
