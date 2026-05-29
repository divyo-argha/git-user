#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');
const tar = require('tar');
const pkg = require('./package.json');

const REPO = pkg.config.repo;
const BIN_DIR = path.join(__dirname, 'bin');

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

// Download file from URL
function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    
    https.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Follow redirect
        return download(response.headers.location, dest).then(resolve).catch(reject);
      }
      
      if (response.statusCode !== 200) {
        reject(new Error(`Failed to download: ${response.statusCode}`));
        return;
      }
      
      response.pipe(file);
      
      file.on('finish', () => {
        file.close();
        resolve();
      });
    }).on('error', (err) => {
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
        'User-Agent': 'git-user-npm-installer'
      }
    };
    
    https.get(options, (res) => {
      let data = '';
      
      res.on('data', (chunk) => {
        data += chunk;
      });
      
      res.on('end', () => {
        try {
          const release = JSON.parse(data);
          resolve(release);
        } catch (err) {
          reject(err);
        }
      });
    }).on('error', reject);
  });
}

async function install() {
  console.log('📦 Installing git-user...');
  
  const { os: osName, arch, ext } = getPlatform();
  
  if (!osName || !arch) {
    console.error('❌ Unsupported platform:', os.platform(), os.arch());
    process.exit(1);
  }
  
  try {
    // Create bin directory
    if (!fs.existsSync(BIN_DIR)) {
      fs.mkdirSync(BIN_DIR, { recursive: true });
    }
    
    // Get matching release
    console.log(`🔍 Fetching release v${pkg.version}...`);
    const release = await getRelease();
    
    // Find the right asset dynamically
    const asset = release.assets.find(a => {
      const name = a.name.toLowerCase();
      if (!name.includes(osName)) return false;
      
      if (arch === 'arm64') {
        return name.includes('arm64');
      } else {
        return name.includes('x86_64') || name.includes('amd64') || name.includes('x64');
      }
    });
    
    if (!asset) {
      console.error('❌ No binary found for your platform');
      console.error(`   Looking for a ${osName} binary matching architecture: ${arch}`);
      process.exit(1);
    }
    
    // Download
    console.log(`⬇️  Downloading ${asset.name}...`);
    const archivePath = path.join(BIN_DIR, asset.name);
    await download(asset.browser_download_url, archivePath);
    
    // Extract
    console.log('📂 Extracting...');
    await tar.extract({
      file: archivePath,
      cwd: BIN_DIR
    });
    
    // Cleanup archive
    fs.unlinkSync(archivePath);
    
    // Make executable
    const binaryPath = path.join(BIN_DIR, `git-user${ext}`);
    if (fs.existsSync(binaryPath)) {
      fs.chmodSync(binaryPath, 0o755);
      console.log('✅ git-user installed successfully!');
      console.log('');
      console.log('Quick start:');
      console.log('  npx git-user register    # Create your first identity');
      console.log('  npx git-user switch <n>  # Switch between identities');
      console.log('  npx git-user --help      # Show all commands');
    } else {
      console.error('❌ Binary not found after extraction');
      process.exit(1);
    }
    
  } catch (err) {
    console.error('❌ Installation failed:', err.message);
    process.exit(1);
  }
}

install();
