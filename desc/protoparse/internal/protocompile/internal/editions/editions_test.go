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

package editions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestGetEditionDefaults(t *testing.T) {
	t.Parallel()
	// Make sure all supported editions have defaults.
	for _, edition := range SupportedEditions {
		features := GetEditionDefaults(edition)
		require.NotNil(t, features)
	}
	// Spot check some things
	features := GetEditionDefaults(descriptorpb.Edition_EDITION_PROTO2)
	require.NotNil(t, features)
	assert.Equal(t, descriptorpb.FeatureSet_CLOSED, features.GetEnumType())
	assert.Equal(t, descriptorpb.FeatureSet_EXPLICIT, features.GetFieldPresence())
	assert.Equal(t, descriptorpb.FeatureSet_EXPANDED, features.GetRepeatedFieldEncoding())

	features = GetEditionDefaults(descriptorpb.Edition_EDITION_PROTO3)
	require.NotNil(t, features)
	assert.Equal(t, descriptorpb.FeatureSet_OPEN, features.GetEnumType())
	assert.Equal(t, descriptorpb.FeatureSet_IMPLICIT, features.GetFieldPresence())
	assert.Equal(t, descriptorpb.FeatureSet_PACKED, features.GetRepeatedFieldEncoding())

	features = GetEditionDefaults(descriptorpb.Edition_EDITION_2023)
	require.NotNil(t, features)
	assert.Equal(t, descriptorpb.FeatureSet_OPEN, features.GetEnumType())
	assert.Equal(t, descriptorpb.FeatureSet_EXPLICIT, features.GetFieldPresence())
	assert.Equal(t, descriptorpb.FeatureSet_PACKED, features.GetRepeatedFieldEncoding())
}

func TestComputeSupportedEditions(t *testing.T) {
	t.Parallel()
	assert.Equal(t,
		map[string]descriptorpb.Edition{
			"2023": descriptorpb.Edition_EDITION_2023,
		},
		computeEditionsRange(descriptorpb.Edition_EDITION_2023, descriptorpb.Edition_EDITION_2023),
	)
	assert.Equal(t,
		map[string]descriptorpb.Edition{
			"2023": descriptorpb.Edition_EDITION_2023,
			"2024": descriptorpb.Edition_EDITION_2024,
		},
		computeEditionsRange(descriptorpb.Edition_EDITION_2023, descriptorpb.Edition_EDITION_2024),
	)
}
