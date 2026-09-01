import { expect, test, type Page } from '@playwright/test';

const PASSWORD = 'correct horse battery staple';

async function createPage(page: Page, password = PASSWORD) {
  await page.goto('/');
  await page.getByLabel('Password').first().fill(password);
  await page.getByLabel('Repeat password').fill(password);
  await page.getByRole('button', { name: 'Create page' }).click();
  await expect(page.getByRole('heading', { name: 'Save this link.' })).toBeVisible();
  const link = await page.locator('.secret-link').innerText();
  await page.getByRole('button', { name: 'Done' }).click();
  return link;
}

async function unlock(page: Page, link: string, password = PASSWORD) {
  await page.goto(link);
  await page.reload();
  await page.getByLabel('Password').fill(password);
  await page.getByRole('button', { name: 'Open' }).click();
}

async function waitForSaved(page: Page) {
  await expect(page.locator('.save-state')).toHaveText('Saved', { timeout: 15_000 });
}

test('create, type, refresh, unlock, and open in a second browser context', async ({ page, browser }) => {
  const plaintext = 'Private thought that must stay encrypted';
  const leakedBodies: string[] = [];
  page.on('request', (request) => {
    if (request.url().includes('/api/')) leakedBodies.push(request.postData() ?? '');
  });

  const link = await createPage(page);
  await page.getByLabel('Outline block').fill(plaintext);
  await waitForSaved(page);
  expect(leakedBodies.join('\n')).not.toContain(plaintext);
  expect(leakedBodies.join('\n')).not.toContain(PASSWORD);
  expect(leakedBodies.join('\n')).not.toContain(new URL(link).hash.slice(1));

  await page.reload();
  await page.getByLabel('Password').fill(PASSWORD);
  await page.getByRole('button', { name: 'Open' }).click();
  await expect(page.getByLabel('Outline block')).toHaveValue(plaintext);

  const second = await browser.newContext();
  const otherPage = await second.newPage();
  await unlock(otherPage, link);
  await expect(otherPage.getByLabel('Outline block')).toHaveValue(plaintext);
  await second.close();
});

test('nested blocks are searchable through inherited metadata', async ({ page }) => {
  await createPage(page);
  const root = page.getByLabel('Outline block').first();
  await root.fill('Work #work project::mindrop');
  await root.press('Enter');
  const child = page.getByLabel('Outline block').nth(1);
  await child.press('Tab');
  await child.fill('Encryption area::security');
  await child.press('Enter');
  const grandchild = page.getByLabel('Outline block').nth(2);
  await grandchild.press('Tab');
  await grandchild.fill('WebCrypto');
  await grandchild.press('Enter');
  const laterChild = page.getByLabel('Outline block').nth(3);
  await laterChild.fill('Later note');
  await root.press('Enter');
  const laterRoot = page.getByLabel('Outline block').last();
  await laterRoot.fill('Encryption archive #work project::mindrop area::security');
  await waitForSaved(page);

  const search = page.getByLabel('Search', { exact: true });
  await search.fill('Encryption #work project:mindrop area:security');
  await expect(page.locator('.block-row.search-match .block-text').first()).toHaveValue('Encryption area::security');
  await expect(page.locator('.autocomplete-menu')).toHaveCount(0);
  await expect(page.getByLabel('Outline block')).toHaveCount(5);
  await expect(page.getByLabel('Outline block').nth(2)).toHaveValue('WebCrypto');
  await expect(page.getByLabel('Outline block').nth(3)).toHaveValue('Later note');
  await expect(page.getByLabel('Outline block').nth(4)).toHaveValue('Encryption archive #work project::mindrop area::security');

  await search.press('Enter');
  await expect(search).toHaveValue('Encryption #work project:mindrop area:security');
  await expect(page.getByLabel('Outline block').nth(1)).toBeFocused();

  await grandchild.focus();
  await grandchild.press('Control+Enter');
  const firstFilteredChild = page.getByLabel('Outline block').nth(3);
  await expect(firstFilteredChild).toBeFocused();
  await firstFilteredChild.fill('First child created in the filtered tree');

  await grandchild.focus();
  await grandchild.press('Enter');
  const newFilteredBlock = page.getByLabel('Outline block').nth(4);
  await expect(newFilteredBlock).toBeFocused();
  await expect(page.getByLabel('Outline block').nth(3)).toHaveValue('First child created in the filtered tree');
  await expect(page.getByLabel('Outline block').nth(5)).toHaveValue('Later note');
  await expect(page.getByLabel('Outline block').nth(6)).toHaveValue('Encryption archive #work project::mindrop area::security');
  await newFilteredBlock.fill('New text inside the filtered tree');
  await expect(newFilteredBlock).toHaveValue('New text inside the filtered tree');
  await expect(page.getByLabel('Outline block')).toHaveCount(7);

  await root.focus();
  await root.press('Enter');
  const newFilteredRoot = page.getByLabel('Outline block').last();
  await expect(newFilteredRoot).toBeFocused();
  await expect(page.getByLabel('Outline block').nth(6)).toHaveValue('Encryption archive #work project::mindrop area::security');
  await newFilteredRoot.fill('New top-level filtered block');
  await expect(newFilteredRoot).toHaveValue('New top-level filtered block');
  await expect(search).toHaveValue('Encryption #work project:mindrop area:security');
});

