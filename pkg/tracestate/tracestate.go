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

package tracestate // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/tracestate"

import (
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// Take tracestate here not span, no need to send whole span
func GetPValue(ts pcommon.TraceState) int {
	state := ts.AsRaw()
	if state == "" {
		return 0
	}
	// ot=p:3;r:2,other=val:1
	idx := findOT(state)
	if idx < 0 {
		return 0
	}
	// p:3;r:2,other=val:1
	state = state[idx+len("ot="):]
	ot := strings.SplitN(state, ",", 1)
	if len(ot) == 0 {
		return 0
	}
	// p:3;r:2
	args := strings.Split(ot[0], ";")
	for _, arg := range args {
		// p:3
		if !strings.HasPrefix(arg, "p:") {
			continue
		}

		// 3
		arg = arg[len("p:"):]
		p, err := strconv.Atoi(strings.TrimSpace(arg))
		if err != nil {
			return 0
		}
		if p < 0 || p > 62 {
			return 0
		}
		return p
	}
	return 0
}

func findOT(state string) int {
	var base int
	for {
		tmp := state[base:]
		idx := strings.Index(tmp, "ot=")
		switch idx {
		case -1: // not found
			return -1
		case 0: // found
			return base
		default: // possibly found
			if tmp[idx-1] != ',' && state[idx-1] != ' ' { // not found
				base += idx + 3 // 3 == len("ot=")
				continue
			}
			// found
			return base + idx
		}
	}
}
