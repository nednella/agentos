import type { ReactNode } from 'react'
import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import type { CleanupMode, DigestSchedule, ThemeSetting } from '../types'
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

const SWITCH: { value: 'on' | 'off'; label: string }[] = [
  { value: 'on', label: 'On' },
  { value: 'off', label: 'Off' },
]

const SCHEDULES: { value: DigestSchedule; label: string }[] = [
  { value: 'weekly', label: 'Weekly' },
  { value: 'off', label: 'Off' },
]

type SettingRowProps = { label: string; hint: string; children: ReactNode }

function SettingRow({ label, hint, children }: SettingRowProps) {
  return (
    <div className="flex items-center gap-6 border-b border-line py-3.5 last:border-b-0">
      <div className="min-w-0 flex-1">
        <p className="text-body font-medium">{label}</p>
        <p className="mt-0.5 text-small text-dim">{hint}</p>
      </div>
      {children}
    </div>
  )
}

export function SettingsPanel() {
  const { overlay, setOverlay, project, settings, setTheme, setKeepAwake, setCleanup, setBrowserEnabled, setDigestSchedule, report } = useAgentos()
  const { scale, zoom } = useLayout()
  if (overlay !== 'settings') return null

  return (
    <Overlay align="center" size="reading" label="Settings" onClose={() => setOverlay(null)}>
      <div className="flex flex-none items-center gap-3 border-b border-line px-5 py-3.5">
        <h2 className="flex-1 text-title font-semibold">Settings</h2>
        <Keycap>Esc</Keycap>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-4">
        <h3 className="label pb-1">App</h3>
        <SettingRow label="Theme" hint="System follows the macOS appearance.">
          <SegmentedControl label="Theme" options={THEMES} value={settings.theme} onChange={(value) => report(() => setTheme(value))} />
        </SettingRow>
        <SettingRow label="Text size" hint="Scales the interface and the terminal.">
          <div role="group" aria-label="Text size" className="flex flex-none items-center gap-1">
            <button className="btn btn-ghost h-7 w-7 justify-center px-0" aria-label="Smaller text" onClick={() => zoom(-1)}>
              −
            </button>
            <button className="btn btn-ghost mono h-7 w-14 justify-center px-0" title="Reset text size" aria-label="Reset text size" onClick={() => zoom(0)}>
              {Math.round(scale * 100)}%
            </button>
            <button className="btn btn-ghost h-7 w-7 justify-center px-0" aria-label="Larger text" onClick={() => zoom(1)}>
              +
            </button>
          </div>
        </SettingRow>
        <SettingRow label="Keep Mac awake" hint="Stops the Mac idle-sleeping while a session works. A project that sets its own in the config file keeps it.">
          <SegmentedControl label="Keep Mac awake" options={SWITCH} value={settings.keepAwake ? 'on' : 'off'} onChange={(value) => report(() => setKeepAwake(value === 'on'))} />
        </SettingRow>
        {project && (
          <>
          <h3 className="label mt-6 pb-1">{project.name}</h3>
          <SettingRow label="Clean up after a merge" hint="Auto removes the worktree and branch. Manual asks first. Unsaved or unpushed work always blocks.">
            <SegmentedControl label="Clean up after a merge" options={MODES} value={settings.cleanup.merge} onChange={(value) => report(() => setCleanup('merge', value))} />
          </SettingRow>
          <SettingRow label="Clean up after a close" hint="Auto cleans up when you close the pull request. Manual asks first.">
            <SegmentedControl label="Clean up after a close" options={MODES} value={settings.cleanup.close} onChange={(value) => report(() => setCleanup('close', value))} />
          </SettingRow>
          <SettingRow label="Browser enabled" hint="Tells new sessions about the browser and evidence commands.">
            <SegmentedControl label="Browser enabled" options={SWITCH} value={settings.browserEnabled ? 'on' : 'off'} onChange={(value) => report(() => setBrowserEnabled(value === 'on'))} />
          </SettingRow>
          <SettingRow label="Digest schedule" hint="Weekly runs the digest by itself. Off leaves it to you.">
            <SegmentedControl label="Digest schedule" options={SCHEDULES} value={settings.digestSchedule} onChange={(value) => report(() => setDigestSchedule(value))} />
          </SettingRow>
          </>
        )}
      </div>
    </Overlay>
  )
}
