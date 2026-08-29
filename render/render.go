package render

import(
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"ti_forge/ansi"
	"ti_forge/state"
	"ti_forge/tokens"
	"unicode/utf8"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"golang.org/x/term"
)

type highlight struct {
	Active bool
	Size int
}

func Display_data(state *state.State) {
	program_data := state.Buffers[state.Buffer_idx]

	max_line_num_len := len(strconv.Itoa(len(program_data)))
	line_num_fmtstr := fmt.Sprintf("%%%dd ", max_line_num_len)
	
	_, height := Get_term_size()
	height -= 6

	var builder strings.Builder

	builder.WriteString(ansi.Clear)
	builder.WriteString(ansi.Reset_cursor)

	indent_block := ""

	for i := 0; i < state.Options["indent_size"]; i++ {
		indent_block = indent_block + " "
	}

	highlight := highlight{
		Active: false,
		Size: 0,
	}

	for i := state.Viewport_row; i < state.Viewport_row + height; i++ {
		if i >= len(program_data) {
			break
		}

		var line_builder strings.Builder

		for j, command := range program_data[i] {
			highlight = process_block_highlight(command, highlight, i, j, state.Cursor_row, state.Cursor_col)

			if is_highlighted(i, j, state) {
				line_builder.WriteString(ansi.Highlight)
			} else if state.Options["block_highlight"] == 1 && highlight.Active && highlight.Size == 0 {
				line_builder.WriteString(ansi.Highlight)
				if highlight.Size == 0 && highlight.Active {
					highlight.Active = false
					highlight.Size = 0
				}
			} else {
				line_builder.WriteString(ansi.Reset_text)
			}

			line_builder.WriteString(command)
		}

		line_builder.WriteString(ansi.Reset_text)
		fmt.Fprintf(&builder, line_num_fmtstr, i + 1) // padded line number, starts at 1
		builder.WriteString(strings.Repeat(indent_block, state.Indentation[state.Buffer_idx][i])) // indent
		builder.WriteString(line_builder.String()) // line
		line_builder.WriteString(ansi.Reset_text)
		builder.WriteByte('\n')
	}

	var screen_col int = max_line_num_len+2
	for i := 0; i < state.Cursor_col; i++ {
    screen_col += utf8.RuneCountInString(program_data[state.Cursor_row][i])
	}

	builder.WriteString(ansi.Reset_text)
	
	builder.WriteString(state.File_names[state.Buffer_idx])
	builder.WriteByte('\n')

	var mode_string string = ""
	var prepend_string string = ""
	switch state.Mode {
		case 0:
			mode_string = "NORMAL"
		case 1:
			mode_string = "INSERT"
		case 2:
			mode_string = "COMMAND"
			prepend_string = ":"
		case 3:
			mode_string = "VISUAL"
	}
	builder.WriteString(mode_string)
	builder.WriteString("   ")
	builder.WriteString(prepend_string)
	
	if len(state.Text_buffer) > 0 {
			builder.WriteString(string(state.Text_buffer))
	}

	if state.Mode == 1 || state.Mode == 2 {
		builder.WriteString(ansi.Highlight)
		builder.WriteString(" ")
		builder.WriteString(ansi.Reset_text)
	}

	builder.WriteByte('\n')

	var display_command_matches []string
	if state.Mode == 1 && len(state.Text_buffer) > 0 { // if in insert mode and text buffer has characters,
		var command_matches []string
		command_matches = fuzzy.FindFold(string(state.Text_buffer), tokens.Commands) // find commands that match the text buffer,
		
		if len(command_matches) > 0 {
			var start int = min(state.Suggestion_idx, len(command_matches) - 1)
			var end int = min(5 + state.Suggestion_idx, len(command_matches))
			display_command_matches = command_matches[start:end]
		}
	}

	for idx, command := range display_command_matches { // print the first 5 commands from the selected command
		if idx == 0 {
			builder.WriteString(ansi.Highlight)
		}

		builder.WriteString(command)

		if idx == 0 {
			builder.WriteString(ansi.Reset_text)
		}

		if idx <= 3 {
			builder.WriteByte('\n')
		}
	}

	os.Stdout.WriteString(builder.String())
}

func process_block_highlight(command string, highlight highlight, row int, col int, cursor_row int, cursor_col int) highlight {
	if command == "For(" || command == "Repeat " || command == "While " || command == "If " {
		if row == cursor_row && col == cursor_col {
			highlight.Active = true
			highlight.Size = 0
		}
		highlight.Size++
	} else if command == "End" {
		highlight.Size--

	} else if command == "Else" {
		if highlight.Active {
			highlight.Size--
		} else {
			if row == cursor_row && col == cursor_col {
				highlight.Active = true
				highlight.Size = 1
			}
		}
	}

	return highlight
}

func is_highlighted(row int, col int, state *state.State) bool {
	if !state.Highlighting {
		if row == state.Cursor_row && col == state.Cursor_col {
			return true
		}
		return false
	}

	if row == state.Cursor_row && col == state.Cursor_col {
		return true
	}
	
	start_row, start_col := state.Cursor_row, state.Cursor_col
	end_row, end_col := state.Highlight_row, state.Highlight_col

	if start_row > end_row || (start_row == end_row && start_col > end_col) {
		start_row, end_row = end_row, start_row
		start_col, end_col = end_col, start_col
	}

	if row < start_row || (row == start_row && col < start_col) {
		return false
	}

	if row > end_row || (row == end_row && col > end_col) {
		return false
	}

	return true
}

func Calculate_indentation(state *state.State) {
	fmt.Println(state.Buffers)

	program_data := state.Buffers[state.Buffer_idx]
	state.Indentation[state.Buffer_idx] = make([]int, len(program_data))

	indent := 0
	for row_idx, row := range program_data {
		indent_change := 0
		if len(row) > 0 {
			for col := range row {
				command := program_data[row_idx][col]

				if command == "For(" || command == "Repeat " || command == "While " || command == "If " {
					indent_change++
				} else if command == "End" {
					if indent > 0 {
						indent--
					}
				} else if command == "Else" {
					if indent > 0 {
						indent--
					}
					indent_change++
				}
			}
		}

		state.Indentation[state.Buffer_idx][row_idx] = indent
		indent += indent_change
	}
}

func Get_term_size() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		log.Fatal("Failed to get terminal dimensions: ", err)
	}
	return width, height
}
