package convert

import (
	"encoding/hex"
	"log"
	"os"
	"strings"
	"encoding/binary"
	"errors"
	"github.com/mehonje/TI-Convert/tokens"
)

var Commands []string

func Read8xp(path string) ([]byte, [4]string) {
	path = strings.TrimSpace(path)   // Remove whitespace
	if !strings.HasSuffix(path, ".8xp") { // If file path doesn't have ".8xp" suffix,
		path = path + ".8xp" // Append it
	}


	var program_metadata [4]string
	var program_data []byte
	byte_data, err := os.ReadFile(path) // Read file data
	if err != nil {
		log.Fatal("Failed to read file data: ", err)
	}

	if len(byte_data) > 76 { // If data is more than 76 bytes long,
		program_metadata[0] = string(byte_data[60:67])                  // Store bytes 60 - 67 (program name)
		program_metadata[1] = string(byte_data[11:52])                  // Store bytes 11 - 52 (transmission comment)
		program_metadata[2] = hex.EncodeToString([]byte{byte_data[59]}) // Store byte 59 (type id)
		program_metadata[3] = hex.EncodeToString([]byte{byte_data[69]}) // Store bytes 69 (flag)
		program_data = byte_data[74 : len(byte_data)-2]                 // Store bytes 74 - end-2 (program), remove the first 74 bytes (program metadata) and last 2 bytes (checksum)
	}

	return program_data, program_metadata
}

func Data_to_strings(program_data []byte, program_metadata [4]string) [][]string {
	var lines [][]string = [][]string{}
	
	lines = append(lines, []string{})
	for _, char := range program_metadata[0] {
		if char == rune(0x00) {
			break
		}
		lines[0] = append(lines[0], string(char))
	}
	lines[0] = append(lines[0], "\n")
	lines = append(lines, []string{})
	for _, char := range program_metadata[1] {
		if char == rune(0x00) {
			break
		}
		lines[1] = append(lines[1], string(char))
	}
	lines[1] = append(lines[1], "\n")
	lines = append(lines, []string{program_metadata[2], "\n"})
	lines = append(lines, []string{program_metadata[3], "\n"})

	var i int
	var program_data_len int = len(program_data)
	var line []string = []string{}

	for i < program_data_len {
		curr_byte := program_data[i]
		var next_byte byte

		step := 1
		if i < program_data_len-1 {
			next_byte = program_data[i+1]
			
			switch curr_byte {
			case 0xbb:
				s, ok := tokens.Tokens_bb[next_byte] // Check if mapping exists
				if ok {
					line = append(line, s) // Replace if yes,
					step = 2
				} else {
					line= append(line, string(curr_byte)) // Turn into string if no
				}
			case 0xef:
				s, ok := tokens.Tokens_ef[next_byte] // Check if mapping exists
				if ok {
					line = append(line, s) // Replace if yes,
					step = 2
				} else {
					line = append(line, "Wait ") // Add "wait" command (0xef) if no
				}
			case 0x63:
				s, ok := tokens.Tokens_63[next_byte] // Check if mapping exists
				if ok {
					line = append(line, s) // Replace if yes,
					step = 2
				} else {
					line = append(line, string(curr_byte)) // Turn into string if no
				}
			case 0x5d:
				s, ok := tokens.Tokens_5d[next_byte] // Check if mapping exists
				if ok {
					line = append(line, s) // Replace if yes,
					step = 2
				} else {
					line = append(line, "/") // Add division operator (0x5d) if no
				}
			case 0x7e:
				s, ok := tokens.Tokens_7e[next_byte] // Check if mapping exists
				if ok {
					line = append(line, s) // Replace if yes
					step = 2
				} else {
					line = append(line, string(curr_byte)) // Turn into string if no
				}
			default:
				s, ok := tokens.Tokens[curr_byte] // Check if normal mapping exists
				if ok {
					line = append(line, s) // Replace if yes,
					if s == "\n" {
						lines = append(lines, line)
						line = []string{}
					}
				} else {
					line = append(line, string(curr_byte)) // Turn into string if no
				}
			}
		} else {
			s, ok := tokens.Tokens[curr_byte] // Check if normal mapping exists
			if ok {
				line = append(line, s) // Replace if yes,
				if s == "\n" {
					lines = append(lines, line)
					line = []string{}
				}
			} else {
				line = append(line, string(curr_byte)) // Turn into string if no
			}
		}
		i += step
	}

	if len(line) > 0 {
		line = append(line, "\n")
		lines = append(lines, line)
	}
	
	return lines
}

