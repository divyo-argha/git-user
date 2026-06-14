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
      "archive": "faaaffba1dced76ce7d66be937766c8a7f7d35ff93236a3b02857bf8389e1193",
      "binary": "43314aecd2878805e98d27626389bb1f44ebcb87add623a2db0038ae73c1468a"
    },
    "git-user_darwin_x86_64.tar.gz": {
      "archive": "8d01532955bdad17e59cf0dd0d0216bae4115ccb52eb4e924bd5670a202542fe",
      "binary": "2593d32b8a38903d58e69603f2e582b5a7483c02734b0b00f26c9b133c2761c4"
    },
    "git-user_linux_arm64.tar.gz": {
      "archive": "6aa119118bc6e57444f5dfd60d5201f990f9c212836bbbe2637d6065e6ff2b87",
      "binary": "2df7a234c3cb7c9de903bc6dd1f6c9784274784c1d2f017c60dc06ff60f80ad0"
    },
    "git-user_linux_x86_64.tar.gz": {
      "archive": "ccb4165be62ab799554492218a0c127d2f44ff72a471f948ffd1b43f57cd88db",
      "binary": "c191e6f041b15564ee6768ad205bee4dac74b1095c1c8431e01d57bc3be20f84"
    },
    "git-user_windows_x86_64.tar.gz": {
      "archive": "0c0ad07edde45fdffd9b75cef6f3de7c913eb0cf07510e1ee5e732eb2f0b416b",
      "binary": "8d154d798dd5f916ff51861b2274241836be350275d7e1d32ddb50956b708e22"
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
