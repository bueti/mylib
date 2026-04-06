<script lang="ts">
	const ACCEPTED = '.epub,.pdf,.mobi,.azw3,.azw';

	let { open = $bindable(false), onDone }: { open: boolean; onDone?: () => void } = $props();

	interface UploadResult {
		file: string;
		error?: string;
		book?: { id: number; title: string };
	}

	let files = $state<File[]>([]);
	let uploading = $state(false);
	let progress = $state(0);
	let results = $state<UploadResult[]>([]);
	let dragOver = $state(false);

	function onDrop(e: DragEvent) {
		e.preventDefault();
		dragOver = false;
		if (!e.dataTransfer?.files) return;
		addFiles(e.dataTransfer.files);
	}

	function onPick(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files) addFiles(input.files);
		input.value = '';
	}

	function addFiles(fl: FileList) {
		for (const f of fl) {
			const ext = '.' + f.name.split('.').pop()?.toLowerCase();
			if (ACCEPTED.includes(ext)) {
				files = [...files, f];
			}
		}
	}

	function removeFile(index: number) {
		files = files.filter((_, i) => i !== index);
	}

	function formatSize(bytes: number): string {
		if (bytes < 1024) return bytes + ' B';
		const mb = bytes / (1024 * 1024);
		if (mb < 0.1) return (bytes / 1024).toFixed(1) + ' KB';
		return mb.toFixed(1) + ' MB';
	}

	async function upload() {
		if (files.length === 0 || uploading) return;
		uploading = true;
		progress = 0;
		results = [];

		const form = new FormData();
		for (const f of files) form.append('files', f);

		try {
			const data: { results: UploadResult[] } = await new Promise((resolve, reject) => {
				const xhr = new XMLHttpRequest();
				xhr.open('POST', '/api/books/upload');
				xhr.withCredentials = true;
				xhr.upload.onprogress = (e) => {
					if (e.lengthComputable) progress = Math.round((e.loaded / e.total) * 100);
				};
				xhr.onload = () => {
					if (xhr.status >= 200 && xhr.status < 300) {
						resolve(JSON.parse(xhr.responseText));
					} else {
						reject(new Error(xhr.responseText || 'Upload failed'));
					}
				};
				xhr.onerror = () => reject(new Error('Network error'));
				xhr.send(form);
			});
			results = data.results;
			files = [];
			onDone?.();
		} catch (e) {
			results = [{ file: '', error: e instanceof Error ? e.message : 'Upload failed' }];
		} finally {
			uploading = false;
		}
	}

	function close() {
		if (uploading) return;
		open = false;
		files = [];
		results = [];
		progress = 0;
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="overlay" onclick={close} onkeydown={(e) => e.key === 'Escape' && close()}>
		<!-- svelte-ignore a11y_no_static_element_interactions a11y_click_events_have_key_events -->
		<div class="dialog" onclick={(e) => e.stopPropagation()}>
			<header>
				<h2>Upload books</h2>
				<button class="close" onclick={close} aria-label="Close">×</button>
			</header>

			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div
				class="dropzone"
				class:dragover={dragOver}
				ondragover={(e) => { e.preventDefault(); dragOver = true; }}
				ondragleave={() => (dragOver = false)}
				ondrop={onDrop}
			>
				<p>Drag & drop EPUB, PDF, MOBI, or AZW3 files here</p>
				<label class="pick">
					or pick files
					<input type="file" accept={ACCEPTED} multiple onchange={onPick} hidden />
				</label>
			</div>

			{#if files.length > 0}
				<ul class="file-list">
					{#each files as f, i (f.name + i)}
						<li>
							<span class="fname">{f.name}</span>
							<span class="fsize">{formatSize(f.size)}</span>
							<button class="remove" onclick={() => removeFile(i)} aria-label="Remove">×</button>
						</li>
					{/each}
				</ul>
				<button class="upload-btn" onclick={upload} disabled={uploading}>
					{uploading ? `Uploading… ${progress}%` : `Upload ${files.length} file${files.length > 1 ? 's' : ''}`}
				</button>
				{#if uploading}
					<div class="progress-bar">
						<div class="progress-fill" style:width="{progress}%"></div>
					</div>
				{/if}
			{/if}

			{#if results.length > 0}
				<ul class="results">
					{#each results as r}
						<li class:success={!!r.book} class:fail={!!r.error}>
							{#if r.book}
								<a href="/books/{r.book.id}">{r.book.title}</a> — added
							{:else}
								{r.file}: {r.error}
							{/if}
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</div>
{/if}

<style>
	.overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.4);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 100;
	}
	.dialog {
		background: #fff;
		border-radius: 8px;
		width: 90%;
		max-width: 520px;
		max-height: 90vh;
		overflow-y: auto;
		box-shadow: 0 8px 30px rgba(0, 0, 0, 0.2);
	}
	header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1rem 1.25rem;
		border-bottom: 1px solid #eee;
	}
	h2 {
		margin: 0;
		font-size: 1.125rem;
	}
	.close {
		background: none;
		border: 0;
		font-size: 1.5rem;
		cursor: pointer;
		color: #666;
	}
	.dropzone {
		margin: 1.25rem;
		padding: 2rem;
		border: 2px dashed #ccc;
		border-radius: 8px;
		text-align: center;
		color: #666;
		transition: border-color 0.15s, background 0.15s;
	}
	.dropzone.dragover {
		border-color: #0366d6;
		background: #f0f7ff;
	}
	.dropzone p {
		margin: 0 0 0.75rem;
	}
	.pick {
		color: #0366d6;
		cursor: pointer;
		text-decoration: underline;
	}
	.file-list {
		list-style: none;
		padding: 0;
		margin: 0 1.25rem;
	}
	.file-list li {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.5rem 0;
		border-bottom: 1px solid #f0f0f0;
		font-size: 0.875rem;
	}
	.fname {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.fsize {
		color: #888;
		font-size: 0.75rem;
	}
	.remove {
		background: none;
		border: 0;
		color: #b00020;
		cursor: pointer;
		font-size: 1rem;
	}
	.upload-btn {
		display: block;
		width: calc(100% - 2.5rem);
		margin: 1rem 1.25rem;
		padding: 0.625rem;
		background: #0366d6;
		color: #fff;
		border: 0;
		border-radius: 4px;
		font-size: 0.9375rem;
		font-weight: 500;
		cursor: pointer;
	}
	.upload-btn:disabled {
		background: #888;
		cursor: wait;
	}
	.upload-btn:hover:not(:disabled) {
		background: #0256b9;
	}
	.progress-bar {
		height: 4px;
		background: #e0e0e0;
		margin: 0 1.25rem 1rem;
		border-radius: 2px;
		overflow: hidden;
	}
	.progress-fill {
		height: 100%;
		background: #0366d6;
		transition: width 0.2s;
	}
	.results {
		list-style: none;
		padding: 0;
		margin: 0 1.25rem 1.25rem;
	}
	.results li {
		padding: 0.375rem 0;
		font-size: 0.875rem;
	}
	.results .success {
		color: #1a7f37;
	}
	.results .success a {
		color: inherit;
		font-weight: 600;
	}
	.results .fail {
		color: #b00020;
	}
</style>
