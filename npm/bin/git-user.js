#!/usr/bin/env node

const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');
const os = require('os');

// Detect platform and architecture
const platform = os.platform();
const arch = os.arch();

const platformMap = {
  'darwin': 'darwin',
  'linux': 'linux',
  'win32': 'windows'
};

const archMap = {
  'x64': 'amd64',
  'arm64': 'arm64'
};

const osName = platformMap[platform];
const archName = archMap[arch];
const ext = platform === 'win32' ? '.exe' : '';

if (!osName || !archName) {
  console.error(`❌ Unsupported platform: ${platform} ${arch}`);
  process.exit(1);
}

const pkgName = `git-userhub-${platform}-${arch}`;

let binaryPath;
try {
  // Find the sub-package directory by resolving its package.json
  const subPkgPath = require.resolve(`${pkgName}/package.json`);
  binaryPath = path.join(path.dirname(subPkgPath), 'bin', `git-user${ext}`);
} catch (e) {
  console.error(`❌ git-user native binary not installed!`);
  console.error(`   npm should have installed the optional dependency '${pkgName}'.`);
  console.error(`   Please try reinstalling the package.`);
  process.exit(1);
}

if (!fs.existsSync(binaryPath)) {
  console.error(`❌ git-user binary not found at ${binaryPath}`);
  console.error(`   Please ensure the package was correctly installed.`);
  process.exit(1);
}

// Forward all arguments to the bundled binary
const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  shell: false
});

child.on('exit', (code) => {
  process.exit(code || 0);
});

child.on('error', (err) => {
  console.error('❌ Failed to start git-user:', err.message);
  process.exit(1);
});
