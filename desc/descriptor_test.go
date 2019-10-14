package desc

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	_ "github.com/golang/protobuf/protoc-gen-go/plugin"
	_ "github.com/golang/protobuf/ptypes/empty"
	_ "google.golang.org/genproto/protobuf/api"
	_ "google.golang.org/genproto/protobuf/field_mask"
	_ "google.golang.org/genproto/protobuf/ptype"
	_ "google.golang.org/genproto/protobuf/source_context"

	"github.com/jhump/protoreflect/internal"
	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestFileDescriptorObjectGraph(t *testing.T) {
	// This checks the structure of the descriptor for desc_test1.proto to make sure
	// the "rich descriptor" accurately models everything therein.
	fd, err := loadProtoset("../internal/testprotos/desc_test1.protoset")
	testutil.Ok(t, err)
	checkDescriptor(t, "file", 0, fd, nil, fd, descCase{
		name: "desc_test1.proto",
		references: map[string]childCases{
			"messages": {(*FileDescriptor).GetMessageTypes, []descCase{
				{
					name: "testprotos.TestMessage",
					references: map[string]childCases{
						"fields": {(*MessageDescriptor).GetFields, []descCase{
							{
								name: "testprotos.TestMessage.nm",
								references: map[string]childCases{
									"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.TestMessage.NestedMessage")},
									"enum type":    {(*FieldDescriptor).GetEnumType, nil},
									"one of":       {(*FieldDescriptor).GetOneOf, nil},
								},
							},
							{
								name: "testprotos.TestMessage.anm",
								references: map[string]childCases{
									"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.TestMessage.NestedMessage.AnotherNestedMessage")},
									"enum type":    {(*FieldDescriptor).GetEnumType, nil},
									"one of":       {(*FieldDescriptor).GetOneOf, nil},
								},
							},
							{
								name: "testprotos.TestMessage.yanm",
								references: map[string]childCases{
									"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage")},
									"enum type":    {(*FieldDescriptor).GetEnumType, nil},
									"one of":       {(*FieldDescriptor).GetOneOf, nil},
								},
							},
							{
								name: "testprotos.TestMessage.ne",
								references: map[string]childCases{
									"message type": {(*FieldDescriptor).GetMessageType, nil},
									"enum type":    {(*FieldDescriptor).GetEnumType, refs("testprotos.TestMessage.NestedEnum")},
									"one of":       {(*FieldDescriptor).GetOneOf, nil},
								},
							},
						}},
						// this rabbit hole goes pretty deep...
						"nested messages": {(*MessageDescriptor).GetNestedMessageTypes, []descCase{
							{
								name: "testprotos.TestMessage.NestedMessage",
								references: map[string]childCases{
									"fields": {(*MessageDescriptor).GetFields, []descCase{
										{
											name: "testprotos.TestMessage.NestedMessage.anm",
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.TestMessage.NestedMessage.AnotherNestedMessage")},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
										{
											name: "testprotos.TestMessage.NestedMessage.yanm",
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage")},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
									}},
									"nested messages": {(*MessageDescriptor).GetNestedMessageTypes, []descCase{
										{
											name: "testprotos.TestMessage.NestedMessage.AnotherNestedMessage",
											references: map[string]childCases{
												"fields": {(*MessageDescriptor).GetFields, []descCase{
													{
														name: "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.yanm",
														references: map[string]childCases{
															"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage")},
															"enum type":    {(*FieldDescriptor).GetEnumType, nil},
															"one of":       {(*FieldDescriptor).GetOneOf, nil},
														},
													},
												}},
												"nested messages": {(*MessageDescriptor).GetNestedMessageTypes, []descCase{
													{
														name: "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage",
														references: map[string]childCases{
															"nested fields": {(*MessageDescriptor).GetFields, []descCase{
																{
																	name: "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.foo",
																	references: map[string]childCases{
																		"message type": {(*FieldDescriptor).GetMessageType, nil},
																		"enum type":    {(*FieldDescriptor).GetEnumType, nil},
																		"one of":       {(*FieldDescriptor).GetOneOf, nil},
																	},
																},
																{
																	name: "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.bar",
																	references: map[string]childCases{
																		"message type": {(*FieldDescriptor).GetMessageType, nil},
																		"enum type":    {(*FieldDescriptor).GetEnumType, nil},
																		"one of":       {(*FieldDescriptor).GetOneOf, nil},
																	},
																},
																{
																	name: "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.baz",
																	references: map[string]childCases{
																		"message type": {(*FieldDescriptor).GetMessageType, nil},
																		"enum type":    {(*FieldDescriptor).GetEnumType, nil},
																		"one of":       {(*FieldDescriptor).GetOneOf, nil},
																	},
																},
																{
																	name: "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.dne",
																	references: map[string]childCases{
																		"message type": {(*FieldDescriptor).GetMessageType, nil},
																		"enum type":    {(*FieldDescriptor).GetEnumType, refs("testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum")},
																		"one of":       {(*FieldDescriptor).GetOneOf, nil},
																	},
																},
																{
																	name: "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.anm",
																	references: map[string]childCases{
																		"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.TestMessage.NestedMessage.AnotherNestedMessage")},
																		"enum type":    {(*FieldDescriptor).GetEnumType, nil},
																		"one of":       {(*FieldDescriptor).GetOneOf, nil},
																	},
																},
																{
																	name: "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.nm",
																	references: map[string]childCases{
																		"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.TestMessage.NestedMessage")},
																		"enum type":    {(*FieldDescriptor).GetEnumType, nil},
																		"one of":       {(*FieldDescriptor).GetOneOf, nil},
																	},
																},
																{
																	name: "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.tm",
																	references: map[string]childCases{
																		"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.TestMessage")},
																		"enum type":    {(*FieldDescriptor).GetEnumType, nil},
																		"one of":       {(*FieldDescriptor).GetOneOf, nil},
																	},
																},
															}},
															"nested messages": {(*MessageDescriptor).GetNestedMessageTypes, nil},
															"nested enums": {(*MessageDescriptor).GetNestedEnumTypes, []descCase{
																{
																	name: "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum",
																	references: map[string]childCases{
																		"values": {(*EnumDescriptor).GetValues, children(
																			"testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum.VALUE1",
																			"testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum.VALUE2"),
																		},
																	},
																},
															}},
															"nested extensions": {(*MessageDescriptor).GetNestedExtensions, nil},
															"one ofs":           {(*MessageDescriptor).GetOneOfs, nil},
														},
													},
												}},
												"nested enums": {(*MessageDescriptor).GetNestedEnumTypes, nil},
												"nested extensions": {(*MessageDescriptor).GetNestedExtensions, []descCase{
													{
														name:   "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.flags",
														number: 200,
														references: map[string]childCases{
															"owner":        {(*FieldDescriptor).GetOwner, refs("testprotos.AnotherTestMessage")},
															"message type": {(*FieldDescriptor).GetMessageType, nil},
															"enum type":    {(*FieldDescriptor).GetEnumType, nil},
															"one of":       {(*FieldDescriptor).GetOneOf, nil},
														},
													},
												}},
												"one ofs": {(*MessageDescriptor).GetOneOfs, nil},
											},
										},
									}},
									"nested enums":      {(*MessageDescriptor).GetNestedEnumTypes, nil},
									"nested extensions": {(*MessageDescriptor).GetNestedExtensions, nil},
									"one ofs":           {(*MessageDescriptor).GetOneOfs, nil},
								},
							},
						}},
						"nested enums": {(*MessageDescriptor).GetNestedEnumTypes, []descCase{
							{
								name: "testprotos.TestMessage.NestedEnum",
								references: map[string]childCases{
									"values": {(*EnumDescriptor).GetValues, children(
										"testprotos.TestMessage.NestedEnum.VALUE1", "testprotos.TestMessage.NestedEnum.VALUE2"),
									},
								},
							},
						}},
						"nested extensions": {(*MessageDescriptor).GetNestedExtensions, nil},
						"one ofs":           {(*MessageDescriptor).GetOneOfs, nil},
					},
				},
				{
					name: "testprotos.AnotherTestMessage",
					references: map[string]childCases{
						"fields": {(*MessageDescriptor).GetFields, []descCase{
							{
								name: "testprotos.AnotherTestMessage.dne",
								references: map[string]childCases{
									"message type": {(*FieldDescriptor).GetMessageType, nil},
									"enum type":    {(*FieldDescriptor).GetEnumType, refs("testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum")},
									"one of":       {(*FieldDescriptor).GetOneOf, nil},
								},
							},
							{
								name: "testprotos.AnotherTestMessage.map_field1",
								references: map[string]childCases{
									"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.AnotherTestMessage.MapField1Entry")},
									"enum type":    {(*FieldDescriptor).GetEnumType, nil},
									"one of":       {(*FieldDescriptor).GetOneOf, nil},
								},
							},
							{
								name: "testprotos.AnotherTestMessage.map_field2",
								references: map[string]childCases{
									"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.AnotherTestMessage.MapField2Entry")},
									"enum type":    {(*FieldDescriptor).GetEnumType, nil},
									"one of":       {(*FieldDescriptor).GetOneOf, nil},
								},
							},
							{
								name: "testprotos.AnotherTestMessage.map_field3",
								references: map[string]childCases{
									"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.AnotherTestMessage.MapField3Entry")},
									"enum type":    {(*FieldDescriptor).GetEnumType, nil},
									"one of":       {(*FieldDescriptor).GetOneOf, nil},
								},
							},
							{
								name: "testprotos.AnotherTestMessage.map_field4",
								references: map[string]childCases{
									"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.AnotherTestMessage.MapField4Entry")},
									"enum type":    {(*FieldDescriptor).GetEnumType, nil},
									"one of":       {(*FieldDescriptor).GetOneOf, nil},
								},
							},
							{
								name: "testprotos.AnotherTestMessage.rocknroll",
								references: map[string]childCases{
									"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.AnotherTestMessage.RockNRoll")},
									"enum type":    {(*FieldDescriptor).GetEnumType, nil},
									"one of":       {(*FieldDescriptor).GetOneOf, nil},
								},
							},
							{
								name: "testprotos.AnotherTestMessage.str",
								references: map[string]childCases{
									"message type": {(*FieldDescriptor).GetMessageType, nil},
									"enum type":    {(*FieldDescriptor).GetEnumType, nil},
									"one of":       {(*FieldDescriptor).GetOneOf, refs("testprotos.AnotherTestMessage.atmoo")},
								},
							},
							{
								name: "testprotos.AnotherTestMessage.int",
								references: map[string]childCases{
									"message type": {(*FieldDescriptor).GetMessageType, nil},
									"enum type":    {(*FieldDescriptor).GetEnumType, nil},
									"one of":       {(*FieldDescriptor).GetOneOf, refs("testprotos.AnotherTestMessage.atmoo")},
								},
							},
							{
								name: "testprotos.AnotherTestMessage.withoptions",
								references: map[string]childCases{
									"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.AnotherTestMessage.WithOptions")},
									"enum type":    {(*FieldDescriptor).GetEnumType, nil},
									"one of":       {(*FieldDescriptor).GetOneOf, nil},
								},
							},
						}},
						"one ofs": {(*MessageDescriptor).GetOneOfs, []descCase{
							{
								name:       "testprotos.AnotherTestMessage.atmoo",
								skipParent: true,
								references: map[string]childCases{
									"fields": {(*OneOfDescriptor).GetChoices, fields(
										fld{"testprotos.AnotherTestMessage.str", 7},
										fld{"testprotos.AnotherTestMessage.int", 8}),
									},
								},
							},
						}},
						"nested messages": {(*MessageDescriptor).GetNestedMessageTypes, []descCase{
							{
								name: "testprotos.AnotherTestMessage.MapField1Entry",
								references: map[string]childCases{
									"fields": {(*MessageDescriptor).GetFields, []descCase{
										{
											name: "testprotos.AnotherTestMessage.MapField1Entry.key",
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, nil},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
										{
											name: "testprotos.AnotherTestMessage.MapField1Entry.value",
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, nil},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
									}},
									"nested messages":   {(*MessageDescriptor).GetNestedMessageTypes, nil},
									"nested enums":      {(*MessageDescriptor).GetNestedEnumTypes, nil},
									"nested extensions": {(*MessageDescriptor).GetNestedExtensions, nil},
									"one ofs":           {(*MessageDescriptor).GetOneOfs, nil},
								},
							},
							{
								name: "testprotos.AnotherTestMessage.MapField2Entry",
								references: map[string]childCases{
									"fields": {(*MessageDescriptor).GetFields, []descCase{
										{
											name: "testprotos.AnotherTestMessage.MapField2Entry.key",
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, nil},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
										{
											name: "testprotos.AnotherTestMessage.MapField2Entry.value",
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, nil},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
									}},
									"nested messages":   {(*MessageDescriptor).GetNestedMessageTypes, nil},
									"nested enums":      {(*MessageDescriptor).GetNestedEnumTypes, nil},
									"nested extensions": {(*MessageDescriptor).GetNestedExtensions, nil},
									"one ofs":           {(*MessageDescriptor).GetOneOfs, nil},
								},
							},
							{
								name: "testprotos.AnotherTestMessage.MapField3Entry",
								references: map[string]childCases{
									"fields": {(*MessageDescriptor).GetFields, []descCase{
										{
											name: "testprotos.AnotherTestMessage.MapField3Entry.key",
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, nil},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
										{
											name: "testprotos.AnotherTestMessage.MapField3Entry.value",
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, nil},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
									}},
									"nested messages":   {(*MessageDescriptor).GetNestedMessageTypes, nil},
									"nested enums":      {(*MessageDescriptor).GetNestedEnumTypes, nil},
									"nested extensions": {(*MessageDescriptor).GetNestedExtensions, nil},
									"one ofs":           {(*MessageDescriptor).GetOneOfs, nil},
								},
							},
							{
								name: "testprotos.AnotherTestMessage.MapField4Entry",
								references: map[string]childCases{
									"fields": {(*MessageDescriptor).GetFields, []descCase{
										{
											name: "testprotos.AnotherTestMessage.MapField4Entry.key",
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, nil},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
										{
											name: "testprotos.AnotherTestMessage.MapField4Entry.value",
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.AnotherTestMessage")},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
									}},
									"nested messages":   {(*MessageDescriptor).GetNestedMessageTypes, nil},
									"nested enums":      {(*MessageDescriptor).GetNestedEnumTypes, nil},
									"nested extensions": {(*MessageDescriptor).GetNestedExtensions, nil},
									"one ofs":           {(*MessageDescriptor).GetOneOfs, nil},
								},
							},
							{
								name: "testprotos.AnotherTestMessage.RockNRoll",
								references: map[string]childCases{
									"fields": {(*MessageDescriptor).GetFields, []descCase{
										{
											name: "testprotos.AnotherTestMessage.RockNRoll.beatles",
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, nil},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
										{
											name: "testprotos.AnotherTestMessage.RockNRoll.stones",
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, nil},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
										{
											name: "testprotos.AnotherTestMessage.RockNRoll.doors",
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, nil},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
									}},
									"nested messages":   {(*MessageDescriptor).GetNestedMessageTypes, nil},
									"nested enums":      {(*MessageDescriptor).GetNestedEnumTypes, nil},
									"nested extensions": {(*MessageDescriptor).GetNestedExtensions, nil},
									"one ofs":           {(*MessageDescriptor).GetOneOfs, nil},
								},
							},
							{
								name: "testprotos.AnotherTestMessage.WithOptions",
								references: map[string]childCases{
									"fields":            {(*MessageDescriptor).GetFields, nil},
									"nested messages":   {(*MessageDescriptor).GetNestedMessageTypes, nil},
									"nested enums":      {(*MessageDescriptor).GetNestedEnumTypes, nil},
									"nested extensions": {(*MessageDescriptor).GetNestedExtensions, nil},
									"one ofs":           {(*MessageDescriptor).GetOneOfs, nil},
								},
							},
						}},
						"nested enums":      {(*MessageDescriptor).GetNestedEnumTypes, nil},
						"nested extensions": {(*MessageDescriptor).GetNestedExtensions, nil},
					},
				},
			}},
			"enums":    {(*FileDescriptor).GetEnumTypes, nil},
			"services": {(*FileDescriptor).GetServices, nil},
			"extensions": {(*FileDescriptor).GetExtensions, []descCase{
				{
					name:   "testprotos.xtm",
					number: 100,
					references: map[string]childCases{
						"owner":        {(*FieldDescriptor).GetOwner, refs("testprotos.AnotherTestMessage")},
						"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.TestMessage")},
						"enum type":    {(*FieldDescriptor).GetEnumType, nil},
						"one of":       {(*FieldDescriptor).GetOneOf, nil},
					},
				},
				{
					name:   "testprotos.xs",
					number: 101,
					references: map[string]childCases{
						"owner":        {(*FieldDescriptor).GetOwner, refs("testprotos.AnotherTestMessage")},
						"message type": {(*FieldDescriptor).GetMessageType, nil},
						"enum type":    {(*FieldDescriptor).GetEnumType, nil},
						"one of":       {(*FieldDescriptor).GetOneOf, nil},
					},
				},
				{
					name:   "testprotos.xi",
					number: 102,
					references: map[string]childCases{
						"owner":        {(*FieldDescriptor).GetOwner, refs("testprotos.AnotherTestMessage")},
						"message type": {(*FieldDescriptor).GetMessageType, nil},
						"enum type":    {(*FieldDescriptor).GetEnumType, nil},
						"one of":       {(*FieldDescriptor).GetOneOf, nil},
					},
				},
				{
					name:   "testprotos.xui",
					number: 103,
					references: map[string]childCases{
						"owner":        {(*FieldDescriptor).GetOwner, refs("testprotos.AnotherTestMessage")},
						"message type": {(*FieldDescriptor).GetMessageType, nil},
						"enum type":    {(*FieldDescriptor).GetEnumType, nil},
						"one of":       {(*FieldDescriptor).GetOneOf, nil},
					},
				},
			}},
		},
	})
}

