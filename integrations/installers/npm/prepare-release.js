#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");

const REQUIRED_ARCHIVES = [
  "veta_darwin_amd64.tar.gz",
  "veta_darwin_arm64.tar.gz",
  "veta_linux_amd64.tar.gz",
  "veta_linux_arm64.tar.gz",
  "veta_windows_amd64.zip",
];

/**
 * Normalizes a release version for npm package.json.
 * @param {string} version Version argument.
 * @returns {string} Normalized version without a v prefix.
 */
function normalizeVersion(version) {
  return version.trim().replace(/^v/, "");
}

/**
 * Parses checksums.txt into an archive checksum map.
 * @param {string} content checksums.txt content.
 * @returns {Record<string, string>} Archive checksum map.
 */
function parseChecksums(content) {
  const checksums = {};
  for (const line of content.trim().split("\n")) {
    const trimmed = line.trim();
    if (!trimmed) continue;

    const parts = trimmed.split(/\s+/);
    if (parts.length !== 2) throw new Error(`Invalid checksum line: ${line}`);

    const [hash, name] = parts;
    if (REQUIRED_ARCHIVES.includes(name)) checksums[name] = hash;
  }

  for (const name of REQUIRED_ARCHIVES) {
    if (!checksums[name]) throw new Error(`Missing checksum for ${name}`);
  }

  return checksums;
}

/**
 * Writes JSON with the package formatting convention.
 * @param {string} filePath Destination path.
 * @param {unknown} value JSON value.
 * @returns {void}
 */
function writeJSON(filePath, value) {
  fs.writeFileSync(filePath, `${JSON.stringify(value, null, 2)}\n`);
}

/**
 * Prepares the npm package for one release publish.
 * @param {string} version Release version.
 * @param {string} checksumsPath Path to release checksums.txt.
 * @returns {void}
 */
function prepareRelease(version, checksumsPath) {
  const packagePath = path.join(__dirname, "package.json");
  const checksumsJSONPath = path.join(__dirname, "checksums.json");
  const packageJSON = JSON.parse(fs.readFileSync(packagePath, "utf8"));

  packageJSON.version = normalizeVersion(version);
  const checksums = parseChecksums(fs.readFileSync(checksumsPath, "utf8"));

  writeJSON(packagePath, packageJSON);
  writeJSON(checksumsJSONPath, checksums);
}

if (require.main === module) {
  const [version, checksumsPath] = process.argv.slice(2);
  if (!version || !checksumsPath) {
    console.error("Usage: node prepare-release.js <version> <checksums.txt>");
    process.exit(1);
  }

  try {
    prepareRelease(version, checksumsPath);
  } catch (error) {
    console.error(error.message);
    process.exit(1);
  }
}

module.exports = { normalizeVersion, parseChecksums, prepareRelease };
