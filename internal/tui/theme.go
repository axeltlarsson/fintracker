package tui

import (
	lipgloss "charm.land/lipgloss/v2"
	"image/color"
)

// Design token pattern
// 1. Primitive tokens (palette) - stable across an app, no opinion about usage per se.
// 2. Semantic tokens (styles) - carry meaning, reference Primitive tokens (palette)
// 3. Component tokens - consume semantic tokens only

// Theme defines a color palette.
// Based on Rosé Pine (https://rosepinetheme.com/palette/).
//
// Backgrounds:
//   Base       — primary background; application frames, sidebars, tabs,
//                and extensions to the focal context
//   Surface    — secondary background atop base; cards, inputs, status lines
//   Overlay    — tertiary background atop surface; popovers, notifications, dialogs
//
// Foregrounds:
//   Muted      — low contrast foreground; ignored content, disabled elements,
//                unfocused text [comments]
//   Subtle     — medium contrast foreground; secondary content, comments,
//                punctuation, tab names [operators, punctuation]
//   Text       — high contrast foreground; primary content, normal text,
//                variables, active content [variables]
//
// Highlights:
//   HighlightLow  — low contrast highlight; cursorline background
//   HighlightMed  — medium contrast highlight; selection background paired
//                   with text foreground
//   HighlightHigh — high contrast highlight; borders, visual dividers,
//                   cursor background paired with text foreground
//
// Accents:
//   Love — diagnostic errors, deleted git files, terminal red/bright red.
//          "Per favore ama tutti." [builtins]
//   Gold — diagnostic warnings, terminal yellow/bright yellow.
//          "Lemon tea on a summer morning." [strings]
//   Rose — matching search background paired with base foreground,
//          modified git files, terminal cyan/bright cyan.
//          "A beautiful yet cautious blossom." [booleans, functions]
//   Pine — renamed git files, terminal green/bright green.
//          "Fresh winter greenery." [conditionals, keywords]
//   Foam — diagnostic information, git additions, terminal blue/bright blue.
//          "Saltwater tidepools." [keys, tags, types]
//   Iris — diagnostic hints, inline links, merged and staged git modifications,
//          terminal magenta/bright magenta.
//          "Smells of groundedness." [methods, parameters]

type Theme struct {
	// Rosé Pine palette
	Base          color.Color // primary background #191724
	Surface       color.Color // secondary background atop base #1f1d2e
	Overlay       color.Color // tertiary background atop surface #26233a
	Muted         color.Color // low-contrast foreground #6e6a86
	Subtle        color.Color // medium contrast foreground #908caa
	Text          color.Color // high contrast foreground #e0def4
	HighlightLow  color.Color // low contrast highligt #21202e
	HighlightMed  color.Color // medium contrast highligt #403d52
	HighlightHigh color.Color // high contrast highlight #524f67
	Love          color.Color // diagnostic errors #eb6f92
	Gold          color.Color // diagnostic warnings #f6c177
	Rose          color.Color // primary accent #ebbcba
	Pine          color.Color // secondary accent #31748f
	Foam          color.Color // links, info #9ccfd8
	Iris          color.Color // highlights #c4a7e7
}

var RoséPineMain = Theme{
	Base:          lipgloss.Color("#191724"),
	Surface:       lipgloss.Color("#1f1d2e"),
	Overlay:       lipgloss.Color("#26233a"),
	Muted:         lipgloss.Color("#6e6a86"),
	Subtle:        lipgloss.Color("#908caa"),
	Text:          lipgloss.Color("#e0def4"),
	HighlightLow:  lipgloss.Color("#21202e"),
	HighlightMed:  lipgloss.Color("#403d52"),
	HighlightHigh: lipgloss.Color("#524f67"),
	Love:          lipgloss.Color("#eb6f92"),
	Gold:          lipgloss.Color("#f6c177"),
	Rose:          lipgloss.Color("#ebbcba"),
	Pine:          lipgloss.Color("#31748f"),
	Foam:          lipgloss.Color("#9ccfd8"),
	Iris:          lipgloss.Color("#c4a7e7"),
}

var RoséPineMoon = Theme{
	Base:          lipgloss.Color("#232136"),
	Surface:       lipgloss.Color("#2a273f"),
	Overlay:       lipgloss.Color("#393552"),
	Muted:         lipgloss.Color("#6e6a86"),
	Subtle:        lipgloss.Color("#908caa"),
	Text:          lipgloss.Color("#e0def4"),
	HighlightLow:  lipgloss.Color("#2a283e"),
	HighlightMed:  lipgloss.Color("#44415a"),
	HighlightHigh: lipgloss.Color("#56526e"),
	Love:          lipgloss.Color("#eb6f92"),
	Gold:          lipgloss.Color("#f6c177"),
	Rose:          lipgloss.Color("#ea9a97"),
	Pine:          lipgloss.Color("#3e8fb0"),
	Foam:          lipgloss.Color("#9ccfd8"),
	Iris:          lipgloss.Color("#c4a7e7"),
}

var RoséPineDawn = Theme{
	Base:          lipgloss.Color("#faf4ed"),
	Surface:       lipgloss.Color("#fffaf3"),
	Overlay:       lipgloss.Color("#f2e9e1"),
	Muted:         lipgloss.Color("#9893a5"),
	Subtle:        lipgloss.Color("#797593"),
	Text:          lipgloss.Color("#575279"),
	HighlightLow:  lipgloss.Color("#f4ede8"),
	HighlightMed:  lipgloss.Color("#dfdad9"),
	HighlightHigh: lipgloss.Color("#cecacd"),
	Love:          lipgloss.Color("#b4637a"),
	Gold:          lipgloss.Color("#ea9d34"),
	Rose:          lipgloss.Color("#d7827e"),
	Pine:          lipgloss.Color("#286983"),
	Foam:          lipgloss.Color("#56949f"),
	Iris:          lipgloss.Color("#907aa9"),
}
