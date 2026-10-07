// Drives headless Brave over the DevTools protocol, for checking the front end without a window.
// Usage: node scripts/cdp.mjs <url> <width> <height> '[{"wait":500},{"click":[x,y]},{"type":"text"},{"key":{"key":"Enter","code":"Enter"}},{"eval":"js"},{"shot":"name.png"}]'
// A shot path that is not absolute lands in $TMPDIR, the session's own temp folder.
// Drives headless Brave over the DevTools protocol: node cdp.mjs <url> <w> <h> <steps-json>
import { spawn } from 'node:child_process'
import { writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
const [url, w, h, stepsJson] = process.argv.slice(2)
const port = 9333
const brave = spawn('/Applications/Brave Browser.app/Contents/MacOS/Brave Browser',
  ['--headless=new', `--remote-debugging-port=${port}`, `--window-size=${w},${h}`, '--hide-scrollbars', `--user-data-dir=${join(tmpdir(), 'agentos-cdp-profile')}`, 'about:blank'], { stdio: 'ignore' })
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
let target
for (let i = 0; i < 50 && !target; i++) {
  await sleep(200)
  try { target = (await (await fetch(`http://127.0.0.1:${port}/json`)).json()).find((t) => t.type === 'page') } catch {}
}
const ws = new WebSocket(target.webSocketDebuggerUrl)
await new Promise((r) => (ws.onopen = r))
let id = 0; const pending = new Map(); const logs = []
ws.onmessage = (m) => {
  const d = JSON.parse(m.data)
  if (d.id && pending.has(d.id)) { pending.get(d.id)(d.result ?? d.error); pending.delete(d.id) }
  if (d.method === 'Runtime.consoleAPICalled') logs.push(d.params.type + ': ' + d.params.args.map((a) => a.value ?? a.description).join(' '))
  if (d.method === 'Runtime.exceptionThrown') logs.push('EXCEPTION: ' + (d.params.exceptionDetails.exception?.description ?? d.params.exceptionDetails.text))
}
const send = (method, params = {}) => new Promise((r) => { pending.set(++id, r); ws.send(JSON.stringify({ id, method, params })) })
await send('Runtime.enable'); await send('Page.enable')
await send('Emulation.setDeviceMetricsOverride', { width: +w, height: +h, deviceScaleFactor: 1, mobile: false })
await send('Page.navigate', { url })
for (const step of JSON.parse(stepsJson)) {
  if (step.wait) await sleep(step.wait)
  if (step.eval) { const r = await send('Runtime.evaluate', { expression: step.eval, awaitPromise: true, returnByValue: true }); console.log('eval:', JSON.stringify(r.result?.value ?? r.exceptionDetails?.exception?.description)) }
  if (step.click) { const [x, y] = step.click; for (const type of ['mousePressed', 'mouseReleased']) await send('Input.dispatchMouseEvent', { type, x, y, button: 'left', clickCount: 1 }) }
  if (step.type) await send('Input.insertText', { text: step.type })
  if (step.key) { const k = step.key; for (const type of ['keyDown', 'keyUp']) await send('Input.dispatchKeyEvent', { type, ...k }) }
  if (step.shot) { const r = await send('Page.captureScreenshot', { format: 'png' }); const file = resolve(tmpdir(), step.shot); writeFileSync(file, Buffer.from(r.data, 'base64')); console.log('shot:', file) }
}
console.log(logs.slice(0, 20).join('\n'))
ws.close(); brave.kill(); process.exit(0)