test('arrow keys navigate blocks, Shift+Enter adds a line, and keyboard shortcuts zoom branches', async ({ page }) => {
  await createPage(page);
  const first = page.getByLabel('Outline block').first();
  await first.fill('First');
  await first.press('Enter');
  const second = page.getByLabel('Outline block').nth(1);
  await second.fill('Second');

  await second.press('Home');
  await second.press('ArrowUp');
  await expect(first).toBeFocused();
  await first.press('ArrowDown');
  await expect(second).toBeFocused();

  await second.fill('A long line that can wrap without creating a real line break inside the stored block');
  await second.evaluate((element: HTMLTextAreaElement) => element.setSelectionRange(12, 12));
  await second.press('ArrowUp');
  await expect(second).toBeFocused();
  await second.evaluate((element: HTMLTextAreaElement) => element.setSelectionRange(5, 18));
  await second.press('ArrowDown');
  await expect(second).toBeFocused();

  await second.fill('Second');
  await second.press('End');
  await second.press('Shift+Enter');
  await second.type('another line');
  await expect(second).toHaveValue('Second\nanother line');

  await second.press('Control+Enter');
  const child = page.getByLabel('Outline block').nth(2);
  await child.fill('Nested');
  await second.focus();
  await second.press('Control+]');
  await expect(page.getByRole('navigation', { name: 'Focused branch' })).toContainText('Second');
  const zoomedChild = page.getByLabel('Outline block').first();
  await expect(zoomedChild).toHaveValue('Nested');
  await expect(zoomedChild).toBeFocused();

  await zoomedChild.press('Control+[');
  await expect(page.getByRole('navigation', { name: 'Focused branch' })).toHaveText('All notes');
  await expect(second).toBeFocused();
});

test('warns before a dirty page can be reloaded', async ({ page }) => {
  await createPage(page);
  await page.getByLabel('Outline block').fill('Not saved yet');
  const navigationAllowed = await page.evaluate(() => window.dispatchEvent(new Event('beforeunload', { cancelable: true })));
  expect(navigationAllowed).toBe(false);
  await expect(page.getByLabel('Outline block')).toHaveValue('Not saved yet');
  await waitForSaved(page);
});

test('theme choice survives reload', async ({ page }) => {
  await page.goto('/');
  await page.getByRole('button', { name: 'Use dark theme' }).click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await page.reload();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
});

