export const title = 'Changed lines by file';
export const state = v => v.phase === 'finished' ? 'Return changed lines' : v.selected === 'none' ? 'Await file' : 'Read selected file';
export const step = action => ({Select: 'select file', Hunk: 'read hunk', Finish: 'return coalesced lines', Terminal: 'done'})[action] || action;
