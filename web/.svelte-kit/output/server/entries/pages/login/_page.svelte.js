import { a as attr, e as escape_html } from "../../../chunks/attributes.js";
import "@sveltejs/kit/internal";
import "../../../chunks/exports.js";
import "../../../chunks/utils.js";
import "@sveltejs/kit/internal/server";
import "../../../chunks/root.js";
import "../../../chunks/state.svelte.js";
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let username = "";
    let password = "";
    let submitting = false;
    $$renderer2.push(`<div class="wrap svelte-1x05zx6"><h2 class="svelte-1x05zx6">Sign in</h2> <form class="svelte-1x05zx6"><label class="svelte-1x05zx6">Username <input type="text"${attr("value", username)} autocomplete="username" required="" autofocus="" class="svelte-1x05zx6"/></label> <label class="svelte-1x05zx6">Password <input type="password"${attr("value", password)} autocomplete="current-password" required="" class="svelte-1x05zx6"/></label> `);
    {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--> <button type="submit"${attr("disabled", submitting, true)} class="svelte-1x05zx6">${escape_html("Sign in")}</button></form></div>`);
  });
}
export {
  _page as default
};
