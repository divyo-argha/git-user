#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');

// Get the binary path
const binaryName = process.platform === 'win32' ? 'git-user.exe' : 'git-user';
const binPath = path.join(__dirname, '..', 'bin', binaryName);

// Forward all arguments to the binary
const child = spawn(binPath, process.argv.slice(2), {
  stdio: 'inherit',
  shell: false
});

child.on('exit', (code) => {
  process.exit(code || 0);
});
