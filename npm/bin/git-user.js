#!/usr/bin/env node

const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');
const os = require('os');

const platform = os.platform();
const arch = os.arch();
const ext = platform === 'win32' ? '.exe' : '';

const finalBinaryName = `git-user-${platform}-${arch}${ext}`;
const finalBinaryPath = path.join(__dirname, finalBinaryName);

if (!fs.existsSync(finalBinaryPath)) {
  console.error(`❌ git-user binary not found at ${finalBinaryPath}`);
  console.error('Please reinstall the package: npm install -g git-userhub');
  process.exit(1);
}

const child = spawn(finalBinaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  shell: false
});

child.on('exit', (code) => process.exit(code || 0));
child.on('error', (err) => {
  console.error('❌ Failed to start git-user:', err.message);
  process.exit(1);
});
