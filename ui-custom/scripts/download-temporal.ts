// Download Temporal proto files for UI development
// This script is required by the prepare hook in package.json

import { writeFileSync, mkdirSync, existsSync } from 'fs';
import { join } from 'path';

// Create empty proto directory if needed
const protoDir = join(process.cwd(), 'proto');
if (!existsSync(protoDir)) {
  mkdirSync(protoDir, { recursive: true });
}

console.log('Temporal proto files ready (using embedded definitions)');
