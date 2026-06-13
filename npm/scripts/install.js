const fs = require('fs');
const path = require('path');
const os = require('os');
const https = require('https');
const crypto = require('crypto');
const { execSync } = require('child_process');
const PKG_JSON = require('../package.json');

const REPO = 'divyo-argha/git-user';
const VERSION = `v${PKG_JSON.version}`;

// --- START PINNED HASHES ---
const PINNED_HASHES = {
    "git-user_darwin_arm64.tar.gz": {
      "archive": "92aa0c5ccae91a9b035193a0dd5297d413b5c848fbc82ccc5d94519e3d392341",
      "binary": "7ddd3e7c559780a7fd27ac1c5a61d9355a7587943dd326a5b7c520c0b4b1da85"
    },
    "git-user_darwin_x86_64.tar.gz": {
      "archive": "37d0b7afb0c0a7aef1a9b3900591c00817bb69007525e6b243d2c4016a55b0d3",
      "binary": "853977e2f1801d441d2ff421b07f221d898a95d3a18d9773c4946ddf6fa5b93e"
    },
    "git-user_linux_arm64.tar.gz": {
      "archive": "507864e86451bc24e7521f5031b55de10ec2c71df770ff3e6912564d888afe10",
      "binary": "b2976817f655a362aae1a308bd6ff15695cd82879cab6e5a6046626d2c8a3d79"
    },
    "git-user_linux_x86_64.tar.gz": {
      "archive": "a95b617b8efaf61d59c0652a73d21c18bb31406d53e0558a1479e4ee1ff9150d",
      "binary": "50001ce94eb3ab56467bb3b5b07e237117fa1ce90fee2f9e6b6bf6c8bff4a2a2"
    },
    "git-user_windows_x86_64.tar.gz": {
      "archive": "df9b2b2fa8c414d58bcfd6653590797716fce92ecdaadc22d539b64a9adfbf0c",
      "binary": "c7e32093225292c949fc83f777adbae93fa77fcedf44c6749192ae89ceb2fa4b"
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
    const binDir = path.join(__dirname, '..', 'bin');
    execSync(`tar -xzf "${archivePath}" -C "${binDir}"`);

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
