import js from '@eslint/js'
import svelte from 'eslint-plugin-svelte'
import globals from 'globals'

export default [
  js.configs.recommended,
  ...svelte.configs['flat/prettier'],
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node
      }
    },
    rules: {
      'no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      // ✅ Отключаем строгие a11y-правила, которые дают ложные срабатывания на кастомных компонентах
      'svelte/a11y-click-events-have-key-events': 'off',
      'svelte/a11y-no-static-element-interactions': 'off',
      'svelte/no-unused-svelte-ignore': 'off'
    }
  },
  {
    ignores: ['dist/**', 'node_modules/**']
  }
]