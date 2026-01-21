import { Language } from './enums';

export interface LanguageMetadata {
    name: string;
    nativeName: string;
    code: Language;
    searchQuery: string;
}

export const LANGUAGE_CONFIG: Record<Language, LanguageMetadata> = {
    [Language.ENGLISH]: {
        name: 'English',
        nativeName: 'English',
        code: Language.ENGLISH,
        searchQuery: '',
    },
    [Language.RUSSIAN]: {
        name: 'Russian',
        nativeName: 'Русский',
        code: Language.RUSSIAN,
        searchQuery: 'Russian Translation (RU)',
    },
    [Language.TURKISH]: {
        name: 'Turkish',
        nativeName: 'Türkçe',
        code: Language.TURKISH,
        searchQuery: 'Türkçe çeviri',
    },
    [Language.FRENCH]: {
        name: 'French',
        nativeName: 'Français',
        code: Language.FRENCH,
        searchQuery: 'French Translation',
    },
};
