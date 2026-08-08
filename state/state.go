package state

type State struct {
	Quit bool
	Cursor_row int
	Cursor_col int
	Mode int
	Text_buffer []rune
	Suggestion_idx int
	Input []byte
	Buffers [][][]string
	File_names []string
	Buffer_idx int
	Highlight_row int
	Highlight_col int
	Highlighting bool
	Copy_buffer [][]string
	Options map[string]int
}

