import { a as attr, e as escape_html } from "../../../../chunks/attributes.js";
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    $$renderer2.push(`<h2 class="svelte-1t8qm3z">Admin</h2> <div class="admin-actions svelte-1t8qm3z"><button${attr("disabled", false, true)} class="svelte-1t8qm3z">${escape_html("Rescan embedded metadata")}</button> <button${attr("disabled", false, true)} class="svelte-1t8qm3z">${escape_html("Enrich all from Open Library")}</button> <button${attr("disabled", false, true)} class="svelte-1t8qm3z">${escape_html("Clean up tags")}</button> `);
    {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--></div> <h3 class="svelte-1t8qm3z">Duplicate candidates</h3> <p class="note svelte-1t8qm3z">Books grouped by shared ISBN or matching title + author. Resolve by deleting the unwanted file on disk, then clicking Rescan on the home page.</p> `);
    {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<p>Loading…</p>`);
    }
    $$renderer2.push(`<!--]-->`);
  });
}
export {
  _page as default
};
