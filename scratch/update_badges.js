const fs = require('fs');
const files = ['README.md', 'npm/README.md'];

for (const file of files) {
  let content = fs.readFileSync(file, 'utf8');
  content = content.replace(/style=flat-square/g, 'style=flat');
  content = content.replace(/style=for-the-badge/g, 'style=flat');
  fs.writeFileSync(file, content);
  console.log(`Updated ${file}`);
}
