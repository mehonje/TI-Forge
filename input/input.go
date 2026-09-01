package input

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"ti_forge/ansi"
	"ti_forge/convert"
	"ti_forge/render"
	"ti_forge/state"
	"ti_forge/tokens"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/pkg/browser"
)

func Get_input(state *state.State) {
    buf := make([]byte, 10)
    n, err := os.Stdin.Read(buf)
    if err != nil || n == 0 {
        state.Input = []byte{}
    }
		state.Input = buf[:n]
}

func Process_input(state *state.State) {
	old_buffers := make([][][]string, len(state.Buffers))
	for i, buffer := range state.Buffers {
		old_buffers[i] = make([][]string, len(buffer))
		for j, row := range buffer {
			old_buffers[i][j] = slices.Clone(row)
		}
	}

	switch state.Mode {
		case 0: Process_normal_input(state)
		case 1: Process_insert_input(state)
		case 2: Process_command_input(state)
		case 3: Process_visual_input(state)
	}

	if len(state.Buffers) == 0 {
		state.Cursor_row = 0
		state.Cursor_col = 0
	} else {
		if state.Cursor_row < 0 {
			state.Cursor_row = 0
		} else if state.Cursor_row >= len(state.Buffers[state.Buffer_idx]) {
			state.Cursor_row = len(state.Buffers[state.Buffer_idx]) - 1
		}
	
		var line_length int = len(state.Buffers[state.Buffer_idx][state.Cursor_row])
		if line_length == 0 {
			state.Cursor_col = 0
		} else if state.Cursor_col < 0 {
				state.Cursor_col = 0
		} else if state.Cursor_col >= line_length {
			state.Cursor_col = line_length - 1
		}
	}
	
	changed := !slices.EqualFunc(state.Buffers, old_buffers, func(slice1, slice2 [][]string) bool {
		return slices.EqualFunc(slice1, slice2, func(row1, row2 []string) bool {
			return slices.Equal(row1, row2)
		})
	})

	if changed {
		render.Calculate_indentation(state)
	}

	switch {
		case slices.Equal(state.Input, []byte{27}): // [esc]
			state.Mode = 0
			state.Text_buffer = []rune{}
			state.Highlighting = false
	}
}

