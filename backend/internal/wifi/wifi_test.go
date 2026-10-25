package wifi

import "testing"

func TestSplitTerse(t *testing.T) {
	cases := map[string][]string{
		`*:My\:Network:88:WPA2`: {"*", "My:Network", "88", "WPA2"},
		` :Open:40:--`:          {" ", "Open", "40", "--"},
		`a:b\:c:d`:              {"a", "b:c", "d"},
	}
	for input, expected := range cases {
		got := splitTerse(input)
		if len(got) != len(expected) {
			t.Fatalf("%q: got %#v", input, got)
		}
		for index := range expected {
			if got[index] != expected[index] {
				t.Fatalf("%q: got %#v, want %#v", input, got, expected)
			}
		}
	}
}
