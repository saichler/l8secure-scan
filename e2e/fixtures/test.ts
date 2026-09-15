import { test as base, expect, type Page } from '@playwright/test';
import { ensureSession, applySession } from './auth';

type Fixtures = {
  opsadminPage: Page;
  customerPage: Page;
};

// Extends Playwright's base test with two ready-to-use, already-authenticated
// pages, one per role, backed by a cached session (see fixtures/auth.ts) so
// every later spec (Phase 2+) gets a logged-in page for free instead of
// re-implementing the login flow itself.
export const test = base.extend<Fixtures>({
  opsadminPage: async ({ browser }, use) => {
    const session = await ensureSession(browser, 'opsadmin');
    const context = await browser.newContext({ ignoreHTTPSErrors: true });
    await applySession(context, session);
    const page = await context.newPage();
    await use(page);
    await context.close();
  },
  customerPage: async ({ browser }, use) => {
    const session = await ensureSession(browser, 'customer');
    const context = await browser.newContext({ ignoreHTTPSErrors: true });
    await applySession(context, session);
    const page = await context.newPage();
    await use(page);
    await context.close();
  },
});

export { expect };
