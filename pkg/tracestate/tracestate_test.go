// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tracestate

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestProcessorStart(t *testing.T) {
	for _, tc := range []struct {
		name     string
		state    string
		expected int
	}{
		{"no tracestate", "", 0},
		{"no pvalue", "ot=r:9;foo:bar", 0},
		{"normal pvalue", "ot=p:1", 1},
		{"normal pvalue, more state", "ot=p:2;r:9;foo:bar", 2},
		{"invalid pvalue", "ot=p:foo", 0},
		{"too big pvalue", "ot=p:63", 0},
		{"too small pvalue", "ot=p:-1", 0},
		{"trailing space", "ot=p:9 ;foo:bar", 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Make a dummy span to stuff tracestate on
			ts := pcommon.NewTraceState()
			ts.FromRaw(tc.state)
			assert.Equal(t, tc.expected, GetPValue(ts))
		})
	}
}

func BenchmarkTracestateToPValue(b *testing.B) {
	tests := []struct {
		name     string
		state    string
		expected int
	}{
		{
			name:     "empty tracestate",
			state:    "",
			expected: 0,
		},
		{
			name:     "no ot",
			state:    "shop=p:3",
			expected: 0,
		},
		{
			name:     "basic",
			state:    "ot=p:3",
			expected: 3,
		},
		{
			name:     "multiples",
			state:    "ot=p:4;r:8",
			expected: 4,
		},
		{
			name:     "with custom",
			state:    "ot=p:4;r:8,shop=val:1",
			expected: 4,
		},
		{
			name:     "has ot but not the rest",
			state:    "ot=",
			expected: 0,
		},
		{
			name:     "second field",
			state:    "shot=p:1,ot=p:3;r:4",
			expected: 3,
		},
		{
			name:     "second field (malformed)",
			state:    "shot=hello:1,ot=p+3;r:2",
			expected: 0,
		},
		{
			name:     "malformed",
			state:    "ot=p+3;r:2",
			expected: 0,
		},
		{
			name:     "malformed2",
			state:    "shoot=p:3;r:4",
			expected: 0,
		},
		{
			name:     "malformed3",
			state:    "ot=p+k3;r:2",
			expected: 0,
		},
		{
			name:     "not actually a number",
			state:    "ot=p:yerawizardharry;r:2",
			expected: 0,
		},
	}
	for _, tt := range tests {
		b.Run("ours/"+tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ts := pcommon.NewTraceState()
				ts.FromRaw(tt.state)
				assert.Equal(b, tt.expected, GetPValue(ts))
			}
		})
	}
}