func TestOneOfDescriptors(t *testing.T) {
	fd, err := LoadFileDescriptor("desc_test2.proto")
	testutil.Ok(t, err)
	md, err := LoadMessageDescriptor("testprotos.Frobnitz")
	testutil.Ok(t, err)
	checkDescriptor(t, "message", 0, md, fd, fd, descCase{
		name: "testprotos.Frobnitz",
		references: map[string]childCases{
			"fields": {(*MessageDescriptor).GetFields, []descCase{
				{
					name: "testprotos.Frobnitz.a",
					references: map[string]childCases{
						"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.TestMessage")},
						"enum type":    {(*FieldDescriptor).GetEnumType, nil},
						"one of":       {(*FieldDescriptor).GetOneOf, nil},
					},
				},
				{
					name: "testprotos.Frobnitz.b",
					references: map[string]childCases{
						"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.AnotherTestMessage")},
						"enum type":    {(*FieldDescriptor).GetEnumType, nil},
						"one of":       {(*FieldDescriptor).GetOneOf, nil},
					},
				},
				{
					name: "testprotos.Frobnitz.c1",
					references: map[string]childCases{
						"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.TestMessage.NestedMessage")},
						"enum type":    {(*FieldDescriptor).GetEnumType, nil},
						"one of":       {(*FieldDescriptor).GetOneOf, refs("testprotos.Frobnitz.abc")},
					},
				},
				{
					name: "testprotos.Frobnitz.c2",
					references: map[string]childCases{
						"message type": {(*FieldDescriptor).GetMessageType, nil},
						"enum type":    {(*FieldDescriptor).GetEnumType, refs("testprotos.TestMessage.NestedEnum")},
						"one of":       {(*FieldDescriptor).GetOneOf, refs("testprotos.Frobnitz.abc")},
					},
				},
				{
					name: "testprotos.Frobnitz.d",
					references: map[string]childCases{
						"message type": {(*FieldDescriptor).GetMessageType, refs("testprotos.TestMessage.NestedMessage")},
						"enum type":    {(*FieldDescriptor).GetEnumType, nil},
						"one of":       {(*FieldDescriptor).GetOneOf, nil},
					},
				},
				{
					name: "testprotos.Frobnitz.e",
					references: map[string]childCases{
						"message type": {(*FieldDescriptor).GetMessageType, nil},
						"enum type":    {(*FieldDescriptor).GetEnumType, refs("testprotos.TestMessage.NestedEnum")},
						"one of":       {(*FieldDescriptor).GetOneOf, nil},
					},
				},
				{
					name: "testprotos.Frobnitz.f",
					references: map[string]childCases{
						"message type": {(*FieldDescriptor).GetMessageType, nil},
						"enum type":    {(*FieldDescriptor).GetEnumType, nil},
						"one of":       {(*FieldDescriptor).GetOneOf, nil},
					},
				},
				{
					name: "testprotos.Frobnitz.g1",
					references: map[string]childCases{
						"message type": {(*FieldDescriptor).GetMessageType, nil},
						"enum type":    {(*FieldDescriptor).GetEnumType, nil},
						"one of":       {(*FieldDescriptor).GetOneOf, refs("testprotos.Frobnitz.def")},
					},
				},
				{
					name: "testprotos.Frobnitz.g2",
					references: map[string]childCases{
						"message type": {(*FieldDescriptor).GetMessageType, nil},
						"enum type":    {(*FieldDescriptor).GetEnumType, nil},
						"one of":       {(*FieldDescriptor).GetOneOf, refs("testprotos.Frobnitz.def")},
					},
				},
				{
					name: "testprotos.Frobnitz.g3",
					references: map[string]childCases{
						"message type": {(*FieldDescriptor).GetMessageType, nil},
						"enum type":    {(*FieldDescriptor).GetEnumType, nil},
						"one of":       {(*FieldDescriptor).GetOneOf, refs("testprotos.Frobnitz.def")},
					},
				},
			}},
			"nested messages":   {(*MessageDescriptor).GetNestedMessageTypes, nil},
			"nested enums":      {(*MessageDescriptor).GetNestedEnumTypes, nil},
			"nested extensions": {(*MessageDescriptor).GetNestedExtensions, nil},
			"one ofs": {(*MessageDescriptor).GetOneOfs, []descCase{
				{
					name:       "testprotos.Frobnitz.abc",
					skipParent: true,
					references: map[string]childCases{
						"fields": {(*OneOfDescriptor).GetChoices, fields(
							fld{"testprotos.Frobnitz.c1", 3},
							fld{"testprotos.Frobnitz.c2", 4}),
						},
					},
				},
				{
					name:       "testprotos.Frobnitz.def",
					skipParent: true,
					references: map[string]childCases{
						"fields": {(*OneOfDescriptor).GetChoices, fields(
							fld{"testprotos.Frobnitz.g1", 8},
							fld{"testprotos.Frobnitz.g2", 9},
							fld{"testprotos.Frobnitz.g3", 10}),
						},
					},
				},
			}},
		},
	})
}

