import assert from 'node:assert/strict';
import { test } from 'node:test';
import { architectureFailures } from './check-architecture.mjs';

const path = (relative) => `/repo/web/src/${relative}`;
const check = (relative, contents) => architectureFailures([{ path: path(relative), contents }]);

test('feature UI cannot import infrastructure', () => {
  assert.match(
    check('features/editor/ui/Editor.svelte', "import { api } from '../../../infrastructure/api';")[0],
    /feature imports composition or infrastructure/,
  );
});

test('feature logic cannot import Svelte UI', () => {
  assert.match(
    check('features/editor/logic/editor.ts', "import View from '../ui/View.svelte';")[0],
    /feature logic imports Svelte UI/,
  );
});

test('features cannot deep-import another feature', () => {
  assert.match(
    check('features/editor/logic/editor.ts', "import { save } from '../../page-session/logic/use-cases';")[0],
    /feature imports private code from page-session/,
  );
});

test('feature dynamic imports follow the same boundaries', () => {
  assert.match(
    check('features/editor/logic/editor.ts', "const api = import('../../../infrastructure/api');")[0],
    /feature imports composition or infrastructure/,
  );
});

test('import-like text in comments is ignored', () => {
  assert.deepEqual(
    check('features/editor/logic/editor.ts', "// import api from '../../../infrastructure/api';"),
    [],
  );
});

test('entities cannot depend on features', () => {
  assert.match(
    check('entities/outline/document.ts', "import type { Session } from '../../features/page-session/logic/model';")[0],
    /entity imports an outer layer/,
  );
});

test('entities cannot depend on external packages', () => {
  assert.match(
    check('entities/outline/document.ts', "import { setup } from 'xstate';")[0],
    /entity imports an external package/,
  );
});

test('app composition can wire feature UI and logic', () => {
  assert.deepEqual(
    check('app/AppShell.svelte', [
      "import StartView from '../features/page-session/ui/StartView.svelte';",
      "import { createSession } from '../features/page-session/logic/use-cases';",
    ].join('\n')),
    [],
  );
});

test('shared utilities cannot depend on feature code', () => {
  assert.match(
    check('shared/format.ts', "import type { Session } from '../features/page-session/logic/model';")[0],
    /shared imports an inner or outer application layer/,
  );
});

test('infrastructure may implement feature ports but cannot import feature UI', () => {
  assert.deepEqual(
    check('infrastructure/api/page-api.ts', "import type { PageRepository } from '../../features/page-session/logic/ports';"),
    [],
  );
  assert.match(
    check('infrastructure/browser/dialog.ts', "import Dialog from '../../features/page-session/ui/Dialogs.svelte';")[0],
    /infrastructure imports app or feature UI/,
  );
});
