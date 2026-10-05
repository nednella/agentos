import type { Action, ActionContext } from './types'

export const noteActions = ({ a, layout }: ActionContext): Action[] => [
  {
    id: 'note',
    label: 'New note',
    group: 'Notes',
    palette: { prompt: 'Note text' },
    async run(args) {
      if (args.length === 0) {
        layout.showSidebarTab('notes')
        a.focus('note-input')
        return
      }
      await a.addNote(args.join(' '))
      return 'Note added'
    },
  },
  {
    id: 'show-notes',
    label: 'Show notes',
    group: 'Notes',
    uiCommand: 'notes',
    run() {
      layout.showSidebarTab('notes')
      a.focus('note-input')
    },
  },
]
