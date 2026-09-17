import { test, expect } from '../../fixtures/test';
import { NavPage } from '../../pages/nav.page';

test.describe('csv export', () => {
  test('Reports section downloads a vulnerabilities CSV', async ({ opsadminPage: page }) => {
    const nav = new NavPage(page);

    await page.goto('/app.html');
    await nav.goToSection('reports');

    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.click('#secscan-vuln-report-btn'),
    ]);

    expect(download.suggestedFilename()).toMatch(/\.csv$/);
    const path = await download.path();
    expect(path).toBeTruthy();
    const fs = await import('fs');
    const content = fs.readFileSync(path!, 'utf-8');
    // §10's consolidated 6-column set (Name, Category, Newest, Oldest,
    // Reduction %, Image Refs -- one column per severity-count group
    // instead of one per severity, matching the Images table's own
    // Vulnerabilities column format).
    const header = content.split('\n')[0];
    expect(header).toBe('Name,Category,Newest,Oldest,Reduction %,Image Refs');
  });
});
