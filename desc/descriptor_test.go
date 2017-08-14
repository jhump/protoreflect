package desc

import (
	"fmt"
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

	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestFileDescriptorObjectGraph(t *testing.T) {
	// This checks the structure of the descriptor for desc_test1.proto to make sure
	// the "rich descriptor" accurately models everything therein.
	fd, err := CreateFileDescriptorFromSet(testprotos.GetDescriptorSet())
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
											name:   "testprotos.AnotherTestMessage.RockNRoll.beatles",
											number: 7,
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, nil},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
										{
											name:   "testprotos.AnotherTestMessage.RockNRoll.stones",
											number: 8,
											references: map[string]childCases{
												"message type": {(*FieldDescriptor).GetMessageType, nil},
												"enum type":    {(*FieldDescriptor).GetEnumType, nil},
												"one of":       {(*FieldDescriptor).GetOneOf, nil},
											},
										},
										{
											name:   "testprotos.AnotherTestMessage.RockNRoll.doors",
											number: 9,
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
						}},
						"nested enums":      {(*MessageDescriptor).GetNestedEnumTypes, nil},
						"nested extensions": {(*MessageDescriptor).GetNestedExtensions, nil},
						"one ofs":           {(*MessageDescriptor).GetOneOfs, nil},
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
		// Try one with some imports
		fd, err := LoadFileDescriptor(file)
		testutil.Ok(t, err)
		testutil.Eq(t, file, fd.GetName())
		for _, typ := range types {
			d := fd.FindSymbol(typ)
			testutil.Require(t, d != nil)
		}
	}
}
