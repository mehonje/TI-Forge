package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"ti_forge/convert"
	"unicode/utf8"

	"golang.org/x/term"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

func main() {
	enable_utf8()

	var cursor_row int = 1
	var cursor_col int = 1
	var text_buffer []rune = []rune{}
	var mode int = 0
	
	fd := int(os.Stdin.Fd())
	cooked_state, err := term.MakeRaw(fd)
	if err != nil {
		log.Fatal("Failed to enter raw mode: ", err)
	}
	defer term.Restore(fd, cooked_state)

	var program_data []byte = convert.Read8xp("C:/Users/Jack/Onedrive/Desktop/go_work/ti_forge/TEST3.8xp")
	program_data_commands := convert.Data_to_strings(program_data)
	for {
		cursor_row, cursor_col = display_data(program_data_commands, cursor_row, cursor_col, text_buffer, mode)

		var quit bool = false
		quit, text_buffer, cursor_row, cursor_col = process_input(get_input(), cursor_row, cursor_col, text_buffer)
		
		if quit {
			return
		}
	}
}

func display_data(program_data_lines [][]string, cursor_row int, cursor_col int, text_buffer []rune, mode int) (int, int) {
	if cursor_row < 1 {
		cursor_row = 1
	}
	if cursor_row > len(program_data_lines) {
		cursor_row = len(program_data_lines)
	}
	if cursor_col < 1 {
		cursor_col = 1
	}
	var line_length int = len(program_data_lines[cursor_row-1])
	if cursor_col > line_length {
		cursor_col = line_length
	}

	max_line_num_len := len(strconv.Itoa(len(program_data_lines)))
	line_num_fmtstr := fmt.Sprintf("%%%ds %%s", max_line_num_len)

	fmt.Print("\033[H\033[2J")

	_, height := get_term_size()
	height -= 6
	half_height := height/2
	var buffer_row int = cursor_row-1
	var buffer_start int = max(0, buffer_row-half_height)
	var buffer_end int = min(buffer_row+half_height, len(program_data_lines))

	var builder strings.Builder

	for i := buffer_start; i < buffer_end; i++ {
		var line_builder strings.Builder
		for _, command := range program_data_lines[i] {
			line_builder.WriteString(command)
		}
		fmt.Fprintf(&builder, line_num_fmtstr, strconv.Itoa(i+1),line_builder.String())
	}
	fmt.Print(builder.String())
	var mode_string string = ""
	switch mode {
	case 0:
		mode_string = "NORMAL"
	}
	fmt.Print(mode_string, "   ", string(text_buffer), "\n")
	
	if len(text_buffer) > 0 {
		var command_matches []string = fuzzy.FindFold(string(text_buffer), convert.Tokens)
		for _, command := range command_matches[0:min(5, len(command_matches))] {
			fmt.Println(command)
		}
	}

	var screen_row int = buffer_row-buffer_start
	var screen_col int = max_line_num_len+2
	for i := 0; i < cursor_col-1; i++ {
    screen_col += utf8.RuneCountInString(program_data_lines[cursor_row-1][i])
	}

	fmt.Printf("\033[%d;%dH", screen_row+1, screen_col) // move cursor
	
	return cursor_row, cursor_col
}

func get_input() byte {
    buf := make([]byte, 1)
    n, err := os.Stdin.Read(buf)
    if err != nil || n == 0 {
        return 0
    }
    return buf[0]
}

func process_input(input byte, cursor_row int, cursor_col int, text_buffer []rune) (bool, []rune, int, int) {
	var quit bool = false
	var new_text_buffer []rune = text_buffer

	switch input {
	case 113: // "q"
		quit = true
	}

	var rune_input rune = rune(input)
	switch rune_input {
	case 'h':
		cursor_col--
	case 'l':
		cursor_col++
	case 'j':
		cursor_row++
	case 'k':
		cursor_row--
	default:
		new_text_buffer = append(new_text_buffer, rune_input)
	}

	return quit, new_text_buffer, cursor_row, cursor_col
}

func get_term_size() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		log.Fatal("Failed to get terminal dimensions: ", err)
	}
	return width, height
}

func enable_utf8() {
	cmd := exec.Command("cmd.exe", "/c", "chcp 65001 > nul")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

