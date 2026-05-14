const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

const envExamplePath = path.join(__dirname, '../config/.env.example');
const envPath = path.join(__dirname, '../.env');

function generateJwtSecret() {
  return crypto.randomBytes(64).toString('hex');
}

function copyEnvFile() {
  if (!fs.existsSync(envExamplePath)) {
    console.error('Error: .env.example file not found');
    return false;
  }

  if (fs.existsSync(envPath)) {
    console.log('.env file already exists. Skipping copy.');
    return true;
  }

  const content = fs.readFileSync(envExamplePath, 'utf8');
  const updatedContent = content.replace(
    'JWT_SECRET=your-secret-key-change-in-production',
    `JWT_SECRET=${generateJwtSecret()}`
  );

  fs.writeFileSync(envPath, updatedContent);
  console.log('.env file created successfully');
  return true;
}

function validateRequiredEnvVars() {
  const requiredVars = [
    'DB_HOST',
    'DB_PORT',
    'DB_NAME',
    'DB_USER',
    'DB_PASSWORD',
    'JWT_SECRET',
  ];

  if (!fs.existsSync(envPath)) {
    console.error('Error: .env file not found. Run setup-env.js first.');
    return false;
  }

  const envContent = fs.readFileSync(envPath, 'utf8');
  const missingVars = [];

  for (const varName of requiredVars) {
    const regex = new RegExp(`^${varName}=`, 'm');
    if (!regex.test(envContent)) {
      missingVars.push(varName);
    }
  }

  if (missingVars.length > 0) {
    console.error('Missing required environment variables:');
    missingVars.forEach(v => console.error(`  - ${v}`));
    return false;
  }

  console.log('All required environment variables are set');
  return true;
}

function updateEnvVar(key, value) {
  if (!fs.existsSync(envPath)) {
    console.error('Error: .env file not found. Run setup-env.js first.');
    return false;
  }

  let content = fs.readFileSync(envPath, 'utf8');
  const regex = new RegExp(`^${key}=.*$`, 'm');

  if (regex.test(content)) {
    content = content.replace(regex, `${key}=${value}`);
  } else {
    content += `\n${key}=${value}`;
  }

  fs.writeFileSync(envPath, content);
  console.log(`Updated ${key} in .env file`);
  return true;
}

function main() {
  console.log('Setting up environment variables...\n');

  if (!copyEnvFile()) {
    process.exit(1);
  }

  if (!validateRequiredEnvVars()) {
    process.exit(1);
  }

  console.log('\nEnvironment setup completed successfully!');
  console.log('Please review the .env file and adjust values if needed.');
}

if (require.main === module) {
  main();
}

module.exports = {
  generateJwtSecret,
  copyEnvFile,
  validateRequiredEnvVars,
  updateEnvVar,
};
