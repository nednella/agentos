import type { ITheme } from '@xterm/xterm'

const dark: ITheme = {
  background: '#0a0c10',
  foreground: '#dfe3ec',
  cursor: '#a78bfa',
  cursorAccent: '#0a0c10',
  selectionBackground: 'rgba(167, 139, 250, 0.35)',
  black: '#1a1f36',
  red: '#ff5c70',
  green: '#3de08f',
  yellow: '#ffc247',
  blue: '#5a8dff',
  magenta: '#d27cff',
  cyan: '#3fd2f0',
  white: '#cfd5ee',
  brightBlack: '#5b6488',
  brightRed: '#ff8794',
  brightGreen: '#7af0b4',
  brightYellow: '#ffd879',
  brightBlue: '#8fb0ff',
  brightMagenta: '#e5a8ff',
  brightCyan: '#8fe6f8',
  brightWhite: '#f4f6ff',
}

const light: ITheme = {
  background: '#fbfbfd',
  foreground: '#23272f',
  cursor: '#7c5ce0',
  cursorAccent: '#fbfbfd',
  selectionBackground: 'rgba(124, 92, 224, 0.25)',
  black: '#23272f',
  red: '#c8283c',
  green: '#1a8a55',
  yellow: '#9a6a00',
  blue: '#2a5fd0',
  magenta: '#9a3fd0',
  cyan: '#12829c',
  white: '#8b93a5',
  brightBlack: '#5b6488',
  brightRed: '#e0485a',
  brightGreen: '#2aa86c',
  brightYellow: '#b8820a',
  brightBlue: '#4a7de8',
  brightMagenta: '#b05ee8',
  brightCyan: '#2aa0ba',
  brightWhite: '#c4c9d6',
}

export const terminalThemes = { dark, light }

export const terminalFont = '"JetBrainsMono Nerd Font Mono", "JetBrains Mono", monospace'