func TestMessageDescriptorFindField(t *testing.T) {
	md, err := LoadMessageDescriptor("testprotos.Frobnitz")
	testutil.Ok(t, err)
	for _, fd := range md.GetFields() {
		found := md.FindFieldByName(fd.GetName())
		testutil.Eq(t, fd, found)
		found = md.FindFieldByNumber(fd.GetNumber())
		testutil.Eq(t, fd, found)
	}
	testutil.Eq(t, (*FieldDescriptor)(nil), md.FindFieldByName("junk name"))
	testutil.Eq(t, (*FieldDescriptor)(nil), md.FindFieldByNumber(99999))
}

func TestEnumDescriptorFindValue(t *testing.T) {
	fd, err := LoadFileDescriptor("desc_test_defaults.proto")
	testutil.Ok(t, err)
	ed, ok := fd.FindSymbol("testprotos.Number").(*EnumDescriptor)
	testutil.Eq(t, true, ok)
	lastNumber := int32(-1)
	for _, vd := range ed.GetValues() {
		found := ed.FindValueByName(vd.GetName())
		testutil.Eq(t, vd, found)
		found = ed.FindValueByNumber(vd.GetNumber())
		if lastNumber == vd.GetNumber() {
			// found value will be the first one with the given number, not this one
			testutil.Eq(t, false, vd == found)
		} else {
			testutil.Eq(t, vd, found)
			lastNumber = vd.GetNumber()
		}
	}
	testutil.Eq(t, (*EnumValueDescriptor)(nil), ed.FindValueByName("junk name"))
	testutil.Eq(t, (*EnumValueDescriptor)(nil), ed.FindValueByNumber(99999))
}

