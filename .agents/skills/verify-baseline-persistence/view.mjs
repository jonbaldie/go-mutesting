export const title = 'Baseline persistence';
export const state = v => ({load: 'Load baseline', run: 'Finish mutation run', reload: 'Reload written baseline', done: 'Finished', failed: 'Report failure'})[v.phase];
export const step = action => ({Load: 'load baseline', Run: 'finish run', Reload: 'reload baseline', Terminal: 'done'})[action] || action;
