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

// --- START PINNED HASHES ---
const PINNED_HASHES = {};
// --- END PINNED HASHES ---

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
const pinnedData = PINNED_HASHES[assetName];

function computeHashSync(filePath) {
  const hash = crypto.createHash('sha256');
  const buffer = fs.readFileSync(filePath);
  hash.update(buffer);
  return hash.digest('hex');
}

function computeHash(file) {
  return new Promise((resolve, reject) => {
    const hash = crypto.createHash('sha256');
    const stream = fs.createReadStream(file);
    stream.on('data', d => hash.update(d));
    stream.on('end', () => resolve(hash.digest('hex')));
    stream.on('error', reject);
  });
}

function runBinary() {
  if (pinnedData) {
    const currentHash = computeHashSync(finalBinaryPath);
    if (currentHash !== pinnedData.binary) {
      console.error(`❌ Security Error: Cached binary hash mismatch! Deleting compromised binary.`);
      fs.unlinkSync(finalBinaryPath);
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

if (fs.existsSync(finalBinaryPath)) {
  runBinary();
  return;
}

// FIRST RUN: Download binary
console.log(`[git-user] Downloading cryptographically signed binary for ${platform}-${arch}...`);

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

async function install() {
  try {
    const archivePath = path.join(__dirname, assetName);
    const downloadUrl = `https://github.com/${REPO}/releases/download/${VERSION}/${assetName}`;

    await fetchFile(downloadUrl, archivePath);
    
    if (pinnedData) {
      const archiveHash = await computeHash(archivePath);
      if (archiveHash !== pinnedData.archive) {
        throw new Error("Archive checksum mismatch! Connection may be compromised.");
      }
    }

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

    console.log(`[git-user] Installation and verification complete.\n`);
    runBinary();
  } catch (err) {
    console.error(`\n❌ git-user installation failed: ${err.message}`);
    process.exit(1);
  }
}

install();