func TestServiceDescriptors(t *testing.T) {
	fd, err := LoadFileDescriptor("desc_test_proto3.proto")
	testutil.Ok(t, err)
	sd := fd.FindSymbol("testprotos.TestService").(*ServiceDescriptor)
	// check the descriptor graph for this service and its descendants
	checkDescriptor(t, "service", 0, sd, fd, fd, descCase{
		name: "testprotos.TestService",
		references: map[string]childCases{
			"methods": {(*ServiceDescriptor).GetMethods, []descCase{
				{
					name: "testprotos.TestService.DoSomething",
					references: map[string]childCases{
						"request":  {(*MethodDescriptor).GetInputType, refs("testprotos.TestRequest")},
						"response": {(*MethodDescriptor).GetOutputType, refs("jhump.protoreflect.desc.Bar")},
					},
				},
				{
					name: "testprotos.TestService.DoSomethingElse",
					references: map[string]childCases{
						"request":  {(*MethodDescriptor).GetInputType, refs("testprotos.TestMessage")},
						"response": {(*MethodDescriptor).GetOutputType, refs("testprotos.TestResponse")},
					},
				},
				{
					name: "testprotos.TestService.DoSomethingAgain",
					references: map[string]childCases{
						"request":  {(*MethodDescriptor).GetInputType, refs("jhump.protoreflect.desc.Bar")},
						"response": {(*MethodDescriptor).GetOutputType, refs("testprotos.AnotherTestMessage")},
					},
				},
				{
					name: "testprotos.TestService.DoSomethingForever",
					references: map[string]childCases{
						"request":  {(*MethodDescriptor).GetInputType, refs("testprotos.TestRequest")},
						"response": {(*MethodDescriptor).GetOutputType, refs("testprotos.TestResponse")},
					},
				},
			}},
		},
	})
	// now verify that FindMethodByName works correctly
	for _, md := range sd.GetMethods() {
		found := sd.FindMethodByName(md.GetName())
		testutil.Eq(t, md, found)
	}
	testutil.Eq(t, (*MethodDescriptor)(nil), sd.FindMethodByName("junk name"))
}

