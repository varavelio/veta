#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");

/**
 * Returns the installed Veta binary path.
 * @returns {string} Absolute path to the Veta binary.
 */
function getBinaryPath() {
  const binaryName = process.platform === "win32" ? "veta.exe" : "veta";
  const binaryPath = path.join(__dirname, "bin", binaryName);

  if (!fs.existsSync(binaryPath)) {
    throw new Error(
      `Veta binary not found at ${binaryPath}. `
        + "Installation may have failed. Try reinstalling: npm install --global @varavel/veta",
    );
  }

  return binaryPath;
}

/**
 * Returns the installed npm package version.
 * @returns {string} Package version without a v prefix.
 */
function getVersion() {
  const packageJson = require("./package.json");
  return packageJson.version.replace(/^v/, "");
}

module.exports = {
  getBinaryPath,
  getVersion,
};

if (require.main === module) {
  console.log(getBinaryPath());
}
