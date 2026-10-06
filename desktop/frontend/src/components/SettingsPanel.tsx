import type { ReactNode } from 'react'
import { useAgentos } from '../AgentosContext'
import type { CleanupMode, ThemeSetting } from '../types'
import { Keycap } from './Keycap'
import { Overlay } from './Overlay'
import { SegmentedControl } from './SegmentedControl'

const THEMES: { value: ThemeSetting; label: string }[] = [
  { value: 'system', label: 'System' },
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
]

const MODES: { value: CleanupMode; label: string }[] = [
  { value: 'auto', label: 'Auto' },
  { value: 'manual', label: 'Manual' },
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
  const { overlay, setOverlay, project, settings, setTheme, setCleanup, report } = useAgentos()
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
          <SegmentedControl label="Theme" options={THEMES} value={settings.theme} onChange={(value) => report(() => setTheme(value))} />
        </SettingRow>
        {project && (
          <>
          <h3 className="label mt-3 pb-0.5">{project.name}</h3>
          <SettingRow label="Clean up after a merge" hint="Auto removes the worktree and branch. Manual asks first. Unsaved or unpushed work always blocks.">
            <SegmentedControl label="Clean up after a merge" options={MODES} value={settings.cleanup.merge} onChange={(value) => report(() => setCleanup('merge', value))} />
          </SettingRow>
          <SettingRow label="Clean up after a close" hint="Auto cleans up when you close the pull request. Manual asks first.">
            <SegmentedControl label="Clean up after a close" options={MODES} value={settings.cleanup.close} onChange={(value) => report(() => setCleanup('close', value))} />
          </SettingRow>
          </>
        )}
      </div>
    </Overlay>
  )
}
