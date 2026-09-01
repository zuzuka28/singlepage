const savedTheme = localStorage.getItem('singlepage-theme');
const initialTheme = savedTheme || (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');

document.documentElement.dataset.theme = initialTheme;
document.querySelector<HTMLMetaElement>('meta[name="theme-color"]')?.setAttribute(
  'content',
  initialTheme === 'dark' ? '#1d2027' : '#f7f8fb',
);