func Process_normal_input(state *state.State) {
	if len(state.Input) == 1 {
		state.Text_buffer = append(state.Text_buffer, rune(state.Input[0]))
	}
	
	var old_text_buffer []rune = slices.Clone(state.Text_buffer) // set text buffer to empty, reset to current state if no match found
	state.Text_buffer = []rune{}

	switch string(old_text_buffer) {
		case "h": // [h], move cursor left
			state.Cursor_col--
		case "l": // [l], right
			state.Cursor_col++
		case "j": // [j], down
			state.Cursor_row++
		case "k": // [k], up
			state.Cursor_row--
		case "i": // [i], insert, enter insert mode
			state.Mode = 1
		case "a": // [a], append, enter insert mode after current command
			state.Cursor_col++
			state.Mode = 1
		case ":": // [:], enter command mode
			state.Mode = 2
		case "x": // [x], delete command at cursor
			if state.Buffers[state.Buffer_idx][state.Cursor_row][state.Cursor_col] != "　" {
				state.Buffers[state.Buffer_idx][state.Cursor_row] = slices.Delete(state.Buffers[state.Buffer_idx][state.Cursor_row], state.Cursor_col, state.Cursor_col + 1)
				if slices.Equal(state.Buffers[state.Buffer_idx][state.Cursor_row], []string{}) {
					state.Buffers[state.Buffer_idx] = slices.Delete(state.Buffers[state.Buffer_idx], state.Cursor_row, state.Cursor_row + 1)
					if len(state.Buffers) == 0 {
						state.Buffers = [][][]string{{{""}}}
					}
				}
			}
		case "v": // [v], enter visual mode
			state.Mode = 3
			state.Highlighting = true
			state.Highlight_row = state.Cursor_row
			state.Highlight_col = state.Cursor_col
		case "p": // [p], put
			if len(state.Copy_buffer) == 1 {
				slices.Reverse(state.Copy_buffer[0])
				for _, command := range state.Copy_buffer[0] {
					if command == "\n" {
						continue
					}
					state.Buffers[state.Buffer_idx][state.Cursor_row] = slices.Insert(state.Buffers[state.Buffer_idx][state.Cursor_row], state.Cursor_col, command)
				}
			} else {
				slices.Reverse(state.Copy_buffer)
				for _, line := range state.Copy_buffer {
					state.Buffers[state.Buffer_idx] = slices.Insert(state.Buffers[state.Buffer_idx], state.Cursor_row, line)
				}
			}
		case "dd": // delete line
			if len(state.Buffers[state.Buffer_idx]) > 1 {
				state.Buffers[state.Buffer_idx] = slices.Delete(state.Buffers[state.Buffer_idx], state.Cursor_row, state.Cursor_row + 1)
			}
		case "gg": // go to top
			state.Cursor_row = 0
			state.Cursor_col = 0
		case "G": // go to bottom
			state.Cursor_row = len(state.Buffers[state.Buffer_idx]) - 1
			state.Cursor_col = 0
		case "I": // go to beginning of line
			state.Cursor_col = 0
			state.Mode = 1
		case "A": // go to beginning of line
			state.Cursor_col = len(state.Buffers[state.Buffer_idx][state.Cursor_row]) - 1
			state.Mode = 1
		case "o": // [o], newline below cursor
			state.Buffers[state.Buffer_idx] = slices.Insert(state.Buffers[state.Buffer_idx], state.Cursor_row + 1, []string{"　"})
			state.Cursor_row++
			state.Mode = 1
		case "O": // [O], newline above cursor
			state.Buffers[state.Buffer_idx] = slices.Insert(state.Buffers[state.Buffer_idx], state.Cursor_row, []string{"　"})
			state.Mode = 1
		default:
			state.Text_buffer = old_text_buffer
	}

	bound_cursor(state)
}

func Process_insert_input(state *state.State) {
	switch {
		case slices.Equal(state.Input, []byte{127}): // [backspace]
			if len(state.Text_buffer) > 0 {
				state.Text_buffer = state.Text_buffer[:len(state.Text_buffer) - 1]
			}
		case slices.Equal(state.Input, []byte{9}): // [tab], suggestion down
			state.Suggestion_idx++

			command_matches := get_commands(state)

			state.Suggestion_idx = bound_suggestion_idx(state.Suggestion_idx, len(command_matches))
		case slices.Equal(state.Input, []byte{27, 91, 90}): // [shift][tab], suggestion up
			state.Suggestion_idx--

			command_matches := get_commands(state)
		
			state.Suggestion_idx = bound_suggestion_idx(state.Suggestion_idx, len(command_matches))
		case slices.Equal(state.Input, []byte{13}): // [enter]
			if len(state.Text_buffer) > 0 { // if text buffer has characters
				command_matches := get_commands(state)
		
				state.Suggestion_idx = bound_suggestion_idx(state.Suggestion_idx, len(command_matches) - 1)

				if len(command_matches) >= 1 {
					var idx int = min(state.Suggestion_idx, len(command_matches))

					var commands []string

					s := command_matches[idx]

					if strings.HasSuffix(s, "　<expand>") {
						for _, char := range strings.TrimSuffix(s, "　<expand>") {
							in := slices.Contains(tokens.Commands, string(char))
							if in {
								commands = append(commands, string(char))
							}
						}
					} else {
						alias, ok := tokens.Token_aliases[s]
						if ok {
							commands = append(commands, alias)
						} else {
							commands = append(commands, s)
						}
					}

					slices.Reverse(commands)
					for _, command := range commands {
						state.Buffers[state.Buffer_idx][state.Cursor_row] = slices.Insert(state.Buffers[state.Buffer_idx][state.Cursor_row], state.Cursor_col, command)
					}

					state.Input = []byte{}
					state.Suggestion_idx = 0
					state.Text_buffer = []rune{}
					state.Cursor_col += len(commands)
				}
			}
		default:
			if len(state.Input) == 1 {
				state.Text_buffer = append(state.Text_buffer, rune(state.Input[0]))
			}
	}
}

