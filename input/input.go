package input

import(
	"math/rand/v2"
	"os"
	"reflect"
	"slices"
	"strings"
	"ti_forge/convert"
	"ti_forge/state"

	"github.com/atotto/clipboard"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/pkg/browser"
)

func Get_input(state state.State) state.State {
    buf := make([]byte, 10)
    n, err := os.Stdin.Read(buf)
    if err != nil || n == 0 {
        state.Input = []byte{}
				return state
    }
		state.Input = buf[:n]
		return state
}

func Process_input(state state.State) state.State {
	switch state.Mode {
		case 0: state = Process_normal_input(state)
		case 1: state = Process_insert_input(state)
		case 2: state = Process_command_input(state)
		case 3: state = Process_visual_input(state)
	}

	if state.Cursor_row < 0 {
		state.Cursor_row = 0
	} else if state.Cursor_row >= len(state.Program_data[state.Buffer_idx]) {
		state.Cursor_row = len(state.Program_data[state.Buffer_idx]) - 1
	}

	var line_length int = len(state.Program_data[state.Buffer_idx][state.Cursor_row])
	if state.Cursor_col < 0 {
		state.Cursor_col = 0
	} else if state.Cursor_col >= line_length {
		state.Cursor_col = line_length - 1
	}

	switch {
		case slices.Equal(state.Input, []byte{27}): // [esc]
			state.Mode = 0
			state.Text_buffer = []rune{}
			state.Highlighting = false
	}

	return state
}

func Process_normal_input(state state.State) state.State {
	switch {
		case slices.Equal(state.Input, []byte{104}): // [h], left
			state.Cursor_col--
		case slices.Equal(state.Input, []byte{108}): // [l], right
			state.Cursor_col++
		case slices.Equal(state.Input, []byte{106}): // [j], down
			state.Cursor_row++
		case slices.Equal(state.Input, []byte{107}): // [k], up
			state.Cursor_row--
		case slices.Equal(state.Input, []byte{105}): // [i], enter insert mode
			state.Mode = 1
		case slices.Equal(state.Input, []byte{58}): // [:], enter command mode
			state.Mode = 2
		case slices.Equal(state.Input, []byte{120}): // [x], delete command at cursor
			state.Program_data[state.Buffer_idx][state.Cursor_row] = slices.Delete(state.Program_data[state.Buffer_idx][state.Cursor_row], state.Cursor_col, state.Cursor_col + 1)
			if slices.Equal(state.Program_data[state.Buffer_idx][state.Cursor_row], []string{}) {
				state.Program_data[state.Buffer_idx] = slices.Delete(state.Program_data[state.Buffer_idx], state.Cursor_row, state.Cursor_row + 1)
				if len(state.Program_data) == 0 {
					state.Program_data = [][][]string{{{""}}}
				}
			}
		case slices.Equal(state.Input, []byte{118}): // [v], enter visual mode
			state.Mode = 3
			state.Highlighting = true
			state.Highlight_row = state.Cursor_row
			state.Highlight_col = state.Cursor_col
		case slices.Equal(state.Input, []byte{112}): // [p], put
			if len(state.Copy_buffer) == 1 {
				slices.Reverse(state.Copy_buffer[0])
				for _, command := range state.Copy_buffer[0] {
					if command == "\n" {
						continue
					}
					state.Program_data[state.Buffer_idx][state.Cursor_row] = slices.Insert(state.Program_data[state.Buffer_idx][state.Cursor_row], state.Cursor_col, command)
				}
			} else {
				slices.Reverse(state.Copy_buffer)
				for _, line := range state.Copy_buffer {
					state.Program_data[state.Buffer_idx] = slices.Insert(state.Program_data[state.Buffer_idx], state.Cursor_row, line)
				}
			}
	default:
		if len(state.Input) == 1 {
			state.Text_buffer = append(state.Text_buffer, rune(state.Input[0]))
		}
	}
	
	switch string(state.Text_buffer) {
		case "dd": // delete line
			if len(state.Program_data[state.Buffer_idx]) > 1 {
				state.Program_data[state.Buffer_idx] = slices.Delete(state.Program_data[state.Buffer_idx], state.Cursor_row, state.Cursor_row + 1)
			}
			state.Text_buffer = []rune{}
	}

	return state
}

