// Bakes the game's art and audio (run from anywhere):
//   node tools/bake/bake.mjs          everything
//   node tools/bake/bake.mjs cards    just the card faces, props and icons
// Needs Node, Playwright's Chromium and ffmpeg (music is stored as Ogg Vorbis).
import { chromium } from 'playwright';
import { writeFileSync, mkdirSync, rmSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const only = process.argv[2] || '';
const browser = await chromium.launch({ args: ['--allow-file-access-from-files', '--autoplay-policy=no-user-gesture-required'] });
const page = await browser.newPage();
page.on('pageerror', e => console.error('page error:', e.message));
page.on('console', m => console.log('page:', m.text()));
await page.goto('file://' + root + '/tools/bake/bake.html' + (only ? '?only=' + only : ''));
await page.waitForFunction(() => document.title !== '', null, { timeout: 900000 });
const title = await page.title();
if (title !== 'ready') { console.error(title); process.exit(1); }
const out = await page.evaluate(() => window.bake);
await browser.close();

for (const [name, url] of Object.entries(out.images)) {
  const f = `${root}/${name}`; mkdirSync(dirname(f), { recursive: true });
  writeFileSync(f, Buffer.from(url.split(',')[1], 'base64'));
}
if (Object.keys(out.audio).length) {
  mkdirSync(`${root}/assets/audio`, { recursive: true });
  for (const [name, b64] of Object.entries(out.audio)) {
    const wav = `${root}/assets/audio/${name}.wav`;
    writeFileSync(wav, Buffer.from(b64, 'base64'));
    if (name.startsWith('tune-')) {   // music -> Ogg Vorbis to keep the exe small
      execFileSync('ffmpeg', ['-y', '-loglevel', 'error', '-i', wav, '-c:a', 'libvorbis', '-q:a', '5', wav.replace('.wav', '.ogg')]);
      rmSync(wav);
    }
  }
  writeFileSync(`${root}/assets/audio/tunes.txt`, out.tuneNames.join('\n') + '\n');
  console.log('peaks', JSON.stringify(out.peaks));
}
console.log('baked', Object.keys(out.images).length, 'images and', Object.keys(out.audio).length, 'sounds');
