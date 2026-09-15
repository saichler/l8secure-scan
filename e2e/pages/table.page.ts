import { Page, Locator, expect } from '@playwright/test';

// Generic wrapper over Layer8DTable (go/secscan/ui/web/l8ui/edit_table/).
// Selectors verified against layer8d-table-render.js / layer8d-table-events.js:
// rows carry no data-id (only data-row-index), so row lookup is by visible
// text; action buttons carry data-action + data-id.
export class TablePage {
  constructor(private readonly page: Page, private readonly containerSelector: string) {}

  get container(): Locator {
    return this.page.locator(this.containerSelector);
  }

  rowByText(text: string): Locator {
    return this.container.locator('tbody tr').filter({ hasText: text });
  }

  async clickAdd(): Promise<void> {
    await this.container.locator('[data-action="add"]').click();
  }

  async clickEditFor(rowText: string): Promise<void> {
    await this.rowByText(rowText).locator('[data-action="edit"]').click();
  }

  async clickDeleteFor(rowText: string): Promise<void> {
    await this.rowByText(rowText).locator('[data-action="delete"]').click();
  }

  async clickRow(rowText: string): Promise<void> {
    // The app itself ignores clicks that land on an action button/toggle
    // (layer8d-table-events.js), so clicking anywhere else on the row is
    // the real onRowClick trigger.
    await this.rowByText(rowText).click();
  }

  async waitForRow(text: string, timeout = 10000): Promise<void> {
    await expect(this.rowByText(text).first()).toBeVisible({ timeout });
  }

  async waitForRowGone(text: string, timeout = 10000): Promise<void> {
    await expect(this.rowByText(text)).toHaveCount(0, { timeout });
  }

  async sortBy(columnKey: string): Promise<void> {
    await this.container.locator(`th[data-column="${columnKey}"]`).click();
  }

  async filterBy(columnKey: string, value: string): Promise<void> {
    await this.container.locator(`.l8-filter-input[data-column="${columnKey}"]`).fill(value);
  }
}
