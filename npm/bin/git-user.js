#!/usr/bin/env node

const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');
const os = require('os');
const https = require('https');
const crypto = require('crypto');
const tar = require('tar');
const PKG_JSON = require('../package.json');

const REPO = 'divyo-argha/git-user';
const VERSION = `v${PKG_JSON.version}`;

const platform = os.platform();
const arch = os.arch();

const platformMap = { 'darwin': 'darwin', 'linux': 'linux', 'win32': 'windows' };
const archMap = { 'x64': 'x86_64', 'arm64': 'arm64' };

const osName = platformMap[platform];
const archName = archMap[arch];
const ext = platform === 'win32' ? '.exe' : '';

if (!osName || !archName) {
  console.error(`❌ Unsupported platform: ${platform} ${arch}`);
  process.exit(1);
}

const finalBinaryName = `git-user-${platform}-${arch}${ext}`;
const finalBinaryPath = path.join(__dirname, finalBinaryName);
const assetName = `git-user_${osName}_${archName}.tar.gz`;

function runBinary() {
  const child = spawn(finalBinaryPath, process.argv.slice(2), {
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
}

if (fs.existsSync(finalBinaryPath)) {
  runBinary();
  return;
}

// FIRST RUN: Download binary
console.log(`[git-user] First run detected. Downloading native binary for ${platform}-${arch} (~8MB)...`);

function fetchJson(url) {
  return new Promise((resolve, reject) => {
    https.get(url, { headers: { 'User-Agent': 'node' } }, (res) => {
      if (res.statusCode !== 200) return reject(new Error(`API Error ${res.statusCode}`));
      let data = '';
      res.on('data', c => data += c);
      res.on('end', () => resolve(JSON.parse(data)));
    }).on('error', reject);
  });
}

function fetchFile(url, dest) {
  return new Promise((resolve, reject) => {
    https.get(url, { headers: { 'User-Agent': 'node' } }, (res) => {
      if (res.statusCode === 301 || res.statusCode === 302) {
        return fetchFile(res.headers.location, dest).then(resolve).catch(reject);
      }
      if (res.statusCode !== 200) return reject(new Error(`Download Error ${res.statusCode}`));
      const file = fs.createWriteStream(dest);
      res.pipe(file);
      file.on('finish', () => { file.close(); resolve(); });
    }).on('error', reject);
  });
}

function verifyHash(file, expected) {
  return new Promise((resolve, reject) => {
    const hash = crypto.createHash('sha256');
    const stream = fs.createReadStream(file);
    stream.on('data', d => hash.update(d));
    stream.on('end', () => {
      if (hash.digest('hex') !== expected) reject(new Error("Checksum mismatch!"));
      else resolve();
    });
  });
}

async function install() {
  try {
    const release = await fetchJson(`https://api.github.com/repos/${REPO}/releases/tags/${VERSION}`);
    const checksumAsset = release.assets.find(a => a.name.endsWith('checksums.txt'));
    const binAsset = release.assets.find(a => a.name === assetName);

    if (!checksumAsset || !binAsset) {
      throw new Error("Could not find required assets on GitHub Release");
    }

    const checksumPath = path.join(__dirname, 'checksums.txt');
    const archivePath = path.join(__dirname, assetName);

    await fetchFile(checksumAsset.browser_download_url, checksumPath);
    const checksums = fs.readFileSync(checksumPath, 'utf8');
    const hashLine = checksums.split('\n').find(l => l.endsWith(assetName));
    if (!hashLine) throw new Error("Checksum missing in txt file");
    const expectedHash = hashLine.split(/\s+/)[0];

    await fetchFile(binAsset.browser_download_url, archivePath);
    await verifyHash(archivePath, expectedHash);

    const binaryNameInArchive = platform === 'win32' ? 'git-user.exe' : 'git-user';
    await tar.extract({ 
      file: archivePath, 
      cwd: __dirname,
      filter: (p) => p.replace(/^\.\//, '') === binaryNameInArchive
    });

    const extractedPath = path.join(__dirname, binaryNameInArchive);
    fs.renameSync(extractedPath, finalBinaryPath);

    if (platform !== 'win32') fs.chmodSync(finalBinaryPath, 0o755);

    fs.unlinkSync(archivePath);
    fs.unlinkSync(checksumPath);

    console.log(`[git-user] Installation complete.\n`);
    runBinary();
  } catch (err) {
    console.error(`\n❌ git-user installation failed: ${err.message}`);
    process.exit(1);
  }
}

install();
