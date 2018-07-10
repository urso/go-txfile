// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package txerr

type Error interface {
	error

	Op() string
	Kind() error
	Message() string
	Cause() error
}

// Selective accessors. These accessors allows user to implement a subset of
// Error, but still use the query-functions like Is(err, <kind>).
type (
	withOp       interface{ Op() string }
	withKind     interface{ Kind() error }
	withMessage  interface{ Message() string }
	withChild    interface{ Cause() error }
	withChildren interface{ Causes() []error }
)

// FindErrWith returns the first error in the error tree, that matches the
// given predicate.
func FindErrWith(in error, pred func(err error) bool) error {
	var found error
	Iter(in, func(err error) bool {
		matches := pred(err)
		if matches {
			found = err
			return false
		}
		return true
	})

	return found
}

// Iter iterates the complete error tree call fn on each found error value.
// The user function fn can stop the iteration by returning false.
func Iter(in error, fn func(err error) bool) {
	doIter(in, fn)
}

func doIter(in error, fn func(err error) bool) bool {
	for {
		if in == nil {
			return true
		}

		if cont := fn(in); !cont {
			return cont
		}

		switch err := in.(type) {
		case withChild:
			in = err.Cause()

		case withChildren:
			for _, sub := range err.Causes() {
				if cont := doIter(sub, fn); !cont {
					return cont
				}
			}
			return true

		default:
			return true
		}
	}
}

func directMsg(in error) string {
	if err, ok := in.(withMessage); ok {
		return err.Message()
	}
	return ""
}
