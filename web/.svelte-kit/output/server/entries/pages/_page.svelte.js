import { a as attr_class, e as ensure_array_like, s as stringify, b as bind_props, c as attr_style, d as derived } from "../../chunks/root.js";
import { a as attr, e as escape_html } from "../../chunks/attributes.js";
import { p as page } from "../../chunks/index2.js";
import { g as goto } from "../../chunks/client.js";
function UploadDialog($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    const ACCEPTED = ".epub,.pdf,.mobi,.azw3,.azw";
    let { open = false, onDone } = $$props;
    let files = [];
    let uploading = false;
    let results = [];
    let dragOver = false;
    function formatSize(bytes) {
      if (bytes < 1024) return bytes + " B";
      const mb = bytes / (1024 * 1024);
      if (mb < 0.1) return (bytes / 1024).toFixed(1) + " KB";
      return mb.toFixed(1) + " MB";
    }
    if (open) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<div class="overlay svelte-n1zzka"><div class="dialog svelte-n1zzka"><header class="svelte-n1zzka"><h2 class="svelte-n1zzka">Upload books</h2> <button class="close svelte-n1zzka" aria-label="Close">×</button></header> <div${attr_class("dropzone svelte-n1zzka", void 0, { "dragover": dragOver })}><p class="svelte-n1zzka">Drag &amp; drop EPUB, PDF, MOBI, or AZW3 files here</p> <label class="pick svelte-n1zzka">or pick files <input type="file"${attr("accept", ACCEPTED)} multiple="" hidden=""/></label></div> `);
      if (files.length > 0) {
        $$renderer2.push("<!--[0-->");
        $$renderer2.push(`<ul class="file-list svelte-n1zzka"><!--[-->`);
        const each_array = ensure_array_like(files);
        for (let i = 0, $$length = each_array.length; i < $$length; i++) {
          let f = each_array[i];
          $$renderer2.push(`<li class="svelte-n1zzka"><span class="fname svelte-n1zzka">${escape_html(f.name)}</span> <span class="fsize svelte-n1zzka">${escape_html(formatSize(f.size))}</span> <button class="remove svelte-n1zzka" aria-label="Remove">×</button></li>`);
        }
        $$renderer2.push(`<!--]--></ul> <button class="upload-btn svelte-n1zzka"${attr("disabled", uploading, true)}>${escape_html(`Upload ${files.length} file${files.length > 1 ? "s" : ""}`)}</button> `);
        {
          $$renderer2.push("<!--[-1-->");
        }
        $$renderer2.push(`<!--]-->`);
      } else {
        $$renderer2.push("<!--[-1-->");
      }
      $$renderer2.push(`<!--]--> `);
      if (results.length > 0) {
        $$renderer2.push("<!--[0-->");
        $$renderer2.push(`<ul class="results svelte-n1zzka"><!--[-->`);
        const each_array_1 = ensure_array_like(results);
        for (let $$index_1 = 0, $$length = each_array_1.length; $$index_1 < $$length; $$index_1++) {
          let r = each_array_1[$$index_1];
          $$renderer2.push(`<li${attr_class("svelte-n1zzka", void 0, { "success": !!r.book, "fail": !!r.error })}>`);
          if (r.book) {
            $$renderer2.push("<!--[0-->");
            $$renderer2.push(`<a${attr("href", `/books/${stringify(r.book.id)}`)} class="svelte-n1zzka">${escape_html(r.book.title)}</a> — added`);
          } else {
            $$renderer2.push("<!--[-1-->");
            $$renderer2.push(`${escape_html(r.file)}: ${escape_html(r.error)}`);
          }
          $$renderer2.push(`<!--]--></li>`);
        }
        $$renderer2.push(`<!--]--></ul>`);
      } else {
        $$renderer2.push("<!--[-1-->");
      }
      $$renderer2.push(`<!--]--></div></div>`);
    } else {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]-->`);
    bind_props($$props, { open });
  });
}
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let query = "";
    let activeAuthor = null;
    let activeSeries = null;
    let activeTags = [];
    let activeFormat = null;
    let activeSort = "-added";
    let books = [];
    let total = 0;
    let loading = false;
    let loadingMore = false;
    let error = null;
    const PAGE_SIZE = 60;
    let recent = [];
    let allTags = [];
    let sidebarOpen = true;
    let sidebarTags = derived(() => allTags.filter((t) => t.count >= 2));
    let scanning = false;
    let uploadOpen = false;
    async function loadTags() {
      try {
        const res = await fetch("/api/tags", { credentials: "same-origin" });
        if (!res.ok) return;
        const data = await res.json();
        allTags = data?.tags ?? [];
      } catch {
      }
    }
    function buildParams(f, offset) {
      const params = new URLSearchParams();
      if (f.tags.length > 0) params.set("tag", f.tags.join(","));
      params.set("sort", f.sort);
      params.set("limit", String(PAGE_SIZE));
      params.set("offset", String(offset));
      return params;
    }
    async function load(f) {
      loading = true;
      error = null;
      try {
        const params = buildParams(f, 0);
        const res = await fetch("/api/books?" + params.toString(), { credentials: "same-origin" });
        if (!res.ok) throw new Error("HTTP " + res.status);
        const data = await res.json();
        books = data?.books ?? [];
        total = data?.total ?? 0;
      } catch (e) {
        error = e instanceof Error ? e.message : "Failed to load";
        books = [];
        total = 0;
      } finally {
        loading = false;
      }
    }
    function setSort(s) {
      const sp = new URLSearchParams(page.url.searchParams);
      sp.set("sort", s);
      goto("/?" + sp.toString(), {});
    }
    let hasFilters = derived(() => activeTags.length > 0 || activeAuthor || activeSeries || activeFormat || query);
    let $$settled = true;
    let $$inner_renderer;
    function $$render_inner($$renderer3) {
      $$renderer3.push(`<div${attr_class("layout svelte-1uha8ag", void 0, { "sidebar-visible": sidebarTags().length > 0 })}>`);
      if (sidebarTags().length > 0) {
        $$renderer3.push("<!--[0-->");
        $$renderer3.push(`<aside${attr_class("sidebar svelte-1uha8ag", void 0, { "open": sidebarOpen })}><header class="svelte-1uha8ag"><span>Genres &amp; Topics</span> <button aria-label="Close sidebar" class="svelte-1uha8ag">×</button></header> <ul class="svelte-1uha8ag"><!--[-->`);
        const each_array = ensure_array_like(sidebarTags());
        for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
          let tag = each_array[$$index];
          $$renderer3.push(`<li><button${attr_class("svelte-1uha8ag", void 0, { "active": activeTags.includes(tag.name) })}><span class="tag-name">${escape_html(tag.name)}</span> <span class="tag-count svelte-1uha8ag">${escape_html(tag.count)}</span></button></li>`);
        }
        $$renderer3.push(`<!--]--></ul></aside>`);
      } else {
        $$renderer3.push("<!--[-1-->");
      }
      $$renderer3.push(`<!--]--> <div class="main svelte-1uha8ag"><div class="toolbar svelte-1uha8ag">`);
      if (allTags.length > 0 && !sidebarOpen) {
        $$renderer3.push("<!--[0-->");
        $$renderer3.push(`<button class="sidebar-toggle svelte-1uha8ag" title="Show genres">☰</button>`);
      } else {
        $$renderer3.push("<!--[-1-->");
      }
      $$renderer3.push(`<!--]--> <input type="search" placeholder="Search title, author, series, genre…"${attr("value", query)} aria-label="Search" class="svelte-1uha8ag"/> `);
      $$renderer3.select(
        {
          value: activeSort,
          onchange: (e) => setSort(e.target.value),
          class: ""
        },
        ($$renderer4) => {
          $$renderer4.option({ value: "-added" }, ($$renderer5) => {
            $$renderer5.push(`Recently added`);
          });
          $$renderer4.option({ value: "title" }, ($$renderer5) => {
            $$renderer5.push(`Title A–Z`);
          });
          $$renderer4.option({ value: "-title" }, ($$renderer5) => {
            $$renderer5.push(`Title Z–A`);
          });
          $$renderer4.option({ value: "added" }, ($$renderer5) => {
            $$renderer5.push(`Oldest first`);
          });
        },
        "svelte-1uha8ag"
      );
      $$renderer3.push(` <button class="upload svelte-1uha8ag">Upload</button> <button${attr("disabled", scanning, true)} class="rescan svelte-1uha8ag">${escape_html("Rescan")}</button> <span class="count svelte-1uha8ag">${escape_html(total)} ${escape_html(total === 1 ? "book" : "books")}</span></div> `);
      {
        $$renderer3.push("<!--[-1-->");
      }
      $$renderer3.push(`<!--]--> `);
      if (activeTags.length > 0 || activeFormat) {
        $$renderer3.push("<!--[0-->");
        $$renderer3.push(`<div class="chips svelte-1uha8ag"><span class="chips-label svelte-1uha8ag">Filters:</span> `);
        {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]--> `);
        {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]--> <!--[-->`);
        const each_array_1 = ensure_array_like(activeTags);
        for (let $$index_1 = 0, $$length = each_array_1.length; $$index_1 < $$length; $$index_1++) {
          let tag = each_array_1[$$index_1];
          $$renderer3.push(`<button class="chip svelte-1uha8ag">${escape_html(tag)} ✕</button>`);
        }
        $$renderer3.push(`<!--]--> `);
        {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]--></div>`);
      } else {
        $$renderer3.push("<!--[-1-->");
      }
      $$renderer3.push(`<!--]--> `);
      if (error) {
        $$renderer3.push("<!--[0-->");
        $$renderer3.push(`<p class="error svelte-1uha8ag">Error: ${escape_html(error)}</p>`);
      } else {
        $$renderer3.push("<!--[-1-->");
      }
      $$renderer3.push(`<!--]--> `);
      if (recent.length > 0) {
        $$renderer3.push("<!--[0-->");
        $$renderer3.push(`<section class="continue svelte-1uha8ag"><h2 class="svelte-1uha8ag">Continue reading</h2> <ul class="row svelte-1uha8ag"><!--[-->`);
        const each_array_2 = ensure_array_like(recent);
        for (let $$index_2 = 0, $$length = each_array_2.length; $$index_2 < $$length; $$index_2++) {
          let entry = each_array_2[$$index_2];
          $$renderer3.push(`<li><a${attr("href", `/books/${stringify(entry.book.id)}/read`)} class="recent-card svelte-1uha8ag">`);
          if (entry.book.has_cover) {
            $$renderer3.push("<!--[0-->");
            $$renderer3.push(`<img${attr("src", `/api/books/${stringify(entry.book.id)}/cover`)} alt="" loading="lazy" class="svelte-1uha8ag"/>`);
          } else {
            $$renderer3.push("<!--[-1-->");
            $$renderer3.push(`<div class="placeholder svelte-1uha8ag">${escape_html(entry.book.title.charAt(0))}</div>`);
          }
          $$renderer3.push(`<!--]--> <div class="progress-bar svelte-1uha8ag"><div class="progress-fill svelte-1uha8ag"${attr_style("", {
            width: `${stringify(Math.max(2, Math.round(entry.progress.percent * 100)))}%`
          })}></div></div> <div class="recent-title svelte-1uha8ag"${attr("title", entry.book.title)}>${escape_html(entry.book.title)}</div></a></li>`);
        }
        $$renderer3.push(`<!--]--></ul></section>`);
      } else {
        $$renderer3.push("<!--[-1-->");
      }
      $$renderer3.push(`<!--]--> `);
      if (!hasFilters() && sidebarTags().length > 0) {
        $$renderer3.push("<!--[0-->");
        $$renderer3.push(`<section class="browse svelte-1uha8ag"><h2 class="svelte-1uha8ag">Browse by genre</h2> <div class="genre-grid svelte-1uha8ag"><!--[-->`);
        const each_array_3 = ensure_array_like(sidebarTags().slice(0, 20));
        for (let $$index_3 = 0, $$length = each_array_3.length; $$index_3 < $$length; $$index_3++) {
          let tag = each_array_3[$$index_3];
          $$renderer3.push(`<button class="genre-card svelte-1uha8ag"><span class="genre-name svelte-1uha8ag">${escape_html(tag.name)}</span> <span class="genre-count svelte-1uha8ag">${escape_html(tag.count)} ${escape_html(tag.count === 1 ? "book" : "books")}</span></button>`);
        }
        $$renderer3.push(`<!--]--></div></section>`);
      } else {
        $$renderer3.push("<!--[-1-->");
      }
      $$renderer3.push(`<!--]--> `);
      if (loading && books.length === 0) {
        $$renderer3.push("<!--[0-->");
        $$renderer3.push(`<p>Loading…</p>`);
      } else if (books.length === 0) {
        $$renderer3.push("<!--[1-->");
        $$renderer3.push(`<p class="empty svelte-1uha8ag">No books match. Point MYLIB_LIBRARY_ROOTS at a directory of EPUBs or PDFs and hit <button class="link svelte-1uha8ag">Rescan</button>.</p>`);
      } else {
        $$renderer3.push("<!--[-1-->");
        $$renderer3.push(`<ul class="grid svelte-1uha8ag"><!--[-->`);
        const each_array_4 = ensure_array_like(books);
        for (let $$index_4 = 0, $$length = each_array_4.length; $$index_4 < $$length; $$index_4++) {
          let book = each_array_4[$$index_4];
          $$renderer3.push(`<li class="card svelte-1uha8ag"><a${attr("href", `/books/${stringify(book.id)}`)} class="cover svelte-1uha8ag">`);
          if (book.has_cover) {
            $$renderer3.push("<!--[0-->");
            $$renderer3.push(`<img${attr("src", `/api/books/${stringify(book.id)}/cover`)} alt="" loading="lazy" class="svelte-1uha8ag"/>`);
          } else {
            $$renderer3.push("<!--[-1-->");
            $$renderer3.push(`<div class="placeholder svelte-1uha8ag">${escape_html(book.title.charAt(0))}</div>`);
          }
          $$renderer3.push(`<!--]--> <span class="format-badge svelte-1uha8ag">${escape_html(book.format.toUpperCase())}</span></a> <div class="meta svelte-1uha8ag"><a${attr("href", `/books/${stringify(book.id)}`)} class="title svelte-1uha8ag"${attr("title", book.title)}>${escape_html(book.title)}</a> <div class="authors svelte-1uha8ag">${escape_html((book.authors ?? []).map((a) => a.name).join(", ") || "—")}</div></div></li>`);
        }
        $$renderer3.push(`<!--]--></ul> `);
        if (books.length < total) {
          $$renderer3.push("<!--[0-->");
          $$renderer3.push(`<div class="load-more svelte-1uha8ag"><button${attr("disabled", loadingMore, true)} class="svelte-1uha8ag">${escape_html(`Load more (${books.length} of ${total})`)}</button></div>`);
        } else {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]-->`);
      }
      $$renderer3.push(`<!--]--></div></div> `);
      UploadDialog($$renderer3, {
        onDone: () => {
          load({
            q: query,
            author: activeAuthor?.id,
            series: activeSeries?.id,
            tags: activeTags,
            format: activeFormat,
            sort: activeSort
          });
          void loadTags();
        },
        get open() {
          return uploadOpen;
        },
        set open($$value) {
          uploadOpen = $$value;
          $$settled = false;
        }
      });
      $$renderer3.push(`<!---->`);
    }
    do {
      $$settled = true;
      $$inner_renderer = $$renderer2.copy();
      $$render_inner($$inner_renderer);
    } while (!$$settled);
    $$renderer2.subsume($$inner_renderer);
  });
}
export {
  _page as default
};
