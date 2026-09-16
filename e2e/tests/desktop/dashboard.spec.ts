import { test, expect } from '../../fixtures/test';
import { NavPage } from '../../pages/nav.page';
import { TablePage } from '../../pages/table.page';

// Container id per AddingModule's `${mod.key}-${svc.key}-table-container`
// convention (layer8-section-generator.js) -- Images registers module
// 'images', service 'groups' (secscan-config.js). The same container is
// reused (innerHTML swapped) for both the table and chart views.
const GROUPS_CONTAINER = '#images-groups-table-container';

test.describe('dashboard', () => {
  test('KPI strip and top-vulnerabilities chart render real data', async ({ opsadminPage: page }) => {
    const nav = new NavPage(page);

    await page.goto('/app.html');
    await nav.goToSection('dashboard');

    // dashboard-page.js: the loading placeholder is swapped for
    // renderStrip()'s 4 Layer8DWidget cards once loadKpis() resolves.
    const kpiStrip = page.locator('#secscan-dashboard-kpi-strip');
    await expect(kpiStrip.locator('.secscan-kpi-loading')).toHaveCount(0, { timeout: 15000 });
    const widgets = kpiStrip.locator('.layer8d-widget');
    await expect(widgets).toHaveCount(4);

    // Never assert exact counts against live/real data (per plans/
    // playwright-e2e-testing.md §3) -- only that each card shows a real,
    // rendered numeric value, not blank/NaN.
    const labels = ['Images', 'Pending Scans', 'Critical CVEs', 'Groups Not Yet Scanned'];
    for (const label of labels) {
      const card = widgets.filter({ hasText: label });
      await expect(card).toHaveCount(1);
      const valueText = await card.locator('.layer8d-widget-value').innerText();
      expect(valueText.trim()).toMatch(/^[\d,]+$/);
    }

    // Toolbar: Add Images always enabled, Scan Images starts disabled
    // (nothing selected yet -- SecScanImageSelection starts empty).
    await expect(page.locator('#secscan-add-images-btn')).toBeEnabled();
    await expect(page.locator('#secscan-scan-images-btn')).toBeDisabled();

    // Top Images with Vulnerabilities chart (loadTopVulnChart) renders for
    // any customer with at least one ImageGroup, regardless of whether any
    // scan has actually found vulnerabilities yet (real Trivy scans depend
    // on registry access outside this suite's control -- e.g. Docker Hub's
    // anonymous pull rate limit -- so this deliberately does NOT assert on
    // bar/rect count, only that the chart itself renders).
    const chart = page.locator('#secscan-top-vuln-chart');
    await expect(chart.locator('svg.layer8d-chart-svg')).toBeVisible({ timeout: 15000 });
  });

  test('Images table sorts, filters, and switches to chart view', async ({ opsadminPage: page }) => {
    const nav = new NavPage(page);
    const table = new TablePage(page, GROUPS_CONTAINER);

    await page.goto('/app.html');
    await nav.goToSection('images');
    // Not a specific row name -- the real ImageGroup count varies (other
    // suites/seed runs add their own groups), and a specific name isn't
    // guaranteed to land on page 1 of the table's real pagination.
    await expect(table.container.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });

    // Sort by Name (col.col('imageName', ...) -> sortKey 'imageName',
    // table.page.ts's sortBy clicks th[data-column]). Sorting is a real
    // server-driven re-fetch (L8Query sortBy/descending), not a client-side
    // reorder, so assert against the row ORDER actually returned rather
    // than a fixed expected sequence -- toggle ascending vs. descending
    // and confirm the first row's name actually changes.
    const firstRowName = () => table.container.locator('tbody tr').first().locator('td').first().innerText();

    await table.sortBy('imageName');
    const ascendingFirst = await firstRowName();
    await table.sortBy('imageName'); // toggles to descending
    const descendingFirst = await firstRowName();
    expect(ascendingFirst).not.toBe(descendingFirst);

    // Filter narrows to only matching rows.
    await table.filterBy('imageName', 'secscan-web');
    await expect(table.container.locator('tbody tr')).toHaveCount(1, { timeout: 10000 });
    await table.waitForRow('secscan-web');
    await table.waitForRowGone('secscan-scanner');

    // Clear the filter before switching views so the chart isn't left
    // scoped to a single-row result by accident.
    await table.filterBy('imageName', '');
    await table.waitForRow('secscan-scanner');

    // Chart view (Layer8ViewSwitcher): secscan-config.js registers
    // ['chart'] as the extra view for the 'groups' service, plotting
    // newestCounts.critical per image (severity comparison, PRD §11.1).
    // The switcher slot (layer8-section-generator.js: "${mod.key}-${svc.key}-
    // view-switcher") is a SIBLING of the table container, not nested
    // inside it -- both share the 'images'/'groups' mod/svc keys.
    const viewSwitcher = page.locator('#images-groups-view-switcher');
    await viewSwitcher.locator('.l8-view-toggle').click();
    await viewSwitcher.locator('.l8-view-menu-item[data-view-type="chart"]').click();
    // layer8d-view-factory.js renders the chart into the SAME containerId
    // the table used, so it's still `table.container` after the switch.
    await expect(table.container.locator('svg.layer8d-chart-svg')).toBeVisible({ timeout: 15000 });
  });
});