type descCase struct {
	name       string
	number     int32
	skipParent bool
	references map[string]childCases
}

type childCases struct {
	query interface{}
	cases []descCase
}

func refs(names ...string) []descCase {
	r := make([]descCase, len(names))
	for i, n := range names {
		r[i] = descCase{name: n, skipParent: true}
	}
	return r
}

func children(names ...string) []descCase {
	ch := make([]descCase, len(names))
	for i, n := range names {
		ch[i] = descCase{name: n}
	}
	return ch
}

type fld struct {
	name   string
	number int32
}

func fields(flds ...fld) []descCase {
	f := make([]descCase, len(flds))
	for i, field := range flds {
		f[i] = descCase{name: field.name, number: field.number, skipParent: true}
	}
	return f
}

func checkDescriptor(t *testing.T, caseName string, num int32, d Descriptor, parent Descriptor, fd *FileDescriptor, c descCase) {
	// name and fully-qualified name
	testutil.Eq(t, c.name, d.GetFullyQualifiedName(), caseName)
	if _, ok := d.(*FileDescriptor); ok {
		testutil.Eq(t, c.name, d.GetName(), caseName)
	} else {
		pos := strings.LastIndex(c.name, ".")
		n := c.name
		if pos >= 0 {
			n = c.name[pos+1:]
		}
		testutil.Eq(t, n, d.GetName(), caseName)
		// check that this object matches the canonical one returned by file descriptor
		testutil.Eq(t, d, d.GetFile().FindSymbol(d.GetFullyQualifiedName()), caseName)
	}

	// number
	switch d := d.(type) {
	case (*FieldDescriptor):
		n := num + 1
		if c.number != 0 {
			n = c.number
		}
		testutil.Eq(t, n, d.GetNumber(), caseName)
	case (*EnumValueDescriptor):
		n := num + 1
		if c.number != 0 {
			n = c.number
		}
		testutil.Eq(t, n, d.GetNumber(), caseName)
	default:
		if c.number != 0 {
			panic(fmt.Sprintf("%s: number should only be specified by fields and enum values! numnber = %d, desc = %v", caseName, c.number, d))
		}
	}

	// parent and file
	if !c.skipParent {
		testutil.Eq(t, parent, d.GetParent(), caseName)
		testutil.Eq(t, fd, d.GetFile(), caseName)
	}

	// comment
	if fd.GetName() == "desc_test1.proto" && d.GetName() != "desc_test1.proto" {
		expectedComment := "Comment for " + d.GetName()
		if msg, ok := d.(*MessageDescriptor); ok && msg.IsMapEntry() {
			// There are no comments on synthetic map-entry messages.
			expectedComment = ""
		} else if field, ok := d.(*FieldDescriptor); ok {
			if field.GetOwner().IsMapEntry() || field.GetType() == dpb.FieldDescriptorProto_TYPE_GROUP {
				// There are no comments for fields of synthetic map-entry messages either.
				// And comments for group fields end up on the synthetic message, not the field.
				expectedComment = ""
			}
		}
		testutil.Eq(t, expectedComment, strings.TrimSpace(d.GetSourceInfo().GetLeadingComments()), caseName)
	}

	// references
	for name, cases := range c.references {
		caseName := fmt.Sprintf("%s>%s", caseName, name)
		children := runQuery(d, cases.query)
		if testutil.Eq(t, len(cases.cases), len(children), caseName+" length") {
			for i, childCase := range cases.cases {
				caseName := fmt.Sprintf("%s[%d]", caseName, i)
				checkDescriptor(t, caseName, int32(i), children[i], d, fd, childCase)
			}
		}
	}
}

