#!/usr/bin/env node

const { spawn } = require('child_process');
const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');
const crypto = require('crypto');
const tar = require('tar');
const pkg = require('../package.json');

const REPO = pkg.config.repo;
const BIN_DIR = path.join(__dirname, '..', 'bin');

// Detect platform and architecture
function getPlatform() {
  const platform = os.platform();
  const arch = os.arch();
  
  const platformMap = {
    'darwin': 'darwin',
    'linux': 'linux',
    'win32': 'windows'
  };
  
  const archMap = {
    'x64': 'amd64',
    'arm64': 'arm64'
  };
  
  return {
    os: platformMap[platform],
    arch: archMap[arch],
    ext: platform === 'win32' ? '.exe' : ''
  };
}

// Download file to memory (for checksums.txt)
function fetchText(url, redirectCount = 0) {
  return new Promise((resolve, reject) => {
    if (redirectCount > 3) return reject(new Error('Too many redirects'));
    
    const parsedUrl = new URL(url);
    if (parsedUrl.protocol !== 'https:') {
      return reject(new Error('Only HTTPS is allowed'));
    }

    https.get(url, { headers: { 'User-Agent': 'git-user-cli' } }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        return fetchText(res.headers.location, redirectCount + 1).then(resolve).catch(reject);
      }
      if (res.statusCode !== 200) {
        return reject(new Error(`Failed to fetch: ${res.statusCode} ${url}`));
      }
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => resolve(data));
    }).on('error', reject);
  });
}

// Download file from URL, and verify hash
function downloadAndVerify(url, dest, expectedHash, redirectCount = 0) {
  return new Promise((resolve, reject) => {
    if (redirectCount > 3) return reject(new Error('Too many redirects'));
    
    const parsedUrl = new URL(url);
    if (parsedUrl.protocol !== 'https:') {
      return reject(new Error('Only HTTPS is allowed'));
    }

    const file = fs.createWriteStream(dest);
    const hash = crypto.createHash('sha256');
    
    https.get(url, { headers: { 'User-Agent': 'git-user-cli' } }, (response) => {
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        file.close();
        fs.unlink(dest, () => {});
        return downloadAndVerify(response.headers.location, dest, expectedHash, redirectCount + 1)
          .then(resolve)
          .catch(reject);
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
          return reject(new Error(`Checksum mismatch! Expected ${expectedHash}, got ${actualHash}`));
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

// Get release info matching this npm package version.
function getRelease() {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: 'api.github.com',
      path: `/repos/${REPO}/releases/tags/v${pkg.version}`,
      headers: {
        'User-Agent': 'git-user-cli'
      }
    };
    
    https.get(options, (res) => {
      if (res.statusCode !== 200) {
        return reject(new Error(`GitHub API returned ${res.statusCode} for release v${pkg.version}`));
      }
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => {
        try {
          resolve(JSON.parse(data));
        } catch (err) {
          reject(err);
        }
      });
    }).on('error', reject);
  });
}

async function installAndRun() {
  const { os: osName, arch, ext } = getPlatform();
  const binaryPath = path.join(BIN_DIR, `git-user${ext}`);

  if (!fs.existsSync(binaryPath)) {
    console.log('📦 First run detected. Downloading git-user binary...');
    
    if (!osName || !arch) {
      console.error('❌ Unsupported platform:', os.platform(), os.arch());
      process.exit(1);
    }
    
    try {
      if (!fs.existsSync(BIN_DIR)) {
        fs.mkdirSync(BIN_DIR, { recursive: true });
      }
      
      console.log(`🔍 Fetching release v${pkg.version}...`);
      const release = await getRelease();
      
      const asset = release.assets?.find(a => {
        const name = a.name.toLowerCase();
        if (!name.includes(osName)) return false;
        if (arch === 'arm64') return name.includes('arm64');
        return name.includes('x86_64') || name.includes('amd64') || name.includes('x64');
      });
      
      if (!asset) {
        console.error('❌ No binary found for your platform');
        console.error(`   Looking for a ${osName} binary matching architecture: ${arch}`);
        process.exit(1);
      }

      console.log('🔐 Fetching checksums...');
      // Fetch checksums.txt from the release assets or release tag
      // GitHub Releases typically attach checksums.txt if goreleaser is used
      const checksumAsset = release.assets?.find(a => a.name === 'checksums.txt');
      let checksumsUrl = '';
      if (checksumAsset) {
        checksumsUrl = checksumAsset.browser_download_url;
      } else {
        // Fallback pattern if asset list doesn't have it, try direct URL
        checksumsUrl = `https://github.com/${REPO}/releases/download/v${pkg.version}/checksums.txt`;
      }
      
      const checksumsText = await fetchText(checksumsUrl);
      
      // Parse checksums.txt to find the hash for our asset
      const expectedHashLine = checksumsText.split('\n').find(line => line.includes(asset.name));
      if (!expectedHashLine) {
        throw new Error(`Checksum for ${asset.name} not found in checksums.txt`);
      }
      const expectedHash = expectedHashLine.trim().split(/\s+/)[0];
      
      console.log(`⬇️  Downloading ${asset.name} (verifying SHA256 checksum)...`);
      const archivePath = path.join(BIN_DIR, asset.name);
      await downloadAndVerify(asset.browser_download_url, archivePath, expectedHash);
      
      console.log('📂 Extracting securely...');
      await tar.extract({ 
        file: archivePath, 
        cwd: BIN_DIR,
        filter: (p) => p === `git-user${ext}` || p === `./git-user${ext}`
      });
      
      fs.unlinkSync(archivePath);
      
      if (fs.existsSync(binaryPath)) {
        fs.chmodSync(binaryPath, 0o755);
        console.log('✅ git-user installed successfully!\n');
      } else {
        console.error('❌ Binary not found after extraction');
        process.exit(1);
      }
    } catch (err) {
      console.error('❌ Installation failed:', err.message);
      process.exit(1);
    }
  }

  // Forward all arguments to the binary
  const child = spawn(binaryPath, process.argv.slice(2), {
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

installAndRun();
