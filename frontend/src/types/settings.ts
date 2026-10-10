// Modified by FnTerminal, 2026-10-08: Uniterm terminal palette types/registry only; desktop settings removed. Apache-2.0.
export interface TerminalThemeColors {
  background: string
  foreground: string
  cursor: string
  selection: string
  black: string
  red: string
  green: string
  yellow: string
  blue: string
  magenta: string
  cyan: string
  white: string
  brightBlack: string
  brightRed: string
  brightGreen: string
  brightYellow: string
  brightBlue: string
  brightMagenta: string
  brightCyan: string
  brightWhite: string
}

export interface CustomTerminalTheme {
  id: string
  name: string
  type: 'dark' | 'light'
  colors: TerminalThemeColors
}

export interface TerminalThemeEntry { label: string; value: string; type: 'dark' | 'light' }

export const TERMINAL_THEMES: TerminalThemeEntry[] = [
  { label: 'uniTerm Dark', value: 'uniterm-dark', type: 'dark' },
  { label: 'uniTerm Light', value: 'uniterm-light', type: 'light' },
  { label: 'Solarized Dark', value: 'solarized-dark', type: 'dark' },
  { label: 'Solarized Light', value: 'solarized-light', type: 'light' },
  { label: 'Monokai', value: 'monokai', type: 'dark' },
  { label: 'Dracula', value: 'dracula', type: 'dark' },
  { label: 'Molokai', value: 'molokai', type: 'dark' },
  { label: 'Tomorrow Night', value: 'tomorrow-night', type: 'dark' },
  { label: 'Tomorrow Night Bright', value: 'tomorrow-night-bright', type: 'dark' },
  { label: 'Tomorrow', value: 'tomorrow', type: 'light' },
  { label: 'One Dark', value: 'one-dark', type: 'dark' },
  { label: 'One Light', value: 'one-light', type: 'light' },
  { label: 'GitHub Dark', value: 'github-dark', type: 'dark' },
  { label: 'GitHub Light', value: 'github-light', type: 'light' },
  { label: 'Gotham', value: 'gotham', type: 'dark' },
  { label: 'Hybrid', value: 'hybrid', type: 'dark' },
  { label: 'Nord', value: 'nord', type: 'dark' },
  { label: 'Gruvbox Dark', value: 'gruvbox-dark', type: 'dark' },
  { label: 'Gruvbox Light', value: 'gruvbox-light', type: 'light' },
  { label: 'Catppuccin Mocha', value: 'catppuccin-mocha', type: 'dark' },
  { label: 'Catppuccin Latte', value: 'catppuccin-latte', type: 'light' },
  { label: 'Tokyo Night', value: 'tokyo-night', type: 'dark' },
  { label: 'Tokyo Day', value: 'tokyo-day', type: 'light' },
  { label: 'Rosé Pine', value: 'rose-pine', type: 'dark' },
  { label: 'Rosé Pine Dawn', value: 'rose-pine-dawn', type: 'light' },
  { label: 'Everforest Dark', value: 'everforest-dark', type: 'dark' },
  { label: 'Everforest Light', value: 'everforest-light', type: 'light' },
  // Popular SSH client defaults
  { label: 'Xshell XTerm', value: 'xshell-xterm', type: 'dark' },
  { label: 'Xshell ANSI on Black', value: 'xshell-ansi-black', type: 'dark' },
  { label: 'Xshell New Black', value: 'xshell-new-black', type: 'dark' },
  { label: 'MobaXterm Default', value: 'mobaxterm-default', type: 'dark' },
  { label: 'MobaXterm Ubuntu', value: 'mobaxterm-ubuntu', type: 'dark' },
  { label: 'FinalShell Dark', value: 'finalshell-dark', type: 'dark' },
  { label: 'FinalShell Light', value: 'finalshell-light', type: 'light' },
]
