package tui

// ViewMode represents the current view mode.
type ViewMode int

const (
	// ViewModeTree displays a session tree on the left and log viewport on the right.
	ViewModeTree ViewMode = iota
	// ViewModePanel displays multiple panels side by side.
	ViewModePanel
)
