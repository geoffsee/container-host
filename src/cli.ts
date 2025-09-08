#!/usr/bin/env node

import { spawn } from "child_process";
import { join } from "path";
import { platform, arch } from "os";
import { existsSync } from "fs";

function getPlatformPackageName(): string {
  const platformName = platform();
  const archName = arch();
  
  let platformSuffix: string;
  let archSuffix: string;

  switch (platformName) {
    case "darwin":
      platformSuffix = "darwin";
      break;
    case "linux":
      platformSuffix = "linux";
      break;
    case "win32":
      platformSuffix = "win32";
      break;
    default:
      throw new Error(`Unsupported platform: ${platformName}`);
  }

  switch (archName) {
    case "x64":
      archSuffix = "x64";
      break;
    case "arm64":
      archSuffix = "arm64";
      break;
    default:
      throw new Error(`Unsupported architecture: ${archName}`);
  }

  return `container-host-cli-${platformSuffix}-${archSuffix}`;
}

function getBinaryPath(): string {
  const packageName = getPlatformPackageName();
  
  // Try to find the binary in the platform-specific package
  try {
    const packagePath = require.resolve(`${packageName}/package.json`);
    const packageDir = packagePath.replace('/package.json', '');
    const packageJson = require(packagePath);
    const binaryPath = join(packageDir, packageJson.main);
    
    if (existsSync(binaryPath)) {
      return binaryPath;
    }
  } catch (error) {
    // Platform package not found, fall back to error
  }
  
  throw new Error(`Binary not found for platform ${platform()}-${arch()}. Please ensure the correct platform package is installed.`);
}

function main() {
  try {
    const binaryPath = getBinaryPath();
    
    const child = spawn(binaryPath, process.argv.slice(2), {
      stdio: "inherit",
    });
    
    child.on("exit", (code) => {
      process.exit(code || 0);
    });
    
    child.on("error", (error) => {
      if (error.message.includes("ENOENT")) {
        console.error(`Binary not found at: ${binaryPath}`);
        console.error("Please ensure the binary is installed correctly.");
        process.exit(1);
      } else {
        console.error("Error running container-host:", error.message);
        process.exit(1);
      }
    });
  } catch (error) {
    console.error("Error:", error instanceof Error ? error.message : error);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}