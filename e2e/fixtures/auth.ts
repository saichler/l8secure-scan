import fs from 'fs';
import path from 'path';
import type { Browser, BrowserContext } from '@playwright/test';
import { LoginPage } from '../pages/login.page';
import { OPSADMIN_USER, OPSADMIN_PASS, CUSTOMER_USER, CUSTOMER_PASS } from './env';

export type Role = 'opsadmin' | 'customer';

const CREDENTIALS: Record<Role, { user: string; pass: string }> = {
  opsadmin: { user: OPSADMIN_USER, pass: OPSADMIN_PASS },
  customer: { user: CUSTOMER_USER, pass: CUSTOMER_PASS },
};

// opsadmin has no L8User.customer scope, so a fresh login always hits the
// "Select Customer" popup (SecScanCustomerPicker) before the app is usable.
// "Local" is this project's one seeded customer (go/tests/mocks/seed.go). A
// customer-role login is already scoped and never sees this popup.
const CUSTOMER_PICKER_SELECTION: Partial<Record<Role, string>> = {
  opsadmin: 'Local',
};

const AUTH_DIR = path.join(__dirname, '..', '.auth');

function sessionFilePath(role: Role): string {
  return path.join(AUTH_DIR, `${role}.session.json`);
}

/**
 * Logs in as `role` through the real UI, resolves the post-login customer
 * picker if one appears, and caches the resulting sessionStorage to disk.
 *
 * This app keeps its bearer token in sessionStorage
 * (l8ui/login/layer8d-login-auth.js: sessionStorage.setItem('bearerToken',
 * ...)), which Playwright's own `context.storageState()` does NOT capture
 * (cookies + localStorage only) -- so the session has to be captured and
 * restored by hand via `applySession` below.
 */
export async function ensureSession(browser: Browser, role: Role): Promise<Record<string, string>> {
  const filePath = sessionFilePath(role);
  if (fs.existsSync(filePath)) {
    return JSON.parse(fs.readFileSync(filePath, 'utf-8'));
  }

  const { user, pass } = CREDENTIALS[role];
  const context = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await context.newPage();
  const login = new LoginPage(page);
  await login.goto();
  await login.login(user, pass);
  await login.expectLoggedIn();

  const pickerCustomer = CUSTOMER_PICKER_SELECTION[role];
  if (pickerCustomer) {
    await login.resolveCustomerPickerIfPresent(pickerCustomer);
  }

  const session = await page.evaluate(() => {
    const out: Record<string, string> = {};
    for (let i = 0; i < window.sessionStorage.length; i++) {
      const key = window.sessionStorage.key(i)!;
      out[key] = window.sessionStorage.getItem(key)!;
    }
    return out;
  });

  fs.mkdirSync(AUTH_DIR, { recursive: true });
  fs.writeFileSync(filePath, JSON.stringify(session));
  await context.close();
  return session;
}

/** Applies a captured sessionStorage snapshot to every page created in `context`, before any page script runs. */
export async function applySession(context: BrowserContext, session: Record<string, string>): Promise<void> {
  await context.addInitScript((data: Record<string, string>) => {
    for (const [key, value] of Object.entries(data)) {
      window.sessionStorage.setItem(key, value);
    }
  }, session);
}
