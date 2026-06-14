const fs = require('fs');
const path = require('path');
const os = require('os');
const https = require('https');
const crypto = require('crypto');
const { spawnSync } = require('child_process');
const PKG_JSON = require('../package.json');

const REPO = 'divyo-argha/git-user';
const VERSION = `v${PKG_JSON.version}`;

// --- START PINNED HASHES ---
const PINNED_HASHES = {
    "git-user_darwin_arm64.tar.gz": {
      "archive": "8f9e59d21977a7f923c027d3312c01ce81771efe3e2bb1dabf837c384504b20c",
      "binary": "0cd7c9be0be6bd22a5dc8adb6c82ca73a5718b182fa3aba0d7e41a2c3ec08d18"
    },
    "git-user_darwin_x86_64.tar.gz": {
      "archive": "4abbe23be36683c76815defc7efdb94157dee946e7e01a529cdd724e4d6118da",
      "binary": "4d8a0127ca8f0f616612c4617cf1bd3824a556e1c92fb401858cb7460e393b81"
    },
    "git-user_linux_arm64.tar.gz": {
      "archive": "9ddfe15bace025795489e6b545dc4dc9a50a24863aa3eb504e2325d5049c84fd",
      "binary": "6d7a710f699a92baa632263a26f34aeb761e373cd0a84a69c5c2215d487188b2"
    },
    "git-user_linux_x86_64.tar.gz": {
      "archive": "088aac2526c6f3a13ece1892d51696e4be12e03937d6cd0f74b50907ac14d716",
      "binary": "855f46795ed5b18f6463049d29ac868772cf00e48a911d5f15d34ef7b28e4e37"
    },
    "git-user_windows_x86_64.tar.gz": {
      "archive": "b206786fea35395a5911631f03f17f69b6ed9dae401b7c3eea8e3b076e9fee55",
      "binary": "d3034bd2b48e5a95282679ee27b4c3b714f20e0ddacb19b6982ffa75167cf05f"
    }
  };
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
const finalBinaryPath = path.join(__dirname, '..', 'bin', finalBinaryName);
const assetName = `git-user_${osName}_${archName}.tar.gz`;
const pinnedData = PINNED_HASHES[assetName];

function computeHash(file) {
  return new Promise((resolve, reject) => {
    const hash = crypto.createHash('sha256');
    const stream = fs.createReadStream(file);
    stream.on('data', d => hash.update(d));
    stream.on('end', () => resolve(hash.digest('hex')));
    stream.on('error', reject);
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

async function install() {
  if (fs.existsSync(finalBinaryPath)) {
    return; // Already installed
  }

  console.log(`[git-user] Downloading cryptographically signed binary for ${platform}-${arch}...`);

  try {
    const archivePath = path.join(__dirname, '..', 'bin', assetName);
    const scheme = 'https';
    const host = 'github.com';
    const downloadUrl = `${scheme}://${host}/${REPO}/releases/download/${VERSION}/${assetName}`;

    await fetchFile(downloadUrl, archivePath);
    
    if (pinnedData) {
      const archiveHash = await computeHash(archivePath);
      if (archiveHash !== pinnedData.archive) {
        throw new Error("Archive checksum mismatch! Connection may be compromised.");
      }
    }

    const binaryNameInArchive = platform === 'win32' ? 'git-user.exe' : 'git-user';
    const binDir = path.join(__dirname, '..', 'bin');
    const result = spawnSync('tar', ['-xzf', archivePath, '-C', binDir]);
    if (result.error || result.status !== 0) {
      throw new Error("Failed to extract tar archive");
    }

    const extractedPath = path.join(__dirname, '..', 'bin', binaryNameInArchive);
    fs.renameSync(extractedPath, finalBinaryPath);

    if (platform !== 'win32') fs.chmodSync(finalBinaryPath, 0o755);

    fs.unlinkSync(archivePath);

    if (pinnedData) {
      const binaryHash = await computeHash(finalBinaryPath);
      if (binaryHash !== pinnedData.binary) {
        fs.unlinkSync(finalBinaryPath);
        throw new Error("Binary checksum mismatch! Payload was modified during extraction.");
      }
    }

    console.log(`[git-user] Installation and verification complete.\n`);
  } catch (err) {
    console.error(`\n❌ git-user installation failed: ${err.message}`);
    throw err;
  }
}

if (require.main === module) {
  install().catch(() => process.exit(1));
} else {
  module.exports = install;
}
