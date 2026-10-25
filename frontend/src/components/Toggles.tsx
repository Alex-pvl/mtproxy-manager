import { useTheme } from '../context/ThemeContext';
import { useLanguage } from '../context/LanguageContext';

function SunIcon() {
  return (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={2}>
      <circle cx="12" cy="12" r="4" />
      <path strokeLinecap="round" d="M12 2v2m0 16v2M4.93 4.93l1.41 1.41m11.32 11.32 1.41 1.41M2 12h2m16 0h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
    </svg>
  );
}

function MoonIcon() {
  return (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={2}>
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </svg>
  );
}

/** Moon / sun segmented switch; the active side gets a raised knob. */
export function ThemeToggle() {
  const { theme, toggleTheme } = useTheme();
  const seg = (on: boolean) =>
    `w-7 h-7 rounded-full flex items-center justify-center transition-colors ${
      on ? 'bg-white dark:bg-gray-700 shadow-sm text-amber-500 dark:text-indigo-400' : 'text-gray-400'
    }`;
  return (
    <button
      type="button"
      onClick={toggleTheme}
      aria-label="Toggle theme"
      className="inline-flex items-center gap-0.5 p-0.5 rounded-full bg-gray-100 dark:bg-gray-800 touch-manipulation shrink-0"
    >
      <span className={seg(theme === 'dark')}><MoonIcon /></span>
      <span className={seg(theme === 'light')}><SunIcon /></span>
    </button>
  );
}

/** Language pill: shows the current language, tap switches. */
export function LangToggle() {
  const { language, setLanguage } = useLanguage();
  return (
    <button
      type="button"
      onClick={() => setLanguage(language === 'ru' ? 'en' : 'ru')}
      aria-label="Toggle language"
      className="h-8 px-3 rounded-full bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 text-sm font-semibold touch-manipulation shrink-0"
    >
      {language === 'ru' ? 'Русский' : 'English'}
    </button>
  );
}
