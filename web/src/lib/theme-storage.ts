/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
export const THEME_STORAGE_KEYS = {
  mode: 'suixing:theme:v1:mode',
  preset: 'suixing:theme:v1:preset',
  font: 'suixing:theme:v1:font',
  radius: 'suixing:theme:v1:radius',
  scale: 'suixing:theme:v1:scale',
  contentLayout: 'suixing:theme:v1:content-layout',
} as const

/**
 * Keys written by earlier builds. Visitors keep their preference instead of
 * being reset on the first paint after the namespace change; each value is
 * copied to the new key and the old one is dropped.
 */
const LEGACY_THEME_STORAGE_KEYS: Record<string, string> = Object.fromEntries(
  Object.values(THEME_STORAGE_KEYS).map((key) => [
    key,
    key.replace('suixing:', 'newapi:'),
  ])
)

export function readThemePreference<T extends string>(
  key: string,
  allowed: ReadonlySet<T>,
  fallback: T
): T {
  if (typeof window === 'undefined') return fallback

  try {
    // Legacy cookies are shared across ports, so importing them would restore
    // preferences that may belong to another local instance.
    let value = window.localStorage.getItem(key)
    if (!value) {
      const legacyKey = LEGACY_THEME_STORAGE_KEYS[key]
      const legacyValue = legacyKey
        ? window.localStorage.getItem(legacyKey)
        : null
      if (legacyValue) {
        window.localStorage.setItem(key, legacyValue)
        window.localStorage.removeItem(legacyKey as string)
        value = legacyValue
      }
    }
    return value && allowed.has(value as T) ? (value as T) : fallback
  } catch {
    return fallback
  }
}

export function writeThemePreference(key: string, value: string | null): void {
  if (typeof window === 'undefined') return

  try {
    if (value === null) {
      window.localStorage.removeItem(key)
    } else {
      window.localStorage.setItem(key, value)
    }
  } catch {
    // Keep theme controls usable when storage is blocked or full.
  }
}
