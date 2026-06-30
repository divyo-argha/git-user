const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const targets = [
  { os: 'darwin', arch: 'amd64', ext: '' },
  { os: 'darwin', arch: 'arm64', ext: '' },
  { os: 'linux', arch: 'amd64', ext: '' },
  { os: 'linux', arch: 'arm64', ext: '' },
  { os: 'windows', arch: 'amd64', ext: '.exe' }
];

const binDir = path.join(__dirname, '..', 'bin');

if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

const rootDir = path.join(__dirname, '..', '..');

console.log('Building binaries for fat package...');

for (const target of targets) {
  const nodeOs = target.os === 'windows' ? 'win32' : target.os;
  const nodeArch = target.arch === 'amd64' ? 'x64' : target.arch;
  const outDir = path.join(binDir, `${nodeOs}-${nodeArch}`);
  
  if (!fs.existsSync(outDir)) {
    fs.mkdirSync(outDir, { recursive: true });
  }

  const outPath = path.join(outDir, `git-user${target.ext}`);
  console.log(`Compiling ${nodeOs}-${nodeArch}...`);
  
  try {
    execSync(`GOOS=${target.os} GOARCH=${target.arch} go build -ldflags="-s -w" -o "${outPath}" .`, {
      cwd: rootDir,
      stdio: 'inherit'
    });
  } catch (error) {
    console.error(`Failed to compile for ${nodeOs}-${nodeArch}:`, error.message);
    process.exit(1);
  }
}
console.log('Done compiling binaries.');
