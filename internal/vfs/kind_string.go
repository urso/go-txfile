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

// Code generated by "stringer -type=Kind -linecomment=true"; DO NOT EDIT.

package vfs

import "strconv"

const _Kind_name = "unknown OS errorpermission deniedfile already existsfiles does not existfile already closedno space or quota exhaustedprocess file desciptor limit reachedcannot resolve pathread/write IO erroroperation not supportedfile lock failedfile unlock failedunknown error kind"

var _Kind_index = [...]uint16{0, 16, 33, 52, 72, 91, 118, 154, 173, 192, 215, 231, 249, 267}

func (i Kind) String() string {
	if i < 0 || i >= Kind(len(_Kind_index)-1) {
		return "Kind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Kind_name[_Kind_index[i]:_Kind_index[i+1]]
}
