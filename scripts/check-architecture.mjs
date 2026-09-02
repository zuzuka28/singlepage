import { readdirSync, readFileSync, statSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import ts from 'typescript';

const projectRoot = resolve(import.meta.dirname, '..');
const sourceRoot = join(projectRoot, 'web/src');

function sourceFiles(directory) {
  return readdirSync(directory).flatMap((entry) => {
    const path = join(directory, entry);
    return statSync(path).isDirectory() ? sourceFiles(path) : [path];
  });
}

function importsOf(path, contents) {
  const scripts = path.endsWith('.svelte')
    ? [...contents.matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/g)].map((match) => match[1])
    : [];
  const source = scripts.length ? scripts.join('\n') : contents;
  const sourceFile = ts.createSourceFile(path, source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
  const imports = [];
  const visit = (node) => {
    if ((ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) && node.moduleSpecifier && ts.isStringLiteral(node.moduleSpecifier)) {
      imports.push(node.moduleSpecifier.text);
    }
    if (ts.isCallExpression(node) && node.expression.kind === ts.SyntaxKind.ImportKeyword) {
      const [specifier] = node.arguments;
      if (specifier && ts.isStringLiteral(specifier)) imports.push(specifier.text);
    }
    ts.forEachChild(node, visit);
  };
  visit(sourceFile);
  return imports;
}

function featureOf(path) {
  return path.match(/\/web\/src\/features\/([^/]+)\//)?.[1] ?? null;
}

function resolveImport(path, specifier) {
  return specifier.startsWith('.') ? resolve(dirname(path), specifier) : null;
}

export function architectureFailures(entries) {
  const failures = [];

  for (const { path, contents } of entries) {
    if (!/\.(?:svelte|ts)$/.test(path) || path.endsWith('.test.ts') || path.includes('/generated/')) continue;

    const name = relative(projectRoot, path);
    const feature = featureOf(path);
    const inFeatureLogic = /\/web\/src\/features\/[^/]+\/logic\//.test(path);
    const inFeatureUI = /\/web\/src\/features\/[^/]+\/ui\//.test(path);
    const inEntity = path.includes('/web/src/entities/');
    const inShared = path.includes('/web/src/shared/');
    const inInfrastructure = path.includes('/web/src/infrastructure/');

    for (const specifier of importsOf(path, contents)) {
      const imported = resolveImport(path, specifier);

      if (inFeatureLogic && (specifier === 'svelte' || specifier.startsWith('@svelte') || specifier.endsWith('.svelte'))) {
        failures.push(`${name}: feature logic imports Svelte UI`);
      }
      if (inEntity && !imported) {
        failures.push(`${name}: entity imports an external package`);
      }
      if (!imported) continue;

      const importedFeature = featureOf(imported);
      if ((inFeatureLogic || inFeatureUI) && importedFeature && importedFeature !== feature) {
        failures.push(`${name}: feature imports private code from ${importedFeature}`);
      }
      if ((inFeatureLogic || inFeatureUI) && (
        imported.includes('/web/src/app/')
        || imported.includes('/web/src/infrastructure/')
        || imported.includes('/web/src/shared/')
      )) {
        failures.push(`${name}: feature imports composition or infrastructure`);
      }
      if (inEntity && !imported.includes('/web/src/entities/')) {
        failures.push(`${name}: entity imports an outer layer`);
      }
      if (inShared && (
        imported.includes('/web/src/app/')
        || imported.includes('/web/src/entities/')
        || imported.includes('/web/src/features/')
        || imported.includes('/web/src/infrastructure/')
      )) {
        failures.push(`${name}: shared imports an inner or outer application layer`);
      }
      if (inInfrastructure && (
        imported.includes('/web/src/app/')
        || imported.includes('/web/src/features/') && imported.includes('/ui/')
      )) {
        failures.push(`${name}: infrastructure imports app or feature UI`);
      }
    }

    if (inFeatureLogic && /\b(?:window|navigator|location|globalThis)\s*\./.test(contents)) {
      failures.push(`${name}: feature logic uses browser globals`);
    }
    if ((inEntity || inFeatureLogic || inFeatureUI) && /cmd\/app|frontend\/bindings|VITE_SINGLEPAGE_RUNTIME/.test(contents)) {
      failures.push(`${name}: runtime selection leaked inward`);
    }
    if (!path.endsWith('/web/src/infrastructure/api/page-api.ts') && contents.includes('/api/pages')) {
      failures.push(`${name}: REST path literal exists outside the API adapter`);
    }
  }

  return failures;
}

export function checkArchitecture(directory = sourceRoot) {
  return architectureFailures(sourceFiles(directory).map((path) => ({ path, contents: readFileSync(path, 'utf8') })));
}

if (fileURLToPath(import.meta.url) === resolve(process.argv[1] ?? '')) {
  const failures = checkArchitecture();
  if (failures.length) {
    console.error(failures.join('\n'));
    process.exitCode = 1;
  }
}
