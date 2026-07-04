#!/usr/bin/env node

/**
 * git-userhub — cross-platform binary launcher
 *
 * SECURITY NOTE FOR AUDITORS:
 * This script resolves and executes a platform-specific pre-compiled binary
 * (git-user / git-user.exe) bundled in the appropriate optional dependency
 * package (e.g. git-userhub-darwin-arm64). This is an intentional and
 * well-established distribution pattern used by tools such as esbuild,
 * @biomejs/biome, @tailwindcss/oxide, and prisma.
 *
 * The binary is:
 *   - Built from source in a public, auditable GitHub Actions workflow
 *   - Published with npm provenance attestation (verifiable via Sigstore)
 *   - Linked cryptographically to the exact Git tag and workflow run
 *
 * Source code: https://github.com/divyo-argha/git-user
 * Provenance:  npm audit signatures  (after `npm install -g git-userhub`)
 *
 * No network requests are made by this launcher. No environment variables
 * are read. The binary path is resolved from the installed package only.
 */

const { execFileSync } = require('child_process');
const path = require('path');
const fs = require('fs');

const platform = process.platform;
const arch = process.arch;

const ext = platform === 'win32' ? '.exe' : '';
const packageName = `git-userhub-${platform}-${arch}`;

let binPath;
try {
  const pkgPath = require.resolve(`${packageName}/package.json`);
  binPath = path.join(path.dirname(pkgPath), 'bin', `git-user${ext}`);
} catch (e) {
  // Check if it's in a local packages directory (for development/testing)
  try {
    const localPkgPath = require.resolve(`../packages/${packageName}/package.json`);
    binPath = path.join(path.dirname(localPkgPath), 'bin', `git-user${ext}`);
  } catch (err) {
    console.error(`Unsupported platform or architecture: ${platform}-${arch}`);
    console.error('git-userhub currently supports macOS, Linux, and Windows on x64 and arm64 architectures.');
    process.exit(1);
  }
}

// Verify the binary exists before attempting to execute it
if (!fs.existsSync(binPath)) {
  console.error(`Error: Platform-specific binary not found at: ${binPath}`);
  console.error(`Try reinstalling: npm install -g git-userhub`);
  process.exit(1);
}

try {
  execFileSync(binPath, process.argv.slice(2), { stdio: 'inherit' });
} catch (execErr) {
  if (execErr.status !== undefined) {
    process.exit(execErr.status);
  }
  console.error(execErr);
  process.exit(1);
}