test('exports the outline and imports a Markdown file', async ({ page }) => {
  await createPage(page);
  await page.getByLabel('Outline block').fill('Original note');

  await page.getByRole('button', { name: 'Settings' }).click();
  const downloadPromise = page.waitForEvent('download');
  await page.getByRole('button', { name: 'Export Markdown' }).click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/^mindrop-outline-\d{4}-\d{2}-\d{2}\.md$/);

  await page.getByRole('button', { name: 'Settings' }).click();
  const chooserPromise = page.waitForEvent('filechooser');
  await page.getByRole('button', { name: 'Import Markdown' }).click();
  const chooser = await chooserPromise;
  page.once('dialog', (dialog) => dialog.accept());
  await chooser.setFiles({
    name: 'notes.md',
    mimeType: 'text/markdown',
    buffer: Buffer.from('- Imported root\n  - Imported child\n- Second root\n')
  });

  await expect(page.getByLabel('Outline block')).toHaveCount(3);
  await expect(page.getByLabel('Outline block').nth(0)).toHaveValue('Imported root');
  await expect(page.getByLabel('Outline block').nth(1)).toHaveValue('Imported child');
  await expect(page.getByLabel('Outline block').nth(2)).toHaveValue('Second root');
  await waitForSaved(page);
});

test('rotating the access link invalidates the old link', async ({ page, browser }) => {
  const oldLink = await createPage(page);
  await page.getByLabel('Outline block').fill('Rotated secret');
  await waitForSaved(page);

  const staleContext = await browser.newContext();
  const stalePage = await staleContext.newPage();
  await unlock(stalePage, oldLink);
  await expect(stalePage.getByLabel('Outline block')).toHaveValue('Rotated secret');

  await page.getByRole('button', { name: 'Settings' }).click();
  await page.getByRole('button', { name: 'Create new access link' }).click();
  await page.getByRole('button', { name: 'Create new link', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Save this link.' })).toBeVisible({ timeout: 15_000 });
  const newLink = await page.locator('.secret-link').innerText();
  expect(newLink).not.toBe(oldLink);
  expect(new URL(newLink).pathname).not.toBe(new URL(oldLink).pathname);

  await stalePage.getByLabel('Outline block').fill('Edit from an old link');
  await expect(stalePage.locator('.save-state')).toHaveText('Link replaced', { timeout: 15_000 });
  await staleContext.close();

  await page.goto(oldLink);
  await expect(page.getByRole('heading', { name: 'Page not found' })).toBeVisible();

  await unlock(page, newLink);
  await expect(page.getByLabel('Outline block')).toHaveValue('Rotated secret');
});

test('changing password invalidates the old password and write capability', async ({ page, browser }) => {
  const link = await createPage(page);
  await page.getByLabel('Outline block').fill('Password rotation');
  await waitForSaved(page);

  const staleContext = await browser.newContext();
  const stalePage = await staleContext.newPage();
  await unlock(stalePage, link);
  await expect(stalePage.getByLabel('Outline block')).toHaveValue('Password rotation');

  await page.getByRole('button', { name: 'Settings' }).click();
  await page.getByRole('button', { name: 'Change password' }).click();
  await page.getByLabel('New password', { exact: true }).fill('a much newer password');
  await page.getByLabel('Repeat new password', { exact: true }).fill('a much newer password');
  await page.getByRole('button', { name: 'Change password', exact: true }).click();
  await expect(page.getByRole('dialog')).toBeHidden({ timeout: 15_000 });

  await stalePage.getByLabel('Outline block').fill('Write with an old password');
  await expect(stalePage.locator('.save-state')).toHaveText('Link replaced', { timeout: 15_000 });
  await staleContext.close();

  await page.goto(link);
  await page.reload();
  await page.getByLabel('Password').fill(PASSWORD);
  await page.getByRole('button', { name: 'Open' }).click();
  await expect(page.getByText('Unable to open page. Check the password and link.')).toBeVisible();

  await page.getByLabel('Password').fill('a much newer password');
  await page.getByRole('button', { name: 'Open' }).click();
  await expect(page.getByLabel('Outline block')).toHaveValue('Password rotation');
});
