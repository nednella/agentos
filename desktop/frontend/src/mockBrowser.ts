import type { BrowserInput } from './types'

export type PageModel = {
  url: string
  title: string
  history: string[]
  index: number
  typed: string
  focused: boolean
  submitted: string
  loading: boolean
  width: number
  height: number
  visible: boolean
  dirty: boolean
}

type Rect = { x: number; y: number; w: number; h: number }

const HEADER_H = 56
const PAD = 24

export function normalizeUrl(input: string): string {
  const url = input.trim()
  return /^[a-z][a-z0-9+.-]*:\/\//i.test(url) ? url : `https://${url}`
}

function pathOf(url: string): string {
  try {
    return new URL(url).pathname
  } catch {
    return '/'
  }
}

export function pageTitle(url: string): string {
  const path = pathOf(url)
  const name = { '/': 'Home', '/products': 'Products', '/cart': 'Cart', '/login': 'Sign in' }[path] ?? 'Page'
  return `${name} · Acme Store`
}

export function newPage(url: string): PageModel {
  return {
    url,
    title: pageTitle(url),
    history: [url],
    index: 0,
    typed: '',
    focused: false,
    submitted: '',
    loading: false,
    width: 800,
    height: 500,
    visible: false,
    dirty: true,
  }
}

function layout(w: number, h: number): { input: Rect; button: Rect } {
  const formY = HEADER_H + 112
  const wide = w >= 560
  const inputW = wide ? Math.min(w - PAD * 2 - 130, 440) : w - PAD * 2
  const input = { x: PAD, y: formY, w: inputW, h: 40 }
  const button = wide ? { x: PAD + inputW + 12, y: formY, w: 110, h: 40 } : { x: PAD, y: formY + 52, w: 110, h: 40 }
  return { input, button: { ...button, h: Math.min(button.h, h) } }
}

export function renderPage(p: PageModel, canvas: HTMLCanvasElement, w = p.width, h = p.height): string {
  canvas.width = w
  canvas.height = h
  const ctx = canvas.getContext('2d')!
  const { input, button } = layout(w, h)
  const path = pathOf(p.url)
  ctx.fillStyle = '#f6f7f9'
  ctx.fillRect(0, 0, w, h)
  ctx.fillStyle = '#1f2937'
  ctx.fillRect(0, 0, w, HEADER_H)
  ctx.fillStyle = '#ffffff'
  ctx.font = '600 18px sans-serif'
  ctx.fillText('Acme Store', PAD, 34)
  ctx.font = '14px sans-serif'
  ctx.fillStyle = '#cbd5e1'
  ;['Products', 'Cart', 'Sign in'].forEach((label, i) => ctx.fillText(label, w - 300 + i * 96, 33))
  ctx.fillStyle = '#111827'
  ctx.font = '700 30px sans-serif'
  ctx.fillText({ '/': 'Welcome back', '/products': 'All products', '/cart': 'Your cart', '/login': 'Sign in' }[path] ?? 'Not found', PAD, HEADER_H + 58)
  ctx.font = '14px sans-serif'
  ctx.fillStyle = '#4b5563'
  ctx.fillText('Search everything in the shop.', PAD, HEADER_H + 84)

  ctx.fillStyle = '#ffffff'
  ctx.strokeStyle = p.focused ? '#4f46e5' : '#cbd5e1'
  ctx.lineWidth = p.focused ? 2 : 1
  ctx.fillRect(input.x, input.y, input.w, input.h)
  ctx.strokeRect(input.x, input.y, input.w, input.h)
  ctx.font = '16px sans-serif'
  ctx.fillStyle = p.typed ? '#111827' : '#9ca3af'
  ctx.fillText(p.typed || 'Search products', input.x + 12, input.y + 26)
  if (p.focused) {
    const caret = input.x + 12 + ctx.measureText(p.typed).width + 1
    ctx.fillStyle = '#4f46e5'
    ctx.fillRect(caret, input.y + 10, 2, 20)
  }
  ctx.fillStyle = '#4f46e5'
  ctx.fillRect(button.x, button.y, button.w, button.h)
  ctx.fillStyle = '#ffffff'
  ctx.font = '600 15px sans-serif'
  ctx.fillText('Search', button.x + 30, button.y + 25)

  const listY = button.y + button.h + 28
  ctx.fillStyle = '#111827'
  ctx.font = '600 15px sans-serif'
  ctx.fillText(p.submitted ? `3 results for "${p.submitted}"` : 'Recent', PAD, listY)
  ;['Canvas tote bag', 'Cotton tote, large', 'Tote with zip'].forEach((name, i) => {
    const y = listY + 16 + i * 52
    if (y + 44 > h) return
    ctx.fillStyle = '#ffffff'
    ctx.fillRect(PAD, y, Math.min(w - PAD * 2, 560), 44)
    ctx.strokeStyle = '#e5e7eb'
    ctx.lineWidth = 1
    ctx.strokeRect(PAD, y, Math.min(w - PAD * 2, 560), 44)
    ctx.fillStyle = '#1f2937'
    ctx.font = '15px sans-serif'
    ctx.fillText(name, PAD + 14, y + 27)
  })
  if (p.loading) {
    ctx.fillStyle = '#4f46e5'
    ctx.fillRect(0, HEADER_H, w * 0.4, 3)
  }
  return canvas.toDataURL('image/jpeg', 0.7)
}

export function handleInput(p: PageModel, input: BrowserInput) {
  const { input: field, button } = layout(p.width, p.height)
  const inside = (r: Rect, x: number, y: number) => x >= r.x && x <= r.x + r.w && y >= r.y && y <= r.y + r.h
  if (input.type === 'mouse' && input.action === 'down') {
    p.focused = inside(field, input.x, input.y)
    if (inside(button, input.x, input.y)) p.submitted = p.typed || 'everything'
    p.dirty = true
  }
  if (input.type === 'key' && input.action === 'down' && p.focused) {
    if (input.key === 'Backspace') p.typed = p.typed.slice(0, -1)
    else if (input.key === 'Enter') p.submitted = p.typed || 'everything'
    else if (input.text) p.typed += input.text
    p.dirty = true
  }
  if (input.type === 'paste' && p.focused) {
    p.typed += input.text
    p.dirty = true
  }
}

export function pageRects(p: PageModel): { field: Rect; button: Rect } {
  const { input, button } = layout(p.width, p.height)
  return { field: input, button }
}