func runQuery(d Descriptor, query interface{}) []Descriptor {
	r := reflect.ValueOf(query).Call([]reflect.Value{reflect.ValueOf(d)})[0]
	if r.Kind() == reflect.Slice {
		ret := make([]Descriptor, r.Len())
		for i := 0; i < r.Len(); i++ {
			ret[i] = r.Index(i).Interface().(Descriptor)
		}
		return ret
	} else if r.IsNil() {
		return []Descriptor{}
	} else {
		return []Descriptor{r.Interface().(Descriptor)}
	}
}

func TestFileDescriptorDeps(t *testing.T) {
	// tests accessors for public and weak dependencies
	fd1 := createDesc(t, &dpb.FileDescriptorProto{Name: proto.String("a")})
	fd2 := createDesc(t, &dpb.FileDescriptorProto{Name: proto.String("b")})
	fd3 := createDesc(t, &dpb.FileDescriptorProto{Name: proto.String("c")})
	fd4 := createDesc(t, &dpb.FileDescriptorProto{Name: proto.String("d")})
	fd5 := createDesc(t, &dpb.FileDescriptorProto{Name: proto.String("e")})
	fd := createDesc(t, &dpb.FileDescriptorProto{
		Name:             proto.String("f"),
		Dependency:       []string{"a", "b", "c", "d", "e"},
		PublicDependency: []int32{1, 3},
		WeakDependency:   []int32{2, 4},
	}, fd1, fd2, fd3, fd4, fd5)

	deps := fd.GetDependencies()
	testutil.Eq(t, 5, len(deps))
	testutil.Eq(t, fd1, deps[0])
	testutil.Eq(t, fd2, deps[1])
	testutil.Eq(t, fd3, deps[2])
	testutil.Eq(t, fd4, deps[3])
	testutil.Eq(t, fd5, deps[4])

	deps = fd.GetPublicDependencies()
	testutil.Eq(t, 2, len(deps))
	testutil.Eq(t, fd2, deps[0])
	testutil.Eq(t, fd4, deps[1])

	deps = fd.GetWeakDependencies()
	testutil.Eq(t, 2, len(deps))
	testutil.Eq(t, fd3, deps[0])
	testutil.Eq(t, fd5, deps[1])

	// Now try on a simple descriptor emitted by protoc
	fd6, err := LoadFileDescriptor("nopkg/desc_test_nopkg.proto")
	testutil.Ok(t, err)
	fd7, err := LoadFileDescriptor("nopkg/desc_test_nopkg_new.proto")
	testutil.Ok(t, err)
	deps = fd6.GetPublicDependencies()
	testutil.Eq(t, 1, len(deps))
	testutil.Eq(t, fd7, deps[0])
}

func createDesc(t *testing.T, fd *dpb.FileDescriptorProto, deps ...*FileDescriptor) *FileDescriptor {
	desc, err := CreateFileDescriptor(fd, deps...)
	testutil.Ok(t, err)
	return desc
}

