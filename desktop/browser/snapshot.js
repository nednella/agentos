(() => {
  const MAX_LINES = 400
  const MAX_TEXT = 200
  document.querySelectorAll('[data-agentos-ref]').forEach((e) => e.removeAttribute('data-agentos-ref'))

  const SKIP = new Set(['SCRIPT', 'STYLE', 'NOSCRIPT', 'TEMPLATE', 'HEAD', 'SVG', 'CANVAS'])
  const LANDMARKS = { NAV: 'nav', MAIN: 'main', HEADER: 'header', FOOTER: 'footer', ASIDE: 'aside', FORM: 'form', DIALOG: 'dialog' }
  const INTERACTIVE = 'a[href], button, input:not([type=hidden]), select, textarea, summary, [role=button], [role=link], [role=checkbox], [role=radio], [role=switch], [role=tab], [role=menuitem], [role=combobox], [role=textbox], [onclick], [contenteditable=""], [contenteditable="true"]'

  const lines = []
  let refs = 0
  let cut = false
  let buffer = ''

  const clean = (s) => (s || '').replace(/\s+/g, ' ').trim()
  const short = (s, n) => (s.length > n ? s.slice(0, n - 1) + '…' : s)
  const push = (line) => {
    if (lines.length >= MAX_LINES) { cut = true; return }
    lines.push(line)
  }
  const flush = () => {
    const text = clean(buffer)
    buffer = ''
    if (text) push(short(text, MAX_TEXT))
  }

  const visible = (el) => {
    const s = getComputedStyle(el)
    if (s.display === 'none' || s.visibility === 'hidden') return false
    const r = el.getBoundingClientRect()
    return r.width > 0 && r.height > 0
  }

  const nameOf = (el) => {
    const label = el.getAttribute('aria-label')
    if (label) return clean(label)
    const by = el.getAttribute('aria-labelledby')
    if (by) {
      const text = by.split(/\s+/).map((id) => { const t = document.getElementById(id); return t ? t.textContent : '' }).join(' ')
      if (clean(text)) return clean(text)
    }
    if (el.labels && el.labels.length) {
      const text = Array.from(el.labels).map((l) => l.textContent).join(' ')
      if (clean(text)) return clean(text)
    }
    const tag = el.tagName
    if (tag === 'INPUT' && ['submit', 'button', 'reset'].includes(el.type)) return clean(el.value) || el.type
    if (tag === 'INPUT' && el.type === 'image') return clean(el.alt)
    const inner = clean(el.innerText || el.textContent)
    if (inner && !['INPUT', 'SELECT', 'TEXTAREA'].includes(tag)) return short(inner, 80)
    return clean(el.title) || clean(el.getAttribute('placeholder')) || clean(el.getAttribute('name'))
  }

  const roleOf = (el) => {
    const role = el.getAttribute('role')
    if (role) return role
    switch (el.tagName) {
      case 'A': return 'link'
      case 'BUTTON': case 'SUMMARY': return 'button'
      case 'SELECT': return 'select'
      case 'TEXTAREA': return 'textbox'
      case 'INPUT':
        if (['checkbox', 'radio'].includes(el.type)) return el.type
        if (['submit', 'button', 'reset', 'image'].includes(el.type)) return 'button'
        if (el.type === 'range') return 'slider'
        if (el.type === 'file') return 'file'
        return 'textbox'
    }
    return el.isContentEditable ? 'textbox' : 'clickable'
  }

  const describe = (el) => {
    refs += 1
    const ref = 'e' + refs
    el.setAttribute('data-agentos-ref', ref)
    const role = roleOf(el)
    const name = nameOf(el)
    const parts = ['[' + ref + '] ' + role]
    if (name) parts.push('"' + short(name, 80) + '"')
    const tag = el.tagName
    if (tag === 'INPUT' && !['checkbox', 'radio', 'submit', 'button', 'reset', 'image'].includes(el.type)) {
      if (el.type && el.type !== 'text') parts.push('type=' + el.type)
      if (el.value) parts.push('value="' + short(el.type === 'password' ? '•'.repeat(el.value.length) : el.value, 60) + '"')
      else if (el.placeholder && !name.includes(el.placeholder)) parts.push('placeholder="' + short(el.placeholder, 40) + '"')
    } else if (tag === 'TEXTAREA') {
      if (el.value) parts.push('value="' + short(clean(el.value), 60) + '"')
    } else if (tag === 'SELECT') {
      const chosen = el.selectedOptions && el.selectedOptions[0]
      if (chosen) parts.push('value="' + short(clean(chosen.text), 40) + '"')
      parts.push('options=' + short(Array.from(el.options).slice(0, 10).map((o) => clean(o.text)).join(' | '), 120))
    } else if (el.isContentEditable && tag !== 'INPUT') {
      const text = clean(el.innerText)
      if (text) parts.push('value="' + short(text, 60) + '"')
    }
    if (el.checked === true || el.getAttribute('aria-checked') === 'true') parts.push('checked')
    if (el.getAttribute('aria-expanded')) parts.push('expanded=' + el.getAttribute('aria-expanded'))
    if (el.disabled || el.getAttribute('aria-disabled') === 'true') parts.push('disabled')
    if (el.required) parts.push('required')
    if (tag === 'A') {
      const href = el.getAttribute('href') || ''
      parts.push('-> ' + short(href, 80))
    }
    return parts.join(' ')
  }

  const walk = (node, depth, inLabel) => {
    if (cut) return
    if (node.nodeType === Node.TEXT_NODE) { if (!inLabel) buffer += ' ' + node.textContent; return }
    if (node.nodeType !== Node.ELEMENT_NODE) return
    const el = node
    if (SKIP.has(el.tagName.toUpperCase()) || el.hidden) return
    const style = getComputedStyle(el)
    if (style.display === 'none') return

    if (el.matches(INTERACTIVE)) {
      if (!visible(el)) return
      flush()
      push('  '.repeat(depth) + describe(el))
      return
    }
    if (el.tagName === 'IMG') {
      const alt = clean(el.alt)
      if (alt) { flush(); push('  '.repeat(depth) + '(image: ' + short(alt, 80) + ')') }
      return
    }
    const heading = /^H([1-6])$/.exec(el.tagName)
    if (heading) {
      flush()
      const text = clean(el.innerText)
      if (text) push('  '.repeat(depth) + '#'.repeat(Number(heading[1])) + ' ' + short(text, MAX_TEXT))
      return
    }
    // A label's words are already the name of its control.
    const labelling = el.tagName === 'LABEL' && !!el.control
    const block = !style.display.startsWith('inline')
    const landmark = LANDMARKS[el.tagName] || (el.getAttribute('role') && ['navigation', 'main', 'banner', 'contentinfo', 'complementary', 'dialog', 'search'].includes(el.getAttribute('role')) ? el.getAttribute('role') : '')
    let inner = depth
    if (landmark) {
      flush()
      const label = clean(el.getAttribute('aria-label'))
      push('  '.repeat(depth) + '<' + landmark + (label ? ' "' + short(label, 40) + '"' : '') + '>')
      inner = depth + 1
    } else if (block) {
      flush()
    }
    const kids = el.shadowRoot ? el.shadowRoot.childNodes : el.childNodes
    for (const kid of kids) walk(kid, inner, inLabel || labelling)
    if (landmark || block) flush()
  }

  walk(document.body || document.documentElement, 0, false)
  flush()
  const header = document.title + ' - ' + location.href
  const out = [header, ...lines]
  if (cut) out.push('... cut at ' + MAX_LINES + ' lines: use "text <ref>" or scroll and run snapshot again for the rest')
  if (refs === 0) out.push('(no interactive elements)')
  return out.join('\n')
})()
