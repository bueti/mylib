

export const index = 3;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/pages/admin/duplicates/_page.svelte.js')).default;
export const universal = {
  "prerender": false,
  "ssr": false
};
export const universal_id = "src/routes/admin/duplicates/+page.ts";
export const imports = ["_app/immutable/nodes/3.BTMLf3mv.js","_app/immutable/chunks/Bq_VSsjf.js","_app/immutable/chunks/B6MMa1Tp.js","_app/immutable/chunks/t1H3BwMP.js","_app/immutable/chunks/I4q_SWcC.js","_app/immutable/chunks/Dmkd57ne.js"];
export const stylesheets = ["_app/immutable/assets/3.BTa5KcCm.css"];
export const fonts = [];
