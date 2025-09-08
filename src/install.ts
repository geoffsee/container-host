#!/usr/bin/env node

import { platform, arch } from "os";
import { existsSync } from "fs";
import { join } from "path";

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

function checkBinaryAvailability(): boolean {
  const packageName = getPlatformPackageName();
  
  try {
    const packagePath = require.resolve(`${packageName}/package.json`);
    const packageDir = packagePath.replace('/package.json', '');
    const packageJson = require(packagePath);
    const binaryPath = join(packageDir, packageJson.main);
    
    return existsSync(binaryPath);
  } catch (error) {
    return false;
  }
}

function main() {
  try {
    const packageName = getPlatformPackageName();
    
    if (checkBinaryAvailability()) {
      console.log(`✓ Binary for ${platform()}-${arch()} is available via ${packageName}`);
      process.exit(0);
    } else {
      console.log(`⚠ Binary for ${platform()}-${arch()} not found.`);
      console.log(`Expected platform package: ${packageName}`);
      console.log(`This might be due to the optional dependency not being installed.`);
      console.log(`The CLI will attempt to locate the binary at runtime.`);
      process.exit(0);
    }
  } catch (error) {
    console.error("Install check failed:", error instanceof Error ? error.message : error);
    console.log("The CLI will attempt to locate the binary at runtime.");
    process.exit(0); // Don't fail the install process
  }
}

if (require.main === module) {
  main();
}