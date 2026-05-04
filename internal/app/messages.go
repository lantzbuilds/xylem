package app

type FileSelectedMsg struct {
	Path  string
	IsDir bool
}

type ThemeChangedMsg struct {
	Name string
}

type LineNumbersToggledMsg struct {
	Enabled bool
}
