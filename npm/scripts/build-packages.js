const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const targets = [
  { os: 'darwin', arch: 'amd64', nodeOs: 'darwin', nodeArch: 'x64', ext: '' },
  { os: 'darwin', arch: 'arm64', nodeOs: 'darwin', nodeArch: 'arm64', ext: '' },
  { os: 'linux', arch: 'amd64', nodeOs: 'linux', nodeArch: 'x64', ext: '' },
  { os: 'linux', arch: 'arm64', nodeOs: 'linux', nodeArch: 'arm64', ext: '' },
  { os: 'windows', arch: 'amd64', nodeOs: 'win32', nodeArch: 'x64', ext: '.exe' }
];

const npmDir = path.join(__dirname, '..');
const packagesDir = path.join(npmDir, 'packages');
const rootDir = path.join(npmDir, '..');

// Read version from main package.json
const mainPkg = JSON.parse(fs.readFileSync(path.join(npmDir, 'package.json'), 'utf8'));
const version = mainPkg.version;
const date = new Date().toISOString().split('T')[0];

console.log(`Building packages for version ${version}...`);

// Recreate packages directory
if (fs.existsSync(packagesDir)) {
  fs.rmSync(packagesDir, { recursive: true, force: true });
}
fs.mkdirSync(packagesDir, { recursive: true });

for (const target of targets) {
  const pkgName = `git-userhub-${target.nodeOs}-${target.nodeArch}`;
  const pkgDir = path.join(packagesDir, pkgName);
  const pkgBinDir = path.join(pkgDir, 'bin');
  
  fs.mkdirSync(pkgBinDir, { recursive: true });
  
  const outPath = path.join(pkgBinDir, `git-user${target.ext}`);
  console.log(`Compiling binary for ${pkgName}...`);
  
  try {
    execSync(`GOOS=${target.os} GOARCH=${target.arch} go build -ldflags="-s -w -X main.version=${version} -X main.date=${date}" -o "${outPath}" .`, {
      cwd: rootDir,
      stdio: 'inherit'
    });
  } catch (error) {
    console.error(`Failed to compile binary for ${pkgName}:`, error.message);
    process.exit(1);
  }
  
  // Write package.json for sub-package
  const subPkgJson = {
    name: pkgName,
    version: version,
    description: `Platform-specific binary of git-userhub for ${target.nodeOs} ${target.nodeArch}`,
    repository: mainPkg.repository,
    license: mainPkg.license,
    os: [target.nodeOs],
    cpu: [target.nodeArch]
  };
  
  fs.writeFileSync(
    path.join(pkgDir, 'package.json'),
    JSON.stringify(subPkgJson, null, 2) + '\n'
  );
  
  // Copy LICENSE and README if they exist
  const licensePath = path.join(rootDir, 'LICENSE');
  if (fs.existsSync(licensePath)) {
    fs.copyFileSync(licensePath, path.join(pkgDir, 'LICENSE'));
  }
  
  const readmeContent = `# ${pkgName}\n\nPlatform-specific binary package of \`git-userhub\` for ${target.nodeOs} ${target.nodeArch}.\n`;
  fs.writeFileSync(path.join(pkgDir, 'README.md'), readmeContent);
}

console.log('Successfully built all packages under npm/packages/.');