func bound_suggestion_idx(idx int, suggestions int) int {
	if idx >= suggestions {
		idx = max(suggestions - 1, 0)
	} else if idx < 0 {
		idx = 0
	}

	return idx
}

func Process_command_input(state *state.State) {
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
						err := convert.Txt_to_eightxp(split_command[1], state.Buffers[state.Buffer_idx])
						if err != nil {
							state.Text_buffer = []rune(ansi.Bold + ansi.Red + err.Error() + ansi.Reset_text)
						}
					} else {
						err := convert.Txt_to_eightxp(state.File_names[state.Buffer_idx], state.Buffers[state.Buffer_idx])
						if err != nil {
							state.Text_buffer = []rune(ansi.Bold + ansi.Red + err.Error() + ansi.Reset_text)
						}
					}
				case "q": // quit
					state.Quit = true
				case "e": // edit, second element is file path
					var program_data []byte = []byte{}
					var program_metadata [4]string = [4]string{}
					program_data, program_metadata, err := convert.Read8xp(split_command[1])

					if err != nil {
						state.Text_buffer = []rune(ansi.Bold + ansi.Red + err.Error() + ansi.Reset_text)
					} else {
						program_data_commands := convert.Data_to_strings(program_data, program_metadata)
						if reflect.DeepEqual(state.Buffers, [][][]string{{{"　"}}}) {
							state.Buffers = [][][]string{program_data_commands}
							state.File_names = []string{split_command[1]}
						} else {
							state.Buffers = append(state.Buffers, program_data_commands)
							state.Indentation = append(state.Indentation, make([]int, 0))
							state.File_names = append(state.File_names, split_command[1])
							state.Buffer_idx++
						}
					}
				case "bn": // next buffer
					if state.Buffer_idx < len(state.Buffers) - 1 {
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
							state.Text_buffer = []rune(ansi.Bold + ansi.Red + "Failed to open browser: " + err.Error() + ansi.Reset_text)
						}
					}

					err := browser.OpenURL("https://github.com/mehonje/TI-Forge")
					if err != nil {
						state.Text_buffer = []rune(ansi.Bold + ansi.Red + "Failed to open browser: " + err.Error() + ansi.Reset_text)
					} else {
						state.Text_buffer = []rune("Check your web browser")
					}
				case "set": // change option, second element is option name, third is value
					val, err := strconv.Atoi(split_command[2])
					if err != nil {
						state.Text_buffer = []rune(ansi.Bold + ansi.Red + "Failed to set option: " + err.Error() + ansi.Reset_text)
					} else {
						err = set_option(split_command[1], val, state)
						if err != nil {
							state.Text_buffer = []rune(ansi.Bold + ansi.Red + "Failed to set option: " + err.Error() + ansi.Reset_text)
						}
					}
				case "lbl": // go to label, second argument is label name
					if len(split_command[1]) > 2 {
						state.Text_buffer = []rune(ansi.Bold + ansi.Red + "Labels cannot be more than 2 characters long" + ansi.Reset_text)
						break
					}

					label := []rune(split_command[1])

					if !unicode.IsLetter(label[0]) && !unicode.IsDigit(label[0]){
						state.Text_buffer = []rune(ansi.Bold + ansi.Red + "Labels can only contain alphanumeric values" + ansi.Reset_text)
						break
					}
					if len(label) > 1 {
						if !unicode.IsLetter(label[1]) && !unicode.IsDigit(label[1]){
							state.Text_buffer = []rune(ansi.Bold + ansi.Red + "Labels can only contain alphanumeric values" + ansi.Reset_text)
							break
						}
					}

					label_found := false

					OuterLoop:
					for row_idx, line := range state.Buffers[state.Buffer_idx] {
						for col_idx, token := range line {
							if token != "Lbl " {
								continue
							}

							var found_label []rune
							if len(line) > col_idx + 1 {
								found_label = append(found_label, []rune(line[col_idx + 1])[0])
							}
							if len(line) > col_idx + 2 {
								found_label = append(found_label, []rune(line[col_idx + 2])[0])
							}
							if slices.Equal(found_label, label) {
								state.Cursor_row = row_idx
								state.Cursor_col = col_idx
								label_found = true
								break OuterLoop
							}
						}
					}

					if label_found {
						break
					}

		
					state.Text_buffer = []rune(ansi.Bold + ansi.Red + "Label \"" + string(label) + "\" not found" + ansi.Reset_text)
				default:
					state.Text_buffer = []rune(ansi.Bold + ansi.Red + "Unknown command \"" + split_command[0] + "\"" + ansi.Reset_text)
			}
		default: 
			if len(state.Input) == 1 {
				state.Text_buffer = append(state.Text_buffer, rune(state.Input[0]))
			}
	}
}

