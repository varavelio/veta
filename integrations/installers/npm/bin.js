#!/usr/bin/env node

const { spawn } = require("node:child_process");
const { getBinaryPath } = require("./index.js");
const { install } = require("./install.js");

/**
 * Runs the installed Veta binary, installing it first when necessary.
 * @returns {Promise<void>}
 */
async function main() {
  let binaryPath;
  try {
    binaryPath = getBinaryPath();
  } catch (_) {
    await install();
    binaryPath = getBinaryPath();
  }

  // Spawn veta
  const child = spawn(binaryPath, process.argv.slice(2), {
    stdio: ["inherit", "inherit", "inherit"],
  });

  // Propagate OS signals
  for (const signal of ["SIGINT", "SIGTERM", "SIGHUP"]) {
    process.on(signal, () => {
      if (child.pid) child.kill(signal);
    });
  }

  child.on("exit", (code) => {
    process.exit(code || 0);
  });

  child.on("error", (error) => {
    console.error(error.message);
    process.exit(1);
  });
}

main().catch((error) => {
  console.error(error.message);
  process.exit(1);
});
