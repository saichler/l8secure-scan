import { test, expect } from '../../fixtures/test';
import { NavPage } from '../../pages/nav.page';
import { PopupPage } from '../../pages/popup.page';
import { deleteImageGroupByName } from '../../fixtures/cleanup';

// Container id per AddingModule convention (module 'images', service 'groups').
const GROUPS_CONTAINER = '#images-groups-table-container';

test.describe('add images (bulk ingestion)', () => {
  let createdGroupName: string | undefined;

  test.afterEach(async ({ opsadminPage: page }) => {
    if (createdGroupName) {
      await deleteImageGroupByName(page, createdGroupName);
      createdGroupName = undefined;
    }
  });

  test('valid, duplicate, and unparseable lines produce the correct created/skipped/errors split', async ({ opsadminPage: page }) => {
    const nav = new NavPage(page);
    const popup = new PopupPage(page);

    // The grouping algorithm derives imageName from repoName's last path
    // segment (PRD §5, "grouped ... regardless of the repo and the tag") --
    // a leading "e2e/" would be stripped, so imageGroupName below is what
    // actually shows up in the Images list, not the full repoName.
    const imageGroupName = `repo-${Date.now()}`;
    createdGroupName = imageGroupName;
    const repo = `e2e/${imageGroupName}`;
    const validLine = `${repo}:v1`;
    // ParseImageRefString rejects any value containing a single-quote
    // (no verified L8QL escaping mechanism, imageref_ingest.go).
    const badLine = `e2e/bad'repo:v1`;

    await page.goto('/app.html');
    await nav.goToSection('dashboard');

    await page.click('#secscan-add-images-btn');
    await popup.expectTitle('Add Images');
    // Paste the same valid line twice -- the second occurrence is a
    // duplicate of the row the first occurrence just created.
    await popup.body.locator('#secscan-add-images-textarea').fill(`${validLine}\n${validLine}\n${badLine}`);
    await popup.save(); // Add Images' save button is labeled "Add"

    await popup.expectTitle('Add Images Result');
    await expect(popup.body).toContainText('1 created, 1 skipped, 1 errors.');
    await expect(popup.body.locator('h4:has-text("Skipped")')).toBeVisible();
    await expect(popup.body).toContainText(validLine); // the skipped line, shown per "no silent fallbacks"
    await expect(popup.body.locator('h4:has-text("Errors")')).toBeVisible();
    await expect(popup.body).toContainText(badLine);

    await popup.save(); // this popup's save button is labeled "Close"
    await popup.expectClosed();

    await nav.goToSection('images');
    await expect(page.locator(GROUPS_CONTAINER).locator('tbody tr').filter({ hasText: imageGroupName })).toBeVisible({ timeout: 10000 });
  });
});
