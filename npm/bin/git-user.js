#!/usr/bin/env node

const { spawnSync } = require('child_process');
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
  const localPkgPath = path.join(__dirname, '..', 'packages', packageName, 'bin', `git-user${ext}`);
  if (fs.existsSync(localPkgPath)) {
    binPath = localPkgPath;
  } else {
    console.error(`Unsupported platform or architecture: ${platform}-${arch}`);
    console.error('git-userhub currently supports macOS, Linux, and Windows on x64 and arm64 architectures.');
    process.exit(1);
  }
}

if (!fs.existsSync(binPath)) {
  console.error(`Error: Platform-specific binary not found at: ${binPath}`);
  process.exit(1);
}

// Spawn the binary
const result = spawnSync(binPath, process.argv.slice(2), {
  stdio: 'inherit'
});

if (result.error) {
  console.error('Failed to start git-user:', result.error);
  process.exit(1);
}

process.exit(result.status ?? 1);
