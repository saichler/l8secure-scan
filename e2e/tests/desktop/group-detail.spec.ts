import { test, expect } from '@playwright/test';
import { NavPage } from '../../pages/nav.page';
import { PopupPage } from '../../pages/popup.page';
import { TablePage } from '../../pages/table.page';
import { LoginPage } from '../../pages/login.page';
import { OPSADMIN_USER, OPSADMIN_PASS } from '../../fixtures/env';
import { deleteImageGroupByName } from '../../fixtures/cleanup';

const GROUPS_CONTAINER = '#images-groups-table-container';

test.describe('group detail', () => {
  let createdGroupName: string | undefined;

  test.afterEach(async ({ page }) => {
    if (createdGroupName) {
      await deleteImageGroupByName(page, createdGroupName);
      createdGroupName = undefined;
    }
  });

  test('checkbox selection and a real scan run reach the dashboard progress bar', async ({ page }) => {
    test.setTimeout(150000); // real Trivy scan against a real cluster, not mocked
    const nav = new NavPage(page);
    const popup = new PopupPage(page);
    const login = new LoginPage(page);

    // Deliberately NOT the shared, cached opsadminPage fixture: every test
    // using it shares one AaaId (one login), and the server's live-query
    // subscription is one-per-AaaId (l8utils/plans/generic-websocket-
    // change-notifications.md, "Known limitations") -- any other opsadmin
    // test registering its own query concurrently would silently steal
    // this test's ScanJob subscription slot, and the progress bar would
    // never update (confirmed: this test passes in isolation, fails only
    // when run alongside other opsadmin-session tests). A fresh login here
    // gets a fresh, uncontended AaaId.
    await login.goto();
    await login.login(OPSADMIN_USER, OPSADMIN_PASS);
    await login.expectLoggedIn();
    await login.resolveCustomerPickerIfPresent('Local');

    const imageGroupName = `repo-${Date.now()}`;
    createdGroupName = imageGroupName;
    const repo = `e2e/${imageGroupName}`;

    // Seed one never-scanned image via the real Add Images flow (not a
    // direct API call -- keeps this test independent of add-images.spec.ts
    // while still exercising only real, already-verified UI paths).
    await page.goto('/app.html');
    await nav.goToSection('dashboard');
    await page.click('#secscan-add-images-btn');
    await popup.expectTitle('Add Images');
    await popup.body.locator('#secscan-add-images-textarea').fill(`${repo}:v1`);
    await popup.save();
    await popup.expectTitle('Add Images Result');
    await popup.save(); // "Close"
    await popup.expectClosed();

    // Open Group Detail via the real row click (row-click override wired
    // in secscan-init.js: ImageGroup -> SecScanGroupDetail.open). Filter
    // first -- the real ImageGroup count grows over time (this project's
    // own images, other seed runs), so a freshly created group isn't
    // guaranteed to land on page 1 of the table's real pagination.
    await nav.goToSection('images');
    const table = new TablePage(page, GROUPS_CONTAINER);
    await table.filterBy('imageName', imageGroupName);
    const groupRow = page.locator(GROUPS_CONTAINER).locator('tbody tr').filter({ hasText: imageGroupName });
    await expect(groupRow).toBeVisible({ timeout: 10000 });
    await groupRow.click();

    await popup.expectTitle(imageGroupName);

    const refCheckbox = popup.body.locator('.secscan-ref-select');
    await expect(refCheckbox).toBeVisible({ timeout: 10000 });
    await expect(popup.body.locator('#secscan-selection-hint')).toContainText('Select images below, then go to Dashboard to scan them.');
    await refCheckbox.click();
    await expect(popup.body.locator('#secscan-selection-hint')).toContainText('1 image selected for scanning (Dashboard → Scan Images).');

    await popup.closeViaX();
    await popup.expectClosed();

    // The scan trigger lives on Dashboard, not in this popup (image-selection.js:
    // selection is cross-page/cross-popup state, read by dashboard-page.js).
    await nav.goToSection('dashboard');
    const scanBtn = page.locator('#secscan-scan-images-btn');
    await expect(scanBtn).toBeEnabled();
    await expect(scanBtn).toContainText('Scan Images (1)');
    await scanBtn.click();

    // Layer8DProgressBar.attach() drives this via the websocket-registered
    // ScanJob subscription (l8utils/plans/generic-websocket-change-notifications.md),
    // not polling -- wait for it to reach a terminal state (COMPLETED,
    // FAILED, or PARTIAL; this sandbox's Trivy scan can fail on network/
    // registry access grounds, which is an infra limitation, not something
    // this test should treat as a product bug -- what's under test here is
    // that the progress bar reaches a terminal state driven by the real
    // backend at all).
    const progressWrap = page.locator('#secscan-scan-progress');
    await expect(progressWrap).toBeVisible({ timeout: 15000 });
    await expect(progressWrap).toContainText(/Completed|Failed|Partially completed/, { timeout: 120000 });
  });
});
