#!/usr/bin/env node

const { execFileSync } = require("node:child_process");
const crypto = require("node:crypto");
const fs = require("node:fs");
const https = require("node:https");
const path = require("node:path");
const manifest = require("./manifest.json");

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

/**
 * Returns the platform name used by Veta release assets.
 * @returns {string} Release platform name.
 */
function getPlatform() {
  const platform = PLATFORM_MAP[process.platform];
  if (!platform) {
    throw new Error(`Unsupported platform: ${process.platform}. Veta supports darwin, linux, and win32.`);
  }
  return platform;
}

/**
 * Returns the architecture name used by Veta release assets.
 * @returns {string} Release architecture name.
 */
function getArch() {
  const arch = ARCH_MAP[process.arch];
  if (!arch) {
    throw new Error(`Unsupported architecture: ${process.arch}. Veta supports x64 and arm64.`);
  }
  if (process.platform === "win32" && arch === "arm64") {
    throw new Error("Unsupported platform: Veta does not currently publish Windows arm64 binaries.");
  }
  return arch;
}

/**
 * Returns the package version without a v prefix.
 * @returns {string} Package version.
 */
function getVersion() {
  const packageJson = require("./package.json");
  return packageJson.version.replace(/^v/, "");
}

/**
 * Returns the platform-specific Veta binary name.
 * @returns {string} Binary filename.
 */
function getBinaryName() {
  return process.platform === "win32" ? "veta.exe" : "veta";
}

/**
 * Returns the release archive filename for this platform.
 * @returns {string} Release archive filename.
 */
function getReleaseFilename() {
  const platform = getPlatform();
  const arch = getArch();
  const extension = platform === "windows" ? "zip" : "tar.gz";
  return `veta_${platform}_${arch}.${extension}`;
}

/**
 * Returns the GitHub release download URL for this platform.
 * @returns {string} Download URL.
 */
function getDownloadURL() {
  const version = getVersion();
  return `https://github.com/varavelio/veta/releases/download/v${version}/${getReleaseFilename()}`;
}

/**
 * Returns the release artifact metadata for filename.
 * @param {string} filename Expected release archive filename.
 * @returns {{ sha256: string }} Release artifact metadata.
 */
function getManifestArtifact(filename) {
  if (!Array.isArray(manifest.artifacts)) {
    throw new Error("Release manifest is missing artifacts.");
  }

  const artifact = manifest.artifacts.find((item) => item.name === filename);
  if (!artifact || !artifact.sha256) {
    throw new Error(`Release manifest does not include ${filename}.`);
  }

  return artifact;
}

/**
 * Verifies an archive buffer against the checksum embedded in this npm package manifest.
 * @param {Buffer} archiveBuffer Downloaded archive bytes.
 * @param {string} filename Expected release archive filename.
 * @returns {void}
 */
function verifyChecksum(archiveBuffer, filename) {
  const expectedHash = getManifestArtifact(filename).sha256;
  const actualHash = crypto.createHash("sha256").update(archiveBuffer).digest("hex");
  if (expectedHash !== actualHash) {
    throw new Error(
      `Checksum verification failed for ${filename}.\nExpected: ${expectedHash}\nActual:   ${actualHash}`,
    );
  }
}

/**
 * Ensures and returns the package-local binary directory.
 * @returns {string} Binary directory path.
 */
function ensureBinDir() {
  const binDir = path.join(__dirname, "bin");
  fs.mkdirSync(binDir, { recursive: true });
  return binDir;
}

/**
 * Extracts a tar.gz archive into the package binary directory.
 * @param {Buffer} buffer Archive bytes.
 * @param {string} binDir Destination directory.
 * @param {string} binaryName Expected binary filename.
 * @returns {void}
 */
