export const title = 'Unified diff hunk';
export const state = v => ({header: 'Await header', body: 'Read hunk', success: 'Accept change', reject: 'Reject hunk'})[v.phase];
export const step = action => ({Header: 'read header', Body: 'read body line', Finish: 'complete hunk', EndOfInput: 'reach end of input', Terminal: 'done'})[action] || action;
