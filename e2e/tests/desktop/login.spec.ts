import { test, expect } from '../../fixtures/test';
import { LoginPage } from '../../pages/login.page';
import { NavPage } from '../../pages/nav.page';
import { OPSADMIN_USER, OPSADMIN_PASS, CUSTOMER_USER, CUSTOMER_PASS } from '../../fixtures/env';

test.describe('login', () => {
  test('opsadmin can log in, see the dashboard, and log out', async ({ page }) => {
    const login = new LoginPage(page);
    const nav = new NavPage(page);

    await login.goto();
    await login.login(OPSADMIN_USER, OPSADMIN_PASS);
    await login.expectLoggedIn();
    await nav.expectUsername(OPSADMIN_USER);

    // opsadmin has no single-customer scope, so a fresh login always hits
    // the "Select Customer" popup before the dashboard becomes usable.
    await login.resolveCustomerPickerIfPresent('Local');

    // Dashboard KPI strip starts as a loading placeholder and its content is
    // replaced with real Layer8DWidget cards once loadKpis() resolves
    // (go/secscan/ui/web/secscan/dashboard-page.js). Note: the outer
    // #secscan-dashboard-kpi-strip element's own class attribute keeps
    // "secscan-kpi-loading" forever -- only .innerHTML is swapped, never
    // .className -- so that class can't be used as a loaded/not-loaded
    // signal; assert on the nested loading placeholder disappearing and the
    // real KPI label appearing instead.
    const kpiStrip = page.locator('#secscan-dashboard-kpi-strip');
    await expect(kpiStrip.locator('.secscan-kpi-loading')).toHaveCount(0, { timeout: 15000 });
    await expect(kpiStrip).toContainText('Images');

    await nav.logout();

    // logout() explicitly removes bearerToken from both storages
    // (js/app.js) -- confirm it's actually gone, not just that we navigated.
    const bearerToken = await page.evaluate(() => window.sessionStorage.getItem('bearerToken'));
    expect(bearerToken).toBeNull();
  });

  test('customer-role user logs in directly onto the dashboard, no customer picker', async ({ page }) => {
    const login = new LoginPage(page);
    const nav = new NavPage(page);

    await login.goto();
    await login.login(CUSTOMER_USER, CUSTOMER_PASS);
    await login.expectLoggedIn();
    await nav.expectUsername(CUSTOMER_USER);

    // Already scoped via L8User.customer -- the picker must never appear.
    await expect(page.locator('#secscan-customer-picker-input')).not.toBeVisible();
    const userCustomer = await page.evaluate(() => window.sessionStorage.getItem('userCustomer'));
    expect(userCustomer).toBe('local');

    await nav.logout();
  });

  test('invalid credentials show an error and do not log in', async ({ page }) => {
    const login = new LoginPage(page);

    await login.goto();
    await login.login(OPSADMIN_USER, 'definitely-the-wrong-password');
    await login.expectError('unknown user/pass');

    // Still on the login page -- no session was established.
    await expect(page).toHaveURL(/\/l8ui\/login\//);
    const bearerToken = await page.evaluate(() => window.sessionStorage.getItem('bearerToken'));
    expect(bearerToken).toBeNull();
  });
});
