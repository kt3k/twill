package twill

import "fmt"

// Position is a 1-based line and column in a source text.
type Position struct {
	Line   int
	Column int
}

// Error is raised by the compiler. Syntax errors carry a position.
type Error struct {
	Message  string
	Position *Position
}

func (e *Error) Error() string {
	if e.Position != nil {
		return fmt.Sprintf("%s (%d:%d)", e.Message, e.Position.Line, e.Position.Column)
	}
	return e.Message
}

// Errorf creates an Error without a position.
func Errorf(format string, args ...any) *Error {
	return &Error{Message: fmt.Sprintf(format, args...)}
}

// PositionAt computes the line and column of index in input.
func PositionAt(input string, index int) Position {
	line, column := 1, 1
	if index > len(input) {
		index = len(input)
	}
	for i := 0; i < index; i++ {
		if input[i] == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return Position{Line: line, Column: column}
}