func Process_insert_input(state state.State) state.State {
	switch {
		case slices.Equal(state.Input, []byte{127}): // [backspace]
			if len(state.Text_buffer) > 0 {
				state.Text_buffer = state.Text_buffer[:len(state.Text_buffer) - 1]
			}
		case slices.Equal(state.Input, []byte{9}): // [tab], suggestion down
			state.Suggestion_idx++

			var command_matches []string
			command_matches = fuzzy.FindFold(string(state.Text_buffer), convert.Commands) // find commands that match the text buffer,
		
			state.Suggestion_idx = bound_suggestion_idx(state.Suggestion_idx, len(command_matches))

		case slices.Equal(state.Input, []byte{27, 91, 90}): // [shift][tab], suggestion up
			state.Suggestion_idx--

			var command_matches []string
			command_matches = fuzzy.FindFold(string(state.Text_buffer), convert.Commands) // find commands that match the text buffer,
		
			state.Suggestion_idx = bound_suggestion_idx(state.Suggestion_idx, len(command_matches))
		case slices.Equal(state.Input, []byte{13}): // [enter]
			if len(state.Text_buffer) > 0 { // if text buffer has characters
				var command_matches []string
				command_matches = fuzzy.FindFold(string(state.Text_buffer), convert.Commands) // find commands that match the text buffer,
		
				state.Suggestion_idx = bound_suggestion_idx(state.Suggestion_idx, len(command_matches))

				if len(command_matches) > 1 {
					var idx int = min(state.Suggestion_idx, len(command_matches) - 1)
					state.Program_data[state.Buffer_idx][state.Cursor_row] = slices.Insert(state.Program_data[state.Buffer_idx][state.Cursor_row], state.Cursor_col, command_matches[idx])

					state.Input = []byte{}
					state.Suggestion_idx = 0
					state.Text_buffer = []rune{}
				}
			}
		default:
			if len(state.Input) == 1 {
				state.Text_buffer = append(state.Text_buffer, rune(state.Input[0]))
			}
	}

	return state
}

func bound_suggestion_idx(idx int, suggestions int) int {
	if idx >= suggestions {
		idx = max(suggestions - 1, 0)
	} else if idx < 0 {
		idx = 0
	}

	return idx
}