func Txt_to_eightxp(to_path string, program_lines [][]string) error {
	if len(program_lines) < 4 {
		return errors.New("Progam must be at least 4 lines")
	}

	var metadata [4][]byte = [4][]byte{}
	for _, command := range program_lines[0][:len(program_lines[0]) - 1] {
		arr, ok := tokens.Reverse_tokens[command]
		if ok {
			for _, token_byte := range arr {
				metadata[0] = append(metadata[0], token_byte)	
			}
		}
	}
	for _, command := range program_lines[1][:len(program_lines[1]) - 1] {
		arr, ok := tokens.Reverse_tokens[command]
		if ok {
			for _, token_byte := range arr {
				metadata[1] = append(metadata[1], token_byte)	
			}
		}
	}
	
	{
		var string_byte string = strings.Join(program_lines[2][:len(program_lines[2])-1], "")
		b, err := hex.DecodeString(string_byte)
		if err != nil {
			return errors.New("Failed to convert string\"" + string_byte + "\" to byte")
		}
		metadata[2] = b
	}
	{
		var string_byte string = strings.Join(program_lines[3][:len(program_lines[3])-1], "")
		b, err := hex.DecodeString(string_byte)
		if err != nil {
			return errors.New("Failed to convert string\"" + string_byte + "\" to byte")
		}
		metadata[3] = b
	}
	if len(metadata[0]) > 8 {
		return errors.New("Program name (line 1) cannot be longer than 8 characters")
	}
	if len(metadata[1]) > 42 {
		return errors.New("Program comment (line 2) cannot be longer than 42 characters")
	}

	var program_byte_data []byte = []byte{}

	program_byte_data = append(program_byte_data, 0x2a, 0x2a, 0x54, 0x49, 0x38, 0x33, 0x46, 0x2a) // Append signature
	program_byte_data = append(program_byte_data, 0x1a, 0x0a)                                     // Append signature_part_2
	program_byte_data = append(program_byte_data, 0x0a)                                           // Append mystery byte
	{                                                                                             // Append comment
		comment_padded := make([]byte, 42)
		copy(comment_padded, []byte(metadata[1]))
		program_byte_data = append(program_byte_data, comment_padded...)
	}
	program_byte_data = append(program_byte_data, 0x00, 0x00) // Append placeholder meta_and_body_length. Set later on
	program_byte_data = append(program_byte_data, 0x0d)       // Append flag
	program_byte_data = append(program_byte_data, 0x00)       // Append unknown
	program_byte_data = append(program_byte_data, 0x00, 0x00) // Append placeholder body_and_checksum_length. Set later
	program_byte_data = append(program_byte_data, metadata[2][0]) // Append file type
	{ // Append program_name
		name_padded := make([]byte, 8)
		copy(name_padded, []byte(metadata[0]))
		program_byte_data = append(program_byte_data, name_padded...)
	}
	program_byte_data = append(program_byte_data, 0x00) // Append version
	program_byte_data = append(program_byte_data, metadata[3][0]) // Append is_archived
	program_byte_data = append(program_byte_data, 0x00, 0x00) // Append placeholder body_and_checksum_length_2. Set later
	program_byte_data = append(program_byte_data, 0x00, 0x00) // Append placeholder body_length. Set later

	var program_lines_no_meta [][]string = program_lines[4:]
	var program_commands []string = []string{}
	for _, line := range program_lines_no_meta {
		var tmp_array []string = []string{}
		for _, command := range line {
			tmp_array = append(tmp_array, command)
		}
		program_commands = append(program_commands, tmp_array...)
	}

	var body_length uint16 = 2
	{ // Append program data
		for _, command := range program_commands {
			arr, ok := tokens.Reverse_tokens[command]
			if ok {
				for _, token_byte := range arr {
					body_length++
					program_byte_data = append(program_byte_data, token_byte)
				}
			}
		}
	}

	{ // Set meta_and_body_length
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(len(program_byte_data)-57))
		program_byte_data[53] = buf[0]
		program_byte_data[54] = buf[1]
	}
	{ // Set body_and_checksum_length
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, body_length)
		program_byte_data[57] = buf[0]
		program_byte_data[58] = buf[1]
	}
	{ // Set body_and_checksum_length_2
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, body_length)
		program_byte_data[70] = buf[0]
		program_byte_data[71] = buf[1]
	}
	{ // Set body_length
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, body_length-2)
		program_byte_data[72] = buf[0]
		program_byte_data[73] = buf[1]
	}
	{ // Append checksum
		var checksum uint16 = 0
		for i := 55; i < len(program_byte_data); i++ {
			checksum += uint16(program_byte_data[i])
		}
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, checksum)
		program_byte_data = append(program_byte_data, buf[0], buf[1])
	}

	err := os.WriteFile(to_path, program_byte_data, 0644)
	if err != nil {
		return errors.New("Failed to create " + to_path + ", " + err.Error())
	}

	return nil
}

func init() {
	for key, val := range tokens.Tokens {
		Commands = append(Commands, val)
	}
	for key, val := range tokens.Tokens_bb {
		Commands = append(Commands, val)
	}
	for key, val := range tokens.Tokens_ef {
		Commands = append(Comands, val)
	}
	for key, val := range tokens.Tokens_63 {
		Commands = append(Commands, val)
	}
	for key, val := range tokens.Tokens_5d {
		Commands = append(Commands, val)
	}
	for key, val := range tokens.Tokens_7e {
		Commands = append(Commands, val)
	}
}
