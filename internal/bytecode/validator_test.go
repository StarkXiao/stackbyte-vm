package bytecode_test

import (
	"strings"
	"testing"

	"stackbyte-vm/internal/bytecode"
)

func TestValidatorRejectsInvalidBytecode(t *testing.T) {
	tests := []struct {
		name  string
		chunk *bytecode.Chunk
		want  string
	}{
		{"unknown", &bytecode.Chunk{Code: []byte{255}, Lines: []int{1}}, "unknown opcode"},
		{"truncated", &bytecode.Chunk{Code: []byte{byte(bytecode.OpConstant)}, Lines: []int{1}}, "truncated"},
		{"no return", &bytecode.Chunk{Code: []byte{byte(bytecode.OpNil)}, Lines: []int{1}}, "RETURN"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := bytecode.Validate(test.chunk)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}