func TestLoadFileDescriptor(t *testing.T) {
	fd, err := LoadFileDescriptor("desc_test1.proto")
	testutil.Ok(t, err)
	// some very shallow tests (we have more detailed ones in other test cases)
	testutil.Eq(t, "desc_test1.proto", fd.GetName())
	testutil.Eq(t, "desc_test1.proto", fd.GetFullyQualifiedName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
}

func TestLoadMessageDescriptor(t *testing.T) {
	// loading enclosed messages should return the same descriptor
	// and have a reference to the same file descriptor
	md, err := LoadMessageDescriptor("testprotos.TestMessage")
	testutil.Ok(t, err)
	testutil.Eq(t, "TestMessage", md.GetName())
	testutil.Eq(t, "testprotos.TestMessage", md.GetFullyQualifiedName())
	fd := md.GetFile()
	testutil.Eq(t, "desc_test1.proto", fd.GetName())
	testutil.Eq(t, fd, md.GetParent())

	md2, err := LoadMessageDescriptorForMessage((*testprotos.TestMessage)(nil))
	testutil.Ok(t, err)
	testutil.Eq(t, md, md2)

	md3, err := LoadMessageDescriptorForType(reflect.TypeOf((*testprotos.TestMessage)(nil)))
	testutil.Ok(t, err)
	testutil.Eq(t, md, md3)
}

func TestLoadEnumDescriptor(t *testing.T) {
	ed, err := LoadEnumDescriptorForEnum(testprotos.TestMessage_NestedMessage_AnotherNestedMessage_YetAnotherNestedMessage_DeeplyNestedEnum(0))
	testutil.Ok(t, err)
	testutil.Eq(t, "DeeplyNestedEnum", ed.GetName())
	testutil.Eq(t, "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum", ed.GetFullyQualifiedName())
	fd := ed.GetFile()
	testutil.Eq(t, "desc_test1.proto", fd.GetName())
	ofd, err := LoadFileDescriptor("desc_test1.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, ofd, fd)

	ed2, err := LoadEnumDescriptorForEnum((*testprotos.TestEnum)(nil)) // pointer type for interface
	testutil.Ok(t, err)
	testutil.Eq(t, "TestEnum", ed2.GetName())
	testutil.Eq(t, "testprotos.TestEnum", ed2.GetFullyQualifiedName())
	fd = ed2.GetFile()
	testutil.Eq(t, "desc_test_field_types.proto", fd.GetName())
	ofd, err = LoadFileDescriptor("desc_test_field_types.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, ofd, fd)
	testutil.Eq(t, fd, ed2.GetParent())

	// now use the APIs that take reflect.Type
	ed3, err := LoadEnumDescriptorForType(reflect.TypeOf((*testprotos.TestMessage_NestedMessage_AnotherNestedMessage_YetAnotherNestedMessage_DeeplyNestedEnum)(nil)))
	testutil.Ok(t, err)
	testutil.Eq(t, ed, ed3)

	ed4, err := LoadEnumDescriptorForType(reflect.TypeOf(testprotos.TestEnum_FIRST))
	testutil.Ok(t, err)
	testutil.Eq(t, ed2, ed4)
}

func TestLoadFileDescriptorWithDeps(t *testing.T) {
	// Try one with some imports
	fd, err := LoadFileDescriptor("desc_test2.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, "desc_test2.proto", fd.GetName())
	testutil.Eq(t, "desc_test2.proto", fd.GetFullyQualifiedName())
	testutil.Eq(t, "testprotos", fd.GetPackage())

	deps := fd.GetDependencies()
	testutil.Eq(t, 3, len(deps))
	testutil.Eq(t, "desc_test1.proto", deps[0].GetName())
	testutil.Eq(t, "pkg/desc_test_pkg.proto", deps[1].GetName())
	testutil.Eq(t, "nopkg/desc_test_nopkg.proto", deps[2].GetName())

	// loading the dependencies yields same descriptor objects
	fd, err = LoadFileDescriptor("desc_test1.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, deps[0], fd)
	fd, err = LoadFileDescriptor("pkg/desc_test_pkg.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, deps[1], fd)
	fd, err = LoadFileDescriptor("nopkg/desc_test_nopkg.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, deps[2], fd)
}

func TestLoadFileDescriptorForWellKnownProtos(t *testing.T) {
	wellKnownProtos := map[string][]string{
		"google/protobuf/any.proto":             {"google.protobuf.Any"},
		"google/protobuf/api.proto":             {"google.protobuf.Api", "google.protobuf.Method", "google.protobuf.Mixin"},
		"google/protobuf/descriptor.proto":      {"google.protobuf.FileDescriptorSet", "google.protobuf.DescriptorProto"},
		"google/protobuf/duration.proto":        {"google.protobuf.Duration"},
		"google/protobuf/empty.proto":           {"google.protobuf.Empty"},
		"google/protobuf/field_mask.proto":      {"google.protobuf.FieldMask"},
		"google/protobuf/source_context.proto":  {"google.protobuf.SourceContext"},
		"google/protobuf/struct.proto":          {"google.protobuf.Struct", "google.protobuf.Value", "google.protobuf.NullValue"},
		"google/protobuf/timestamp.proto":       {"google.protobuf.Timestamp"},
		"google/protobuf/type.proto":            {"google.protobuf.Type", "google.protobuf.Field", "google.protobuf.Syntax"},
		"google/protobuf/wrappers.proto":        {"google.protobuf.DoubleValue", "google.protobuf.Int32Value", "google.protobuf.StringValue"},
		"google/protobuf/compiler/plugin.proto": {"google.protobuf.compiler.CodeGeneratorRequest"},
	}

	for file, types := range wellKnownProtos {
		fd, err := LoadFileDescriptor(file)
		testutil.Ok(t, err)
		testutil.Eq(t, file, fd.GetName())
		for _, typ := range types {
			d := fd.FindSymbol(typ)
			testutil.Require(t, d != nil)
			d2 := fd.FindSymbol("." + typ)
			testutil.Eq(t, d, d2)
		}

		// also try loading via alternate name
		file = internal.StdFileAliases[file]
		if file == "" {
			// not a file that has a known alternate, so nothing else to check...
			continue
		}
		fd, err = LoadFileDescriptor(file)
		testutil.Ok(t, err)
		testutil.Eq(t, file, fd.GetName())
		for _, typ := range types {
			d := fd.FindSymbol(typ)
			testutil.Require(t, d != nil)
			d2 := fd.FindSymbol("." + typ)
			testutil.Eq(t, d, d2)
		}
	}
}

func TestDefaultValues(t *testing.T) {
	fd, err := LoadFileDescriptor("desc_test_defaults.proto")
	testutil.Ok(t, err)

	testCases := []struct {
		message, field string
		defaultVal     interface{}
	}{
		{"testprotos.PrimitiveDefaults", "fl32", float32(3.14159)},
		{"testprotos.PrimitiveDefaults", "fl64", 3.14159},
		{"testprotos.PrimitiveDefaults", "fl32d", float32(6.022140857e23)},
		{"testprotos.PrimitiveDefaults", "fl64d", 6.022140857e23},
		{"testprotos.PrimitiveDefaults", "fl32inf", float32(math.Inf(1))},
		{"testprotos.PrimitiveDefaults", "fl64inf", math.Inf(1)},
		{"testprotos.PrimitiveDefaults", "fl32negInf", float32(math.Inf(-1))},
		{"testprotos.PrimitiveDefaults", "fl64negInf", math.Inf(-1)},
		{"testprotos.PrimitiveDefaults", "fl32nan", float32(math.NaN())},
		{"testprotos.PrimitiveDefaults", "fl64nan", math.NaN()},
		{"testprotos.PrimitiveDefaults", "bl1", true},
		{"testprotos.PrimitiveDefaults", "bl2", false},
		{"testprotos.PrimitiveDefaults", "i32", int32(10101)},
		{"testprotos.PrimitiveDefaults", "i32n", int32(-10101)},
		{"testprotos.PrimitiveDefaults", "i32x", int32(0x20202)},
		{"testprotos.PrimitiveDefaults", "i32xn", int32(-0x20202)},
		{"testprotos.PrimitiveDefaults", "i64", int64(10101)},
		{"testprotos.PrimitiveDefaults", "i64n", int64(-10101)},
		{"testprotos.PrimitiveDefaults", "i64x", int64(0x20202)},
		{"testprotos.PrimitiveDefaults", "i64xn", int64(-0x20202)},
		{"testprotos.PrimitiveDefaults", "i32s", int32(10101)},
		{"testprotos.PrimitiveDefaults", "i32sn", int32(-10101)},
		{"testprotos.PrimitiveDefaults", "i32sx", int32(0x20202)},
		{"testprotos.PrimitiveDefaults", "i32sxn", int32(-0x20202)},
		{"testprotos.PrimitiveDefaults", "i64s", int64(10101)},
		{"testprotos.PrimitiveDefaults", "i64sn", int64(-10101)},
		{"testprotos.PrimitiveDefaults", "i64sx", int64(0x20202)},
		{"testprotos.PrimitiveDefaults", "i64sxn", int64(-0x20202)},
		{"testprotos.PrimitiveDefaults", "i32f", int32(10101)},
		{"testprotos.PrimitiveDefaults", "i32fn", int32(-10101)},
		{"testprotos.PrimitiveDefaults", "i32fx", int32(0x20202)},
		{"testprotos.PrimitiveDefaults", "i32fxn", int32(-0x20202)},
		{"testprotos.PrimitiveDefaults", "i64f", int64(10101)},
		{"testprotos.PrimitiveDefaults", "i64fn", int64(-10101)},
		{"testprotos.PrimitiveDefaults", "i64fx", int64(0x20202)},
		{"testprotos.PrimitiveDefaults", "i64fxn", int64(-0x20202)},
		{"testprotos.PrimitiveDefaults", "u32", uint32(10101)},
		{"testprotos.PrimitiveDefaults", "u32x", uint32(0x20202)},
		{"testprotos.PrimitiveDefaults", "u64", uint64(10101)},
		{"testprotos.PrimitiveDefaults", "u64x", uint64(0x20202)},
		{"testprotos.PrimitiveDefaults", "u32f", uint32(10101)},
		{"testprotos.PrimitiveDefaults", "u32fx", uint32(0x20202)},
		{"testprotos.PrimitiveDefaults", "u64f", uint64(10101)},
		{"testprotos.PrimitiveDefaults", "u64fx", uint64(0x20202)},

		{"testprotos.StringAndBytesDefaults", "dq", "this is a string with \"nested quotes\""},
		{"testprotos.StringAndBytesDefaults", "sq", "this is a string with \"nested quotes\""},
		{"testprotos.StringAndBytesDefaults", "escaped_bytes", []byte("\000\001\a\b\f\n\r\t\v\\'\"\xfe")},
		{"testprotos.StringAndBytesDefaults", "utf8_string", "\341\210\264"},
		{"testprotos.StringAndBytesDefaults", "string_with_zero", "hel\000lo"},
		{"testprotos.StringAndBytesDefaults", "bytes_with_zero", []byte("wor\000ld")},

		{"testprotos.EnumDefaults", "red", int32(0)},
		{"testprotos.EnumDefaults", "green", int32(1)},
		{"testprotos.EnumDefaults", "blue", int32(2)},
		{"testprotos.EnumDefaults", "zero", int32(0)},
		{"testprotos.EnumDefaults", "zed", int32(0)},
		{"testprotos.EnumDefaults", "one", int32(1)},
		{"testprotos.EnumDefaults", "dos", int32(2)},
	}
	for i, tc := range testCases {
		def := fd.FindMessage(tc.message).FindFieldByName(tc.field).GetDefaultValue()
		testutil.Eq(t, tc.defaultVal, def, "wrong default value for case %d: %s.%s", i, tc.message, tc.field)
	}
}

func TestUnescape(t *testing.T) {
	testCases := []struct {
		in, out string
	}{
		// EOF, bad escapes
		{"\\", "\\"},
		{"\\y", "\\y"},
		// octal escapes
		{"\\0", "\000"},
		{"\\7", "\007"},
		{"\\07", "\007"},
		{"\\77", "\077"},
		{"\\78", "\0078"},
		{"\\077", "\077"},
		{"\\377", "\377"},
		{"\\128", "\0128"},
		{"\\0001", "\0001"},
		{"\\0008", "\0008"},
		// bad octal escape
		{"\\8", "\\8"},
		// hex escapes
		{"\\x0", "\x00"},
		{"\\x7", "\x07"},
		{"\\x07", "\x07"},
		{"\\x77", "\x77"},
		{"\\x7g", "\x07g"},
		{"\\xcc", "\xcc"},
		{"\\xfff", "\xfff"},
		// bad hex escape
		{"\\xg7", "\\xg7"},
		{"\\x", "\\x"},
		// short unicode escapes
		{"\\u0020", "\u0020"},
		{"\\u007e", "\u007e"},
		{"\\u1234", "\u1234"},
		{"\\uffff", "\uffff"},
		// long unicode escapes
		{"\\U00000024", "\U00000024"},
		{"\\U00000076", "\U00000076"},
		{"\\U00001234", "\U00001234"},
		{"\\U0010FFFF", "\U0010FFFF"},
		// bad unicode escapes
		{"\\u12", "\\u12"},
		{"\\ug1232", "\\ug1232"},
		{"\\u", "\\u"},
		{"\\U1234567", "\\U1234567"},
		{"\\U12345678", "\\U12345678"},
		{"\\U0010Fghi", "\\U0010Fghi"},
		{"\\U", "\\U"},
	}
	for _, tc := range testCases {
		for _, p := range []string{"", "prefix"} {
			for _, s := range []string{"", "suffix"} {
				i := p + tc.in + s
				o := p + tc.out + s
				u := unescape(i)
				testutil.Eq(t, o, u, "unescaped %q into %q, but should have been %q", i, u, o)
			}
		}
	}
}

func loadProtoset(path string) (*FileDescriptor, error) {
	var fds dpb.FileDescriptorSet
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bb, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if err = proto.Unmarshal(bb, &fds); err != nil {
		return nil, err
	}
	return CreateFileDescriptorFromSet(&fds)
}

func TestToFileDescriptorSet(t *testing.T) {
	fd, err := LoadFileDescriptor("desc_test2.proto")
	testutil.Ok(t, err, "failed to load descriptor")
	fdset := ToFileDescriptorSet(fd)
	expectedOrder := []string{
		"desc_test1.proto",
		"pkg/desc_test_pkg.proto",
		"nopkg/desc_test_nopkg_new.proto",
		"nopkg/desc_test_nopkg.proto",
		"desc_test2.proto",
	}
	testutil.Eq(t, len(expectedOrder), len(fdset.File), "wrong number of files in set")
	for i, f := range fdset.File {
		testutil.Eq(t, expectedOrder[i], f.GetName(), "wrong file at index %d", i+1)
		expectedFile, err := LoadFileDescriptor(f.GetName())
		testutil.Ok(t, err, "failed to load descriptor for %q", f.GetName())
		testutil.Eq(t, expectedFile.AsFileDescriptorProto(), f)
	}
}
