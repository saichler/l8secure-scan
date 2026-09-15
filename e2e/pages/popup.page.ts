import { Page, Locator, expect } from '@playwright/test';

// Generic wrapper over Layer8DPopup (go/secscan/ui/web/l8ui/popup/layer8d-popup.js)
// and the Layer8DForms field convention (#field-<key>, go/secscan/ui/web/l8ui/shared/layer8d-forms-fields.js).
export class PopupPage {
  constructor(private readonly page: Page) {}

  // Topmost non-stacked popup -- matches Layer8DPopup.getBody()'s own
  // convention for which overlay is "current" when popups can stack.
  get overlay(): Locator {
    return this.page.locator('.probler-popup-overlay:not(.stacked)').last();
  }

  get body(): Locator {
    return this.overlay.locator('.probler-popup-body');
  }

  async expectTitle(title: string): Promise<void> {
    await expect(this.overlay.locator('.probler-popup-title')).toHaveText(title);
  }

  field(key: string): Locator {
    return this.body.locator(`#field-${key}`);
  }

  async fillText(key: string, value: string): Promise<void> {
    await this.field(key).fill(value);
  }

  async save(): Promise<void> {
    await this.overlay.locator('.probler-popup-footer .btn-primary').click();
  }

  async cancel(): Promise<void> {
    await this.overlay.locator('.probler-popup-footer .btn-secondary').click();
  }

  async closeViaX(): Promise<void> {
    await this.overlay.locator('.probler-popup-close').click();
  }

  async expectClosed(timeout = 10000): Promise<void> {
    await expect(this.page.locator('.probler-popup-overlay')).toHaveCount(0, { timeout });
  }

  async goToTab(tabId: string): Promise<void> {
    await this.overlay.locator(`.probler-popup-tab[data-tab="${tabId}"]`).click();
  }
}
