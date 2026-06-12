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
const PINNED_HASHES = {
    "git-user_darwin_arm64.tar.gz": {
      "archive": "434850b54bc77e777b7b57416bd029fc3600906addb8036759572d2279081080",
      "binary": "3210a67ac54cbe675c31ef8c4c9e515262253a006c60205f115063c314c91113"
    },
    "git-user_darwin_x86_64.tar.gz": {
      "archive": "85deb7ccc4d487d78c22453b57061d447ca79101d8192e3c66d92539361fdae8",
      "binary": "a75202a115d5c2b65581e18f8a71d2273cbac2708bb38fd37a1626141023e733"
    },
    "git-user_linux_arm64.tar.gz": {
      "archive": "e97a1d23ca85228d7097f313f60be86b692a9a4002d91cd691e3e6ccb913a34f",
      "binary": "eb714726d488b3eb39a1de8b21a8ce48a24eff8bb39ba8a1d332615bcb956e1f"
    },
    "git-user_linux_x86_64.tar.gz": {
      "archive": "e30e0b709c70fa79bb44dcfb7193157ff626e63664ed704cf35f663cecd321e7",
      "binary": "3e37645becc5a2928808a41cdadb492106fc2df2ace18be1d11774a3d636ca71"
    },
    "git-user_windows_x86_64.tar.gz": {
      "archive": "32a771b04e588d87dcb1ce7d8dd5303cc8332080686f55688b5054f4c7f0403c",
      "binary": "570ec56a28741cd5e515aee1b2c18583dd77ea4527b08c68f75d38e80b0bee29"
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
      cwd: path.join(__dirname, '..', 'bin'),
      filter: (p) => p.replace(/^\.\//, '') === binaryNameInArchive
    });

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
    process.exit(1);
  }
}

install();
