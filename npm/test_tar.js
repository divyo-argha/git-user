const fs = require('fs');
const crypto = require('crypto');
const tar = require('tar');

const assetName = "git-user_darwin_arm64.tar.gz";

async function main() {
  console.log("Creating dummy tar file using node-tar...");
  // create dummy git-user
  fs.writeFileSync('git-user', 'dummy binary');
  fs.writeFileSync('malicious.sh', 'echo bad');
  
  await tar.create({
    gzip: true,
    file: assetName,
  }, ['git-user', 'malicious.sh']);
  
  console.log("Tar created.");

  const file = fs.readFileSync(assetName);
  const expectedHash = crypto.createHash('sha256').update(file).digest('hex');
  
  fs.unlinkSync('git-user');
  fs.unlinkSync('malicious.sh');

  console.log("Expected hash:", expectedHash);

  // Now extract securely
  fs.mkdirSync('bin', { recursive: true });
  
  await tar.extract({
    file: assetName,
    cwd: 'bin',
    filter: (p, entry) => {
      console.log('Filtering path:', p, entry.path);
      return p === 'git-user' || p === 'git-user.exe' || p === './git-user' || p === './git-user.exe';
    }
  });

  const binContents = fs.readdirSync('bin');
  console.log("Bin contents:", binContents);
}

main().catch(console.error);