function extractTarGz(buffer, binDir, binaryName) {
  const tempName = Date.now().toString();
  const tempArchive = path.join(binDir, `${tempName}.tar.gz`);
  const tempDir = path.join(binDir, `${tempName}-extract`);

  try {
    fs.writeFileSync(tempArchive, buffer);
    fs.mkdirSync(tempDir, { recursive: true });
    execFileSync("tar", ["-xzf", tempArchive, "-C", tempDir], { stdio: "pipe" });
    installExtractedBinary(tempDir, binDir, binaryName);
  } finally {
    fs.rmSync(tempArchive, { force: true });
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
}

/**
 * Extracts a zip archive into the package binary directory.
 * @param {Buffer} buffer Archive bytes.
 * @param {string} binDir Destination directory.
 * @param {string} binaryName Expected binary filename.
 * @returns {void}
 */
function extractZip(buffer, binDir, binaryName) {
  const tempName = Date.now().toString();
  const tempArchive = path.join(binDir, `${tempName}.zip`);
  const tempDir = path.join(binDir, `${tempName}-extract`);

  try {
    fs.writeFileSync(tempArchive, buffer);
    fs.mkdirSync(tempDir, { recursive: true });
    if (process.platform === "win32") {
      execFileSync(
        "powershell",
        [
          "-NoProfile",
          "-Command",
          "Expand-Archive -LiteralPath $args[0] -DestinationPath $args[1] -Force",
          tempArchive,
          tempDir,
        ],
        { stdio: "pipe" },
      );
    } else {
      execFileSync("unzip", ["-q", tempArchive, "-d", tempDir], { stdio: "pipe" });
    }
    installExtractedBinary(tempDir, binDir, binaryName);
  } finally {
    fs.rmSync(tempArchive, { force: true });
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
}

/**
 * Copies the extracted binary into the package binary directory.
 * @param {string} tempDir Extraction directory.
 * @param {string} binDir Destination binary directory.
 * @param {string} binaryName Expected binary filename.
 * @returns {void}
 */
function installExtractedBinary(tempDir, binDir, binaryName) {
  const source = path.join(tempDir, binaryName);
  if (!fs.existsSync(source)) {
    throw new Error(`Binary ${binaryName} not found in release archive`);
  }

  const destination = path.join(binDir, binaryName);
  fs.copyFileSync(source, destination);
  if (process.platform !== "win32") {
    fs.chmodSync(destination, 0o755);
  }
}

/**
 * Downloads a URL as a Buffer, following redirects.
 * @param {string} url URL to download.
 * @returns {Promise<Buffer>} Downloaded bytes.
 */
function download(url) {
  return new Promise((resolve, reject) => {
    https
      .get(url, (response) => {
        if ([301, 302, 307, 308].includes(response.statusCode)) {
          download(response.headers.location).then(resolve).catch(reject);
          return;
        }
        if (response.statusCode !== 200) {
          reject(new Error(`Failed to download ${url}: HTTP ${response.statusCode}`));
          return;
        }

        const chunks = [];
        response.on("data", (chunk) => chunks.push(chunk));
        response.on("end", () => resolve(Buffer.concat(chunks)));
        response.on("error", reject);
      })
      .on("error", reject);
  });
}

/**
 * Downloads and installs the Veta binary for this platform.
 * @returns {Promise<void>}
 */
async function install() {
  const platform = getPlatform();
  const binaryName = getBinaryName();
  const filename = getReleaseFilename();
  const binDir = ensureBinDir();

  try {
    console.log(`Veta: Downloading ${filename}...`);
    const archiveBuffer = await download(getDownloadURL());

    console.log("Veta: Verifying checksum...");
    verifyChecksum(archiveBuffer, filename);

    console.log("Veta: Extracting...");
    if (platform === "windows") {
      extractZip(archiveBuffer, binDir, binaryName);
    } else {
      extractTarGz(archiveBuffer, binDir, binaryName);
    }

    console.log("Veta: Installation complete. Run 'veta --version' to verify.");
  } catch (error) {
    console.error("Veta: Installation failed");
    console.error(`Veta: ${error.message}`);
    process.exit(1);
  }
}

if (require.main === module) install();
module.exports = { install };
