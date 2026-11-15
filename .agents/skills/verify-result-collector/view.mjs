export const title = 'Collecting and reporting completed mutants';
export const state = v => {
 const recorded = Object.values(v.recorded).reduce((n, values) => n + values.length, 0);
 return `${recorded} recorded; ${v.sent.length} events sent; ${v.summarySent ? 'summary sent' : v.reportReady ? 'snapshot ready' : 'no snapshot'}`;
};
export const step = action => action.startsWith('Add') ? 'record a result' : action.startsWith('SendSummary') ? 'publish summary' : action.startsWith('Send') ? 'publish an event' : action.startsWith('Fill') ? 'capture report' : action;
