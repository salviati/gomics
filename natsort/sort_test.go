/*
	Copyright 2014 Google Inc. All rights reserved.

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package natsort

import (
	"testing"
)

func TestLess(t *testing.T) {
	tests := []struct {
		s, t string
		want cmp
	}{
		{"", "", eq},
		{"a", "", gt},
		{"a", "a", eq},
		{"1", "10", lt},
		{"20", "3", gt},
		{"a1", "a10", lt},
		{"a2", "a10", lt},
		{"a5b2", "a5b7", lt},
		{"a50b2", "a6b7", gt},
		{"世20", "世界3", lt},
		{"50a", "50b", lt},
		{"a50", "a050", gt},
		{"a01b3", "a1b2", lt},
		{"thx1138", "thx1138", eq},
		{"thx1138a", "thx1138b", lt},
		{"thx1138a", "thx1138", gt},

		// a < a0 < a1 < a1a < a1b < a2 < a10 < a20
		{"a", "a0", lt},
		{"a0", "a1", lt},
		{"a1", "a1a", lt},
		{"a1a", "a1b", lt},
		{"a2", "a10", lt},
		{"a10", "a20", lt},

		// 1.001 < 1.002 < 1.010 < 1.02 < 1.1 < 1.3
		{"1.001", "1.002", lt},
		{"1.002", "1.010", lt},
		{"1.010", "1.02", gt}, // TODO(light): should this be lt?
		{"1.02", "1.1", gt},   // TODO(light): should this be lt?
		{"1.1", "1.3", lt},
	}
	for _, test := range tests {
		v := bit(Less(test.s, test.t))
		v |= bit(Less(test.t, test.s)) << 1
		if cmp(v) != test.want {
			t.Errorf("%[1]q %[3]v %[2]q, want %[1]q %[4]v %[2]q", test.s, test.t, cmp(v), test.want)
		}
	}
}

func bit(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

type cmp int

const (
	eq cmp = iota
	lt
	gt
)

func (c cmp) String() string {
	switch c {
	case lt:
		return "<"
	case gt:
		return ">"
	case eq:
		return "=="
	default:
		return "<>"
	}
}
