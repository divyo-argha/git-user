const https = require('https');
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const tar = require('tar');
const pkg = require('../package.json');

const REPO = pkg.config.repo;
const BIN_DIR = path.join(__dirname, '..', 'bin');

const PLATFORMS = [
  { os: 'darwin', arch: 'amd64', assetSubstring: 'darwin_x86_64' },
  { os: 'darwin', arch: 'arm64', assetSubstring: 'darwin_arm64' },
  { os: 'linux', arch: 'amd64', assetSubstring: 'linux_x86_64' },
  { os: 'linux', arch: 'arm64', assetSubstring: 'linux_arm64' },
  { os: 'windows', arch: 'amd64', assetSubstring: 'windows_x86_64' }
];

function fetchText(url, redirectCount = 0) {
  return new Promise((resolve, reject) => {
    if (redirectCount > 3) return reject(new Error('Too many redirects'));
    const parsedUrl = new URL(url);
    if (parsedUrl.protocol !== 'https:') return reject(new Error('Only HTTPS is allowed'));

    https.get(url, { headers: { 'User-Agent': 'git-user-cli' } }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        return fetchText(res.headers.location, redirectCount + 1).then(resolve).catch(reject);
      }
      if (res.statusCode !== 200) return reject(new Error(`Failed to fetch: ${res.statusCode} ${url}`));
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => resolve(data));
    }).on('error', reject);
  });
}

function downloadAndVerify(url, dest, expectedHash, redirectCount = 0) {
  return new Promise((resolve, reject) => {
    if (redirectCount > 3) return reject(new Error('Too many redirects'));
    const parsedUrl = new URL(url);
    if (parsedUrl.protocol !== 'https:') return reject(new Error('Only HTTPS is allowed'));

    const file = fs.createWriteStream(dest);
    const hash = crypto.createHash('sha256');
    
    https.get(url, { headers: { 'User-Agent': 'git-user-cli' } }, (response) => {
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        file.close();
        fs.unlink(dest, () => {});
        return downloadAndVerify(response.headers.location, dest, expectedHash, redirectCount + 1)
          .then(resolve).catch(reject);
      }
      if (response.statusCode !== 200) {
        file.close();
        fs.unlink(dest, () => {});
        return reject(new Error(`Failed to download: ${response.statusCode} ${url}`));
      }
      
      response.on('data', chunk => hash.update(chunk));
      response.pipe(file);
      file.on('finish', () => {
        file.close();
        const actualHash = hash.digest('hex');
        if (actualHash !== expectedHash) {
          fs.unlink(dest, () => {});
          return reject(new Error(`Checksum mismatch for ${dest}! Expected ${expectedHash}, got ${actualHash}`));
        }
        resolve();
      });
    }).on('error', (err) => {
      file.close();
      fs.unlink(dest, () => {});
      reject(err);
    });
  });
}

function getRelease() {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: 'api.github.com',
      path: `/repos/${REPO}/releases/tags/v${pkg.version}`,
      headers: { 'User-Agent': 'git-user-cli' }
    };
    https.get(options, (res) => {
      if (res.statusCode !== 200) return reject(new Error(`GitHub API returned ${res.statusCode} for release v${pkg.version}`));
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => resolve(JSON.parse(data)));
    }).on('error', reject);
  });
}

async function main() {
  if (!fs.existsSync(BIN_DIR)) {
    fs.mkdirSync(BIN_DIR, { recursive: true });
  }

  console.log(`🔍 Fetching release info for v${pkg.version}...`);
  const release = await getRelease();

  const checksumAsset = release.assets?.find(a => a.name === 'checksums.txt');
  const checksumsUrl = checksumAsset ? checksumAsset.browser_download_url : `https://github.com/${REPO}/releases/download/v${pkg.version}/checksums.txt`;
  
  console.log('🔐 Fetching checksums...');
  const checksumsText = await fetchText(checksumsUrl);
  const checksumMap = {};
  checksumsText.split('\n').forEach(line => {
    const parts = line.trim().split(/\s+/);
    if (parts.length >= 2) checksumMap[parts[1]] = parts[0];
  });

  for (const plat of PLATFORMS) {
    const asset = release.assets?.find(a => a.name.toLowerCase().includes(plat.assetSubstring.toLowerCase()));
    if (!asset) {
      console.warn(`⚠️  Warning: No asset found for ${plat.os} ${plat.arch}. Skipping.`);
      continue;
    }
    
    const expectedHash = checksumMap[asset.name];
    if (!expectedHash) {
      throw new Error(`Checksum for ${asset.name} not found in checksums.txt`);
    }

    const archivePath = path.join(BIN_DIR, asset.name);
    console.log(`⬇️  Downloading ${asset.name} (verifying SHA256 checksum)...`);
    await downloadAndVerify(asset.browser_download_url, archivePath, expectedHash);
    
    console.log(`📂 Extracting ${asset.name}...`);
    const isWindows = plat.os === 'windows';
    const binaryNameInArchive = isWindows ? 'git-user.exe' : 'git-user';
    const newBinaryName = `git-user-${plat.os}-${plat.arch}${isWindows ? '.exe' : ''}`;
    
    await tar.extract({ 
      file: archivePath, 
      cwd: BIN_DIR,
      filter: (p) => {
        const base = p.replace(/^\.\//, '');
        return base === binaryNameInArchive;
      }
    });
    
    fs.unlinkSync(archivePath);
    
    // Rename to target name
    const extractedPath = path.join(BIN_DIR, binaryNameInArchive);
    const finalPath = path.join(BIN_DIR, newBinaryName);
    fs.renameSync(extractedPath, finalPath);
    
    if (!isWindows) {
      fs.chmodSync(finalPath, 0o755);
    }
    console.log(`✅ ${newBinaryName} ready.`);
  }

  console.log('🎉 All binaries downloaded and prepared successfully!');
}

main().catch(err => {
  console.error('❌ Failed:', err.message);
  process.exit(1);
});
