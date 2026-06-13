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

async function run() {
  if (!fs.existsSync(finalBinaryPath)) {
    console.log('📦 First run detected. Downloading binary...');
    const install = require('../scripts/install.js');
    try {
      await install();
    } catch (e) {
      console.error('❌ Failed to install git-user binary.');
      process.exit(1);
    }
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
}

run();