func Process_visual_input(state *state.State) {
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
	
			state.Copy_buffer = Copy_selection(start_row, start_col, end_row, end_col, state.Buffers[state.Buffer_idx])

			state.Mode = 0
			state.Highlighting = false
		case slices.Equal(state.Input, []byte{89}): // [Y], yank to clipboard
			start_row, start_col := state.Cursor_row, state.Cursor_col
			end_row, end_col := state.Highlight_row, state.Highlight_col
	
			var buffer [][]string = Copy_selection(start_row, start_col, end_row, end_col, state.Buffers[state.Buffer_idx])

			var builder strings.Builder

			for _, line := range buffer {
				for _, command := range line {
					builder.WriteString(command)
				}
			}
			
			err := clipboard.WriteAll(builder.String())
				if err != nil {
					state.Text_buffer = []rune(ansi.Bold + ansi.Red + "Failed to set clipboard : " + err.Error() + ansi.Reset_text)
				}

			state.Mode = 0
			state.Highlighting = false
	}
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

func set_option(option string, value int, state *state.State) error {
	_, ok := state.Options[option]
	if !ok {
		return fmt.Errorf("Unknown option: \"%s\"", option)
	}

	if option == "indent_size" && value < 0 {
		return errors.New("Value cannot be negative for option \"indent_size\"")
	}

	if option == "block_highlight" && (value != 0 && value != 1) {
		return errors.New("Value must be boolean (1 or 0) for option \"block_highlight\"")
	}

	state.Options[option] = value

	return nil
}

func bound_cursor(state *state.State) {
	lines := len(state.Buffers[state.Buffer_idx])
	_, height := render.Get_term_size()
	height -= 6

	state.Cursor_row = max(0, min(state.Cursor_row, lines))

	if state.Cursor_row >= state.Viewport_row + height {
		state.Viewport_row = state.Cursor_row - height + 1
	}

	if state.Cursor_row < state.Viewport_row {
		state.Viewport_row = state.Cursor_row
	}

	max_viewport := max(0, lines - height)
	state.Viewport_row = max(0, min(state.Viewport_row, max_viewport))
}

func is_valid_number_no_sci(s string) bool {
	if strings.ContainsAny(s, "eE") {
		return false
	}

	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func get_commands(state *state.State) []string {
	var command_matches []string
	command_matches = fuzzy.FindFold(string(state.Text_buffer), tokens.Commands) // find commands that match the text buffer,

	capitalised := strings.ToUpper(string(state.Text_buffer))

	alphabetic := true
	for _, char := range capitalised {
		if char < 'A' || char > 'Z' {
			alphabetic = false
			break
		}
	}

	if alphabetic {
		command_matches = append([]string{capitalised + "　<expand>"}, command_matches...)
	}

	numeric := is_valid_number_no_sci(string(state.Text_buffer))

	if numeric {
		command_matches = append([]string{string(state.Text_buffer) + "　<expand>"}, command_matches...)
	}

	return command_matches
}
