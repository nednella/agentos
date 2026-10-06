import type { ReactNode } from 'react'
import { useAgentos } from '../AgentosContext'
import type { ThemeSetting } from '../types'
import { Keycap } from './Keycap'
import { Overlay } from './Overlay'

const THEMES: { value: ThemeSetting; label: string }[] = [
  { value: 'system', label: 'System' },
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
]

type SettingRowProps = { label: string; hint: string; children: ReactNode }

function SettingRow({ label, hint, children }: SettingRowProps) {
  return (
    <div className="flex items-center gap-4 border-b border-line py-3 last:border-b-0">
      <div className="min-w-0 flex-1">
        <p className="text-body font-medium">{label}</p>
        <p className="text-small text-dim">{hint}</p>
      </div>
      {children}
    </div>
  )
}

export function SettingsPanel() {
  const { overlay, setOverlay, settings, setTheme, report } = useAgentos()
  if (overlay !== 'settings') return null

  return (
    <Overlay align="center" label="Settings" onClose={() => setOverlay(null)}>
      <div className="flex flex-none items-center gap-3 border-b border-line px-4 py-2.5">
        <h2 className="flex-1 text-title font-semibold">Settings</h2>
        <Keycap>Esc</Keycap>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto px-5 py-3">
        <h3 className="label pb-0.5">App</h3>
        <SettingRow label="Theme" hint="System follows the macOS appearance.">
          <div role="group" aria-label="Theme" className="flex flex-none overflow-hidden rounded-md border border-line-strong">
            {THEMES.map(({ value, label }) => (
              <button
                key={value}
                aria-pressed={settings.theme === value}
                className="h-7 px-3 text-small font-medium"
                style={{ background: settings.theme === value ? 'var(--bg-hover)' : 'transparent', color: settings.theme === value ? 'var(--text)' : 'var(--text-soft)' }}
                onClick={() => report(() => setTheme(value))}
              >
                {label}
              </button>
            ))}
          </div>
        </SettingRow>
      </div>
    </Overlay>
  )
}
