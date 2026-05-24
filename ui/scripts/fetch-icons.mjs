import { mkdirSync, existsSync, rmSync } from 'node:fs';
import { execSync } from 'node:child_process';

const COMMIT = process.env.ICONS_COMMIT;
if (!COMMIT) {
  console.error('Set ICONS_COMMIT env var to the dashboard-icons commit SHA to pin.');
  process.exit(1);
}

const dest = 'icons';
if (existsSync(dest)) rmSync(dest, { recursive: true, force: true });
mkdirSync(dest, { recursive: true });

const tarUrl = `https://codeload.github.com/homarr-labs/dashboard-icons/tar.gz/${COMMIT}`;
console.log('Downloading', tarUrl);
execSync(
  `curl -fsSL ${tarUrl} | tar -xz --strip-components=2 -C ${dest} dashboard-icons-${COMMIT}/png`,
  { stdio: 'inherit', shell: '/bin/bash' },
);
console.log('Done.');
