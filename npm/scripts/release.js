const https = require('https');
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { execSync } = require('child_process');
const tar = require('tar');

const REPO = 'divyo-argha/git-user';
const PKG_JSON = require('../package.json');
const VERSION = `v${PKG_JSON.version}`;

// Map our optionalDependencies package names to GitHub release asset names
const platforms = [
  { pkg: 'git-userhub-darwin-x64', os: 'darwin', cpu: 'x64', assetOS: 'darwin', assetArch: 'x86_64', ext: '' },
  { pkg: 'git-userhub-darwin-arm64', os: 'darwin', cpu: 'arm64', assetOS: 'darwin', assetArch: 'arm64', ext: '' },
  { pkg: 'git-userhub-linux-x64', os: 'linux', cpu: 'x64', assetOS: 'linux', assetArch: 'x86_64', ext: '' },
  { pkg: 'git-userhub-linux-arm64', os: 'linux', cpu: 'arm64', assetOS: 'linux', assetArch: 'arm64', ext: '' },
  { pkg: 'git-userhub-win32-x64', os: 'win32', cpu: 'x64', assetOS: 'windows', assetArch: 'x86_64', ext: '.exe' }
];

const PACKAGES_DIR = path.join(__dirname, '..', 'packages');

async function getReleaseData(version) {
  return new Promise((resolve, reject) => {
    https.get(`https://api.github.com/repos/${REPO}/releases/tags/${version}`, {
      headers: { 'User-Agent': 'node' }
    }, (res) => {
      if (res.statusCode !== 200) return reject(new Error(`GitHub API returned ${res.statusCode}`));
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => resolve(JSON.parse(data)));
    }).on('error', reject);
  });
}

function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    https.get(url, { headers: { 'User-Agent': 'node' } }, (res) => {
      if (res.statusCode === 301 || res.statusCode === 302) {
        return downloadFile(res.headers.location, dest).then(resolve).catch(reject);
      }
      if (res.statusCode !== 200) return reject(new Error(`Download failed: ${res.statusCode}`));
      
      const file = fs.createWriteStream(dest);
      res.pipe(file);
      file.on('finish', () => { file.close(); resolve(); });
    }).on('error', reject);
  });
}

function verifyChecksum(file, expectedHash) {
  return new Promise((resolve, reject) => {
    const hash = crypto.createHash('sha256');
    const stream = fs.createReadStream(file);
    stream.on('data', data => hash.update(data));
    stream.on('end', () => {
      const actualHash = hash.digest('hex');
      if (actualHash !== expectedHash) {
        reject(new Error(`Checksum mismatch! Expected ${expectedHash}, got ${actualHash}`));
      } else {
        resolve();
      }
    });
  });
}

async function run() {
  console.log(`🚀 Starting release process for ${VERSION}...`);
  
  if (fs.existsSync(PACKAGES_DIR)) {
    fs.rmSync(PACKAGES_DIR, { recursive: true, force: true });
  }
  fs.mkdirSync(PACKAGES_DIR);

  console.log(`🔍 Fetching release info from GitHub...`);
  const release = await getReleaseData(VERSION);
  
  const checksumAsset = release.assets.find(a => a.name.endsWith('checksums.txt'));
  if (!checksumAsset) throw new Error('checksums.txt not found in release');
  
  const checksumFile = path.join(PACKAGES_DIR, 'checksums.txt');
  await downloadFile(checksumAsset.browser_download_url, checksumFile);
  const checksums = fs.readFileSync(checksumFile, 'utf8');

  for (const plat of platforms) {
    const assetName = `git-user_${plat.assetOS}_${plat.assetArch}.tar.gz`;
    const asset = release.assets.find(a => a.name === assetName);
    if (!asset) throw new Error(`Asset ${assetName} not found in release`);

    // Parse checksum
    const hashLine = checksums.split('\n').find(line => line.endsWith(assetName));
    if (!hashLine) throw new Error(`Checksum for ${assetName} not found`);
    const expectedHash = hashLine.split(/\s+/)[0];

    const pkgDir = path.join(PACKAGES_DIR, plat.pkg);
    const pkgBinDir = path.join(pkgDir, 'bin');
    fs.mkdirSync(pkgBinDir, { recursive: true });

    const archivePath = path.join(pkgDir, assetName);
    console.log(`⬇️  Downloading ${assetName}...`);
    await downloadFile(asset.browser_download_url, archivePath);
    await verifyChecksum(archivePath, expectedHash);

    console.log(`📂 Extracting ${assetName}...`);
    const binaryNameInArchive = plat.os === 'win32' ? 'git-user.exe' : 'git-user';
    await tar.extract({ 
      file: archivePath, 
      cwd: pkgBinDir,
      filter: (p) => p.replace(/^\.\//, '') === binaryNameInArchive
    });
    fs.unlinkSync(archivePath);

    // Make executable
    const finalBinPath = path.join(pkgBinDir, binaryNameInArchive);
    if (plat.os !== 'win32') fs.chmodSync(finalBinPath, 0o755);

    // Write package.json
    const subPkgJson = {
      name: plat.pkg,
      version: PKG_JSON.version,
      description: `The ${plat.os} ${plat.cpu} binary for git-userhub`,
      os: [plat.os],
      cpu: [plat.cpu],
      bin: {
        "git-user": `bin/${binaryNameInArchive}`
      },
      repository: PKG_JSON.repository,
      license: PKG_JSON.license
    };
    fs.writeFileSync(path.join(pkgDir, 'package.json'), JSON.stringify(subPkgJson, null, 2));

    // Write simple README
    fs.writeFileSync(path.join(pkgDir, 'README.md'), `# ${plat.pkg}\nThis package contains the native binary for git-userhub on ${plat.os} ${plat.cpu}.\n\nThis is an internal package and shouldn't be installed directly. Install \`git-userhub\` instead.`);

    // Publish sub-package
    console.log(`📦 Publishing ${plat.pkg} to npm...`);
    execSync('npm publish --access public', { cwd: pkgDir, stdio: 'inherit' });
  }

  // Finally, publish the main package
  console.log(`📦 Publishing main git-userhub package to npm...`);
  execSync('npm publish', { cwd: path.join(__dirname, '..'), stdio: 'inherit' });

  // Clean up
  console.log('🧹 Cleaning up...');
  fs.rmSync(PACKAGES_DIR, { recursive: true, force: true });
  
  console.log(`✅ All done! v${PKG_JSON.version} is fully published.`);
}

run().catch(err => {
  console.error(`❌ Failed:`, err.message);
  process.exit(1);
});
