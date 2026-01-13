// Copyright 2020-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package linker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoOpDescriptors(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, noOpFile)
	assert.NotNil(t, noOpMessage)
	assert.NotNil(t, noOpOneof)
	assert.NotNil(t, noOpField)
	assert.NotNil(t, noOpEnum)
	assert.NotNil(t, noOpEnumValue)
	assert.NotNil(t, noOpExtension)
	assert.NotNil(t, noOpService)
	assert.NotNil(t, noOpMethod)
}

func TestFeatureFieldDescriptors(t *testing.T) {
	t.Parallel()
	// Sanity checks the initialized values of these package vars.
	assert.NotNil(t, fieldPresenceField, "field_presence")
	assert.NotNil(t, repeatedFieldEncodingField, "repeated_field_encoding")
	assert.NotNil(t, messageEncodingField, "message_encoding")
	assert.NotNil(t, enumTypeField, "enum_type")
	assert.NotNil(t, jsonFormatField, "json_format")
}
