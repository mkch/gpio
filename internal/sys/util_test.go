package sys

import "testing"

var char32 = func(p []byte) (chars [32]byte) {
	copy(chars[:31], p)
	return
}

func TestStr32(t *testing.T) {
	type testCase struct {
		name string
		arg  [32]byte
		want string
	}
	tests := []testCase{
		testCase{
			name: "simple",
			arg:  char32([]byte{'a', 'b', 'c', 0, 0}),
			want: "abc",
		},
		testCase{
			name: "not terminated",
			arg:  [32]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 'e', 's', 'u', 'v'},
			want: "0123456789abcdefghijklmnopqresu",
		},
		testCase{
			name: "leading 0",
			arg:  char32([]byte{0, 'a'}),
			want: "",
		},
		testCase{
			name: "empty",
			arg:  char32([]byte{}),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Str32(tt.arg); got != tt.want {
				t.Errorf("str32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyStr32(t *testing.T) {
	type testCase struct {
		name string
		arg  string
		want [32]byte
	}
	tests := []testCase{
		testCase{
			name: "simple",
			want: char32([]byte{'a', 'b', 'c', 0, 0}),
			arg:  "abc",
		},
		testCase{
			name: "not terminated",
			want: char32([]byte{'a'}),
			arg:  "a",
		},
		testCase{
			name: "empty",
			want: char32([]byte{}),
			arg:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Char32(tt.arg); got != tt.want {
				t.Errorf("copyStr32() = %v, want %v", got, tt.want)
			}
		})
	}
}
