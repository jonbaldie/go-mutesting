export const title = 'Index and path-cache lifecycle';
export const state = v => `${v.seen.length ? 'Nodes collected' : 'No nodes collected'}; safety ${v.safetyBuilt ? 'built' : 'empty'}; ${Object.values(v.cached).some(paths => paths.length) ? 'paths cached' : 'no paths cached'}`;
export const step = action => ({Ensure:'Query range', BuildSafety:'Build safety index', ClearSafety:'Clear safety index', ClearIndexes:'Clear AST indexes', Resolve:'Resolve coverage path'})[action] || action;
