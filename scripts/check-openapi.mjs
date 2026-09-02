import { execFileSync } from 'node:child_process';
import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

const root = resolve(import.meta.dirname, '..');
const committed = join(root, 'web/src/generated/openapi/schema.d.ts');
const temporaryDirectory = mkdtempSync(join(tmpdir(), 'singlepage-openapi-'));
const generated = join(temporaryDirectory, 'schema.d.ts');

try {
  execFileSync(
    process.execPath,
    [join(root, 'node_modules/openapi-typescript/bin/cli.js'), join(root, 'api/openapi.yaml'), '-o', generated],
    { cwd: root, stdio: 'pipe' },
  );
  if (readFileSync(committed, 'utf8') !== readFileSync(generated, 'utf8')) {
    console.error('Generated OpenAPI types are stale. Run `npm run api:generate`.');
    process.exitCode = 1;
  }
} finally {
  rmSync(temporaryDirectory, { recursive: true, force: true });
}
