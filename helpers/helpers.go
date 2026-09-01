package helpers

import (
	"fmt"
	"golang.org/x/term"
	"log"
	"os"
	"strconv"
	"strings"
	"ti_forge/state"
)

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

func Is_valid_number_no_sci(s string) bool {
	if strings.ContainsAny(s, "eE") {
		return false
	}

	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
