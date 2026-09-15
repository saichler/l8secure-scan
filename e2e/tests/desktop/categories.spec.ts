import { test, expect } from '../../fixtures/test';
import { NavPage } from '../../pages/nav.page';
import { TablePage } from '../../pages/table.page';
import { PopupPage } from '../../pages/popup.page';

// Container id per AddingModule's `${mod.key}-${svc.key}-table-container`
// convention (layer8-section-generator.js:319) -- categories registers one
// module/service both keyed "categories" (secscan-section-config.js).
const CATEGORIES_CONTAINER = '#categories-categories-table-container';

test.describe('categories', () => {
  test('create, edit, and delete a category', async ({ opsadminPage: page }) => {
    const nav = new NavPage(page);
    const table = new TablePage(page, CATEGORIES_CONTAINER);
    const popup = new PopupPage(page);

    const name = `e2e-cat-${Date.now()}`;
    // Deliberately NOT `${name}-renamed`: TablePage.rowByText/waitForRowGone
    // match by substring, so a renamed value containing the original name
    // would make "the old name is gone" impossible to ever observe.
    const renamedTo = `e2e-cat-renamed-${Date.now()}`;

    await page.goto('/app.html');
    await nav.goToSection('categories');

    await table.clickAdd();
    await popup.expectTitle('Add Category');
    await popup.fillText('name', name);
    await popup.fillText('colorCode', '#3366ff');
    await popup.save();
    await popup.expectClosed();
    await table.waitForRow(name);

    await table.clickEditFor(name);
    await popup.expectTitle('Edit Category');
    await popup.fillText('name', renamedTo);
    await popup.save();
    await popup.expectClosed();
    await table.waitForRow(renamedTo);
    await table.waitForRowGone(name);

    await table.clickDeleteFor(renamedTo);
    await popup.expectTitle('Confirm Delete');
    await popup.save(); // Confirm Delete's Save button is labeled "Delete"
    await popup.expectClosed();
    await table.waitForRowGone(renamedTo);
  });
});
