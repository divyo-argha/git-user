const fs = require('fs');
const path = require('path');
const https = require('https');
const crypto = require('crypto');
const { execSync } = require('child_process');
const PKG_JSON = require('../package.json');

const REPO = 'divyo-argha/git-user';
const VERSION = `v${PKG_JSON.version}`;

const ASSETS = [
  { name: `git-user_darwin_arm64.tar.gz`, bin: `git-user` },
  { name: `git-user_darwin_x86_64.tar.gz`, bin: `git-user` },
  { name: `git-user_linux_arm64.tar.gz`, bin: `git-user` },
  { name: `git-user_linux_x86_64.tar.gz`, bin: `git-user` },
  { name: `git-user_windows_x86_64.tar.gz`, bin: `git-user.exe` }
];

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

function computeHash(file) {
  return new Promise((resolve, reject) => {
    const hash = crypto.createHash('sha256');
    const stream = fs.createReadStream(file);
    stream.on('data', d => hash.update(d));
    stream.on('end', () => resolve(hash.digest('hex')));
    stream.on('error', reject);
  });
}

async function run() {
  console.log(`🔒 Pinning cryptographic hashes for ${VERSION}...`);
  const hashes = {};
  
  try {
    for (const asset of ASSETS) {
      console.log(`Downloading ${asset.name}...`);
      const url = `https://github.com/${REPO}/releases/download/${VERSION}/${asset.name}`;
      const archivePath = path.join(__dirname, asset.name);
      
      await fetchFile(url, archivePath);
      
      const archiveHash = await computeHash(archivePath);
      
      // Extract to get the binary hash
      execSync(`tar -xzf "${archivePath}" -C "${__dirname}"`);
      const extractedPath = path.join(__dirname, asset.bin);
      
      const binaryHash = await computeHash(extractedPath);
      
      hashes[`${asset.name}`] = {
        archive: archiveHash,
        binary: binaryHash
      };
      
      // Cleanup
      fs.unlinkSync(archivePath);
      fs.unlinkSync(extractedPath);
    }

    const targetFile = path.join(__dirname, 'install.js');
    let code = fs.readFileSync(targetFile, 'utf8');

    const hashesJson = JSON.stringify(hashes, null, 2).replace(/\n/g, '\n  ');
    
    const startMarker = '// --- START PINNED HASHES ---';
    const endMarker = '// --- END PINNED HASHES ---';
    
    const startIndex = code.indexOf(startMarker);
    const endIndex = code.indexOf(endMarker);
    
    if (startIndex === -1 || endIndex === -1) {
      throw new Error("Could not find PINNED HASHES markers in scripts/install.js");
    }

    const newCode = code.substring(0, startIndex + startMarker.length) + 
                    '\nconst PINNED_HASHES = ' + hashesJson + ';\n' + 
                    code.substring(endIndex);

    fs.writeFileSync(targetFile, newCode);
    console.log(`✅ Successfully injected cryptographic pins into scripts/install.js`);
  } catch (err) {
    console.error(`❌ Failed to inject hashes: ${err.message}`);
    process.exit(1);
  }
}

run();
