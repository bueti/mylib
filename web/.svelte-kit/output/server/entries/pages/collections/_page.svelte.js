import { a as attr } from "../../../chunks/attributes.js";
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let newName = "";
    $$renderer2.push(`<h2 class="svelte-8lyz9q">Collections</h2> <form class="new svelte-8lyz9q"><input type="text"${attr("value", newName)} placeholder="New collection name" maxlength="100" class="svelte-8lyz9q"/> <button type="submit"${attr("disabled", !newName.trim(), true)} class="svelte-8lyz9q">Create</button></form> `);
    {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--> `);
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
