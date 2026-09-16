import type { Page } from '@playwright/test';

// Deletes an ImageGroup (and its ImageRefs) created by a test, by name, via
// the real REST API -- called from test.afterEach/finally so `repo-*` test
// groups (add-images.spec.ts, group-detail.spec.ts) don't pile up in the
// cluster between runs. Runs as a page.evaluate() fetch so it reuses the
// already-logged-in session's bearerToken (sessionStorage) rather than
// requiring its own login. Never throws -- cleanup best-effort only, must
// not turn a passing test red or mask its own failure.
export async function deleteImageGroupByName(page: Page, imageGroupName: string): Promise<void> {
  try {
    await page.evaluate(async (name) => {
      const token = sessionStorage.getItem('bearerToken');
      if (!token) return;
      const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };
      const base = location.origin;

      const getQs = (text: string) => `?${new URLSearchParams({ body: JSON.stringify({ text }) })}`;

      const groupsResp = await fetch(
        `${base}/scan/60/ImgGroup${getQs(`select * from ImageGroup where customerId='local' and imageName='${name}'`)}`,
        { headers },
      );
      const groups = (await groupsResp.json())?.list || [];

      for (const group of groups) {
        const refsResp = await fetch(
          `${base}/scan/60/ImageRef${getQs(`select * from ImageRef where imageGroupId='${group.imageGroupId}'`)}`,
          { headers },
        );
        const refs = (await refsResp.json())?.list || [];
        for (const ref of refs) {
          await fetch(`${base}/scan/60/ImageRef`, {
            method: 'DELETE',
            headers,
            body: JSON.stringify({ text: `select * from ImageRef where imageRefId='${ref.imageRefId}'` }),
          });
        }
        await fetch(`${base}/scan/60/ImgGroup`, {
          method: 'DELETE',
          headers,
          body: JSON.stringify({ text: `select * from ImageGroup where imageGroupId='${group.imageGroupId}'` }),
        });
      }
    }, imageGroupName);
  } catch {
    // best-effort cleanup -- swallow so a cleanup failure never masks the
    // test's own pass/fail result
  }
}
