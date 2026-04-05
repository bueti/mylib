// Themes injected into each epub.js chapter iframe via
// rendition.themes.register(). Keep them simple and readable.

export const themes = {
	light: {
		body: {
			background: '#ffffff',
			color: '#222222'
		},
		a: { color: '#0366d6' }
	},
	sepia: {
		body: {
			background: '#f4ecd8',
			color: '#3b3024'
		},
		a: { color: '#8b4513' }
	},
	dark: {
		body: {
			background: '#1a1a1a',
			color: '#d0d0d0'
		},
		a: { color: '#66b3ff' }
	}
} as const;

export type ThemeName = keyof typeof themes;

export const fontSizes: Record<string, string> = {
	S: '90%',
	M: '110%',
	L: '130%'
};

export type FontSize = keyof typeof fontSizes;
