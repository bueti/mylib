

export const index = 8;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/pages/login/_page.svelte.js')).default;
export const universal = {
  "prerender": false,
  "ssr": false
};
export const universal_id = "src/routes/login/+page.ts";
export const imports = ["_app/immutable/nodes/8.DQInkFuU.js","_app/immutable/chunks/Bq_VSsjf.js","_app/immutable/chunks/B6MMa1Tp.js","_app/immutable/chunks/t1H3BwMP.js","_app/immutable/chunks/Dmkd57ne.js","_app/immutable/chunks/54iCTxcV.js","_app/immutable/chunks/B7YJYiUB.js","_app/immutable/chunks/DOtZQTI5.js"];
export const stylesheets = ["_app/immutable/assets/8.BnsLoFvU.css"];
export const fonts = [];