func Process_command_input(state state.State) state.State {
	switch {
		case slices.Equal(state.Input, []byte{127}): // [backspace]
			if len(state.Text_buffer) > 0 {
				state.Text_buffer = state.Text_buffer[:len(state.Text_buffer) - 1]
			}
		case slices.Equal(state.Input, []byte{13}): // [enter]
			var split_command []string = strings.Split(string(state.Text_buffer), " ")

			state.Mode = 0
			state.Text_buffer = []rune{}

			switch split_command[0] {
				case "w": // write, second element is file path (if it exists)
					if len(split_command) > 1 {
						err := convert.Txt_to_eightxp(split_command[1], state.Program_data[state.Buffer_idx])
						if err != nil {
							state.Text_buffer = []rune(err.Error())
						}
					} else {
						err := convert.Txt_to_eightxp(state.File_names[state.Buffer_idx], state.Program_data[state.Buffer_idx])
						if err != nil {
							state.Text_buffer = []rune(err.Error())
						}
					}
				case "q": // quit
					state.Quit = true
				case "e": // edit, second element is file path
					var program_data []byte = []byte{}
					var program_metadata [4]string = [4]string{}
					program_data, program_metadata, err := convert.Read8xp(split_command[1])

					if err != nil {
						state.Text_buffer = []rune(err.Error())
					} else {
						program_data_commands := convert.Data_to_strings(program_data, program_metadata)
						if reflect.DeepEqual(state.Program_data, [][][]string{{{"\n"}}}) {
							state.Program_data = [][][]string{program_data_commands}
							state.File_names = []string{split_command[1]}
						} else {
							state.Program_data = append(state.Program_data, program_data_commands)
							state.File_names = append(state.File_names, split_command[1])
							state.Buffer_idx++
						}
					}
				case "bn": // next buffer
					if state.Buffer_idx < len(state.Program_data) - 1 {
						state.Buffer_idx++
					} else {
						state.Text_buffer = []rune("Reached last buffer")
					}
				case "bp": // previous buffer
					if state.Buffer_idx > 0 {
						state.Buffer_idx--
						state.Text_buffer = []rune("Reached first buffer")
					}
				case "help": // help
					num := rand.IntN(100)
					if num == 0 {
						err := browser.OpenURL("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
						if err != nil {
							state.Text_buffer = []rune("Failed to open browser: " + err.Error())
						}
					}

					err := browser.OpenURL("https://github.com/mehonje/TI-Forge")
					if err != nil {
						state.Text_buffer = []rune("Failed to open browser: " + err.Error())
					} else {
						state.Text_buffer = []rune("Check your web browser")
					}
				default:
					state.Text_buffer = []rune("Unknown command \"" + split_command[0] + "\"")
			}
		default: 
			if len(state.Input) == 1 {
				state.Text_buffer = append(state.Text_buffer, rune(state.Input[0]))
			}
	}

	return state
}

func Process_visual_input(state state.State) state.State {
	switch {
		case slices.Equal(state.Input, []byte{104}): // [h], left
			state.Cursor_col--
		case slices.Equal(state.Input, []byte{108}): // [l], right
			state.Cursor_col++
		case slices.Equal(state.Input, []byte{106}): // [j], down
			state.Cursor_row++
		case slices.Equal(state.Input, []byte{107}): // [k], up
			state.Cursor_row--
		case slices.Equal(state.Input, []byte{121}): // [y], yank
			start_row, start_col := state.Cursor_row, state.Cursor_col
			end_row, end_col := state.Highlight_row, state.Highlight_col
	
			state.Copy_buffer = Copy_selection(start_row, start_col, end_row, end_col, state.Program_data[state.Buffer_idx])

			state.Mode = 0
			state.Highlighting = false
		case slices.Equal(state.Input, []byte{89}): // [Y], yank to clipboard
			start_row, start_col := state.Cursor_row, state.Cursor_col
			end_row, end_col := state.Highlight_row, state.Highlight_col
	
			var buffer [][]string = Copy_selection(start_row, start_col, end_row, end_col, state.Program_data[state.Buffer_idx])

			var builder strings.Builder

			for _, line := range buffer {
				for _, command := range line {
					builder.WriteString(command)
				}
			}
			
			err := clipboard.WriteAll(builder.String())
				if err != nil {
					state.Text_buffer = []rune("Failed to set clipboard : " + err.Error())
				}

			state.Mode = 0
			state.Highlighting = false
	}

	return state
}

func Copy_selection(start_row int, start_col int, end_row int, end_col int, program_data [][]string) [][]string {
	if start_row > end_row || (start_row == end_row && start_col > end_col) {
		start_row, end_row = end_row, start_row
		start_col, end_col = end_col, start_col
	}

	var buffer = [][]string{}
	
	if start_row == end_row {
		buffer = append(buffer, slices.Clone(program_data[start_row][start_col:end_col + 1]))
	} else {
		buffer = append(buffer, slices.Clone(program_data[start_row]))
		buffer[0] = buffer[0][start_col:]

		if start_row != end_row {
			for row := start_row + 1; row < end_row; row++ {
				buffer = append(buffer, slices.Clone(program_data[row]))
			}

			buffer = append(buffer, slices.Clone(program_data[end_row]))
			buffer[len(buffer) - 1] = buffer[len(buffer) - 1][:end_col + 1]
		}
	}

	return buffer
}

