import { execFileSync } from 'node:child_process';
import { existsSync } from 'node:fs';

export default function globalSetup() {
  if (!existsSync('../raspicture')) throw new Error('Build the server first: run `make` in the repo root');
  if (!existsSync('.fixtures/DCIM')) execFileSync('./make-fixtures.sh', ['.fixtures'], { stdio: 'inherit' });
}
