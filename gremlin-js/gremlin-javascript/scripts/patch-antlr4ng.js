#!/usr/bin/env node
/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

/**
 * Patches antlr4ng to add CJS TypeScript declaration files.
 *
 * antlr4ng ships a CJS bundle (dist/index.cjs) but lacks the accompanying
 * dist/index.d.cts type declarations that TypeScript's NodeNext module
 * resolution requires when compiling CJS output. This script creates the
 * missing .d.cts file and updates the package exports map so TypeScript
 * correctly resolves types in both ESM and CJS contexts.
 *
 * This is a stop-gap until antlr4ng ships .d.cts files natively.
 * Relevant issues/discussions:
 * - https://github.com/mike-lischke/antlr4ng/issues/23 (CJS/ESM dual module support)
 * - https://github.com/mike-lischke/antlr4ng/issues/26 (TypeScript resolution issues)
 * - https://github.com/mike-lischke/antlr4ng/discussions/21 (Discussion on package structure)
 */

import { copyFileSync, existsSync, readFileSync, writeFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { createRequire } from 'node:module';

// Use createRequire so Node resolves antlr4ng through the normal module
// resolution algorithm, which handles npm workspace hoisting correctly.
const require = createRequire(import.meta.url);

let pkgDir;
try {
  // Resolve antlr4ng's main entry, then walk up to the package root.
  pkgDir = resolve(dirname(require.resolve('antlr4ng')), '..');
} catch {
  console.warn('\n[patch-antlr4ng] WARNING: antlr4ng not found. TypeScript CJS type resolution may be broken.\n');
  process.exit(0); // non-fatal: let the build surface its own errors
}

const distDir = resolve(pkgDir, 'dist');
const dts = resolve(distDir, 'index.d.ts');
const cjs = resolve(distDir, 'index.cjs');
const mjs = resolve(distDir, 'index.mjs');
const pkgPath = resolve(pkgDir, 'package.json');

// Verify expected files exist before patching — loud failure if antlr4ng restructured its dist/
const required = [dts, cjs, mjs, pkgPath];
const missing = required.filter((f) => !existsSync(f));
if (missing.length > 0) {
  console.error(
    '\n[patch-antlr4ng] ERROR: antlr4ng dist layout has changed — patch cannot be applied.\n' +
      'Missing files:\n' +
      missing.map((f) => '  ' + f).join('\n') +
      '\n' +
      'Check whether antlr4ng has fixed CJS .d.cts support (issues #23/#26) and update or remove this script.\n',
  );
  process.exit(1);
}

// Copy index.d.ts → index.d.cts
const dcts = resolve(distDir, 'index.d.cts');
copyFileSync(dts, dcts);

// Update exports map to include "types" in the "require" condition.
// NOTE: this overwrites only the keys we know about from antlr4ng 3.0.16.
// If antlr4ng adds new export conditions in a future version, review this script.
const pkg = JSON.parse(readFileSync(pkgPath, 'utf8'));
pkg.exports = {
  types: './dist/index.d.ts',
  require: { types: './dist/index.d.cts', default: './dist/index.cjs' },
  import: { types: './dist/index.d.ts', default: './dist/index.mjs' },
};
writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n');

console.log('[patch-antlr4ng] patched: added dist/index.d.cts and updated exports map');
