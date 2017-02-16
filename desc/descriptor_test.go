package desc

import (
	"testing"

	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/proto"

	"github.com/jhump/protoreflect/desc/desc_test"
	"fmt"
	"strings"
	"reflect"
)

func TestFileDescriptorObjectGraph(t *testing.T) {
	// This checks the structure of the descriptor for desc_test1.proto to make sure
	// the "rich descriptor" accurately models everything therein.
	fd, err := CreateFileDescriptorFromSet(desc_test.GetDescriptorSet())
	ok(t, err)
	checkDescriptor(t, "file", 0, fd, nil, fd, descCase{
		name: "desc_test1.proto",
		references: map[string]childCases {
			"messages": { (*FileDescriptor).GetMessageTypes, []descCase{
				{
					name: "desc_test.TestMessage",
					references: map[string]childCases {
						"fields": { (*MessageDescriptor).GetFields, []descCase {
							{
								name: "desc_test.TestMessage.nm",
								references: map[string]childCases {
									"message type": { (*FieldDescriptor).GetMessageType, refs("desc_test.TestMessage.NestedMessage") },
									"enum type": { (*FieldDescriptor).GetEnumType, nil },
									"one of": { (*FieldDescriptor).GetOneOf, nil },
								},
							},
							{
								name: "desc_test.TestMessage.anm",
								references: map[string]childCases {
									"message type": { (*FieldDescriptor).GetMessageType, refs("desc_test.TestMessage.NestedMessage.AnotherNestedMessage") },
									"enum type": { (*FieldDescriptor).GetEnumType, nil },
									"one of": { (*FieldDescriptor).GetOneOf, nil },
								},
							},
							{
								name: "desc_test.TestMessage.yanm",
								references: map[string]childCases {
									"message type": { (*FieldDescriptor).GetMessageType, refs("desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage") },
									"enum type": { (*FieldDescriptor).GetEnumType, nil },
									"one of": { (*FieldDescriptor).GetOneOf, nil },
								},
							},
							{
								name: "desc_test.TestMessage.ne",
								references: map[string]childCases {
									"message type": { (*FieldDescriptor).GetMessageType, nil },
									"enum type": { (*FieldDescriptor).GetEnumType, refs("desc_test.TestMessage.NestedEnum") },
									"one of": { (*FieldDescriptor).GetOneOf, nil },
								},
							},
						}},
						// this rabbit hole goes pretty deep...
						"nested messages": { (*MessageDescriptor).GetNestedMessageTypes, []descCase{
							{
								name: "desc_test.TestMessage.NestedMessage",
								references: map[string]childCases {
									"fields": { (*MessageDescriptor).GetFields, []descCase{
										{
											name: "desc_test.TestMessage.NestedMessage.anm",
											references: map[string]childCases {
												"message type": { (*FieldDescriptor).GetMessageType, refs("desc_test.TestMessage.NestedMessage.AnotherNestedMessage") },
												"enum type": { (*FieldDescriptor).GetEnumType, nil },
												"one of": { (*FieldDescriptor).GetOneOf, nil },
											},
										},
										{
											name: "desc_test.TestMessage.NestedMessage.yanm",
											references: map[string]childCases {
												"message type": { (*FieldDescriptor).GetMessageType, refs("desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage") },
												"enum type": { (*FieldDescriptor).GetEnumType, nil },
												"one of": { (*FieldDescriptor).GetOneOf, nil },
											},
										},
									}},
									"nested messages": { (*MessageDescriptor).GetNestedMessageTypes, []descCase{
										{
											name: "desc_test.TestMessage.NestedMessage.AnotherNestedMessage",
											references: map[string]childCases {
												"fields": { (*MessageDescriptor).GetFields, []descCase{
													{
														name: "desc_test.TestMessage.NestedMessage.AnotherNestedMessage.yanm",
														references: map[string]childCases {
															"message type": { (*FieldDescriptor).GetMessageType, refs("desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage") },
															"enum type": { (*FieldDescriptor).GetEnumType, nil },
															"one of": { (*FieldDescriptor).GetOneOf, nil },
														},
													},
												}},
												"nested messages": { (*MessageDescriptor).GetNestedMessageTypes, []descCase{
													{
														name: "desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage",
														references: map[string]childCases {
															"nested fields": { (*MessageDescriptor).GetFields, []descCase{
																{
																	name: "desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.foo",
																	references: map[string]childCases {
																		"message type": { (*FieldDescriptor).GetMessageType, nil },
																		"enum type": { (*FieldDescriptor).GetEnumType, nil },
																		"one of": { (*FieldDescriptor).GetOneOf, nil },
																	},
																},
																{
																	name: "desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.bar",
																	references: map[string]childCases {
																		"message type": { (*FieldDescriptor).GetMessageType, nil },
																		"enum type": { (*FieldDescriptor).GetEnumType, nil },
																		"one of": { (*FieldDescriptor).GetOneOf, nil },
																	},
																},
																{
																	name: "desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.baz",
																	references: map[string]childCases {
																		"message type": { (*FieldDescriptor).GetMessageType, nil },
																		"enum type": { (*FieldDescriptor).GetEnumType, nil },
																		"one of": { (*FieldDescriptor).GetOneOf, nil },
																	},
																},
																{
																	name: "desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.dne",
																	references: map[string]childCases {
																		"message type": { (*FieldDescriptor).GetMessageType, nil },
																		"enum type": { (*FieldDescriptor).GetEnumType, refs("desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum") },
																		"one of": { (*FieldDescriptor).GetOneOf, nil },
																	},
																},
																{
																	name: "desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.anm",
																	references: map[string]childCases {
																		"message type": { (*FieldDescriptor).GetMessageType, refs("desc_test.TestMessage.NestedMessage.AnotherNestedMessage") },
																		"enum type": { (*FieldDescriptor).GetEnumType, nil },
																		"one of": { (*FieldDescriptor).GetOneOf, nil },
																	},
																},
																{
																	name: "desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.nm",
																	references: map[string]childCases {
																		"message type": { (*FieldDescriptor).GetMessageType, refs("desc_test.TestMessage.NestedMessage") },
																		"enum type": { (*FieldDescriptor).GetEnumType, nil },
																		"one of": { (*FieldDescriptor).GetOneOf, nil },
																	},
																},
																{
																	name: "desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.tm",
																	references: map[string]childCases {
																		"message type": { (*FieldDescriptor).GetMessageType, refs("desc_test.TestMessage") },
																		"enum type": { (*FieldDescriptor).GetEnumType, nil },
																		"one of": { (*FieldDescriptor).GetOneOf, nil },
																	},
																},
															}},
															"nested messages": { (*MessageDescriptor).GetNestedMessageTypes, nil },
															"nested enums": { (*MessageDescriptor).GetNestedEnumTypes, []descCase{
																{
																	name: "desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum",
																	references: map[string]childCases {
																		"values": { (*EnumDescriptor).GetValues, children(
																			"desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum.VALUE1",
																			"desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum.VALUE2"),
																		},
																	},

																},
															}},
															"nested extensions": { (*MessageDescriptor).GetNestedExtensions, nil },
															"one ofs": { (*MessageDescriptor).GetOneOfs, nil },
														},
													},
												}},
												"nested enums": { (*MessageDescriptor).GetNestedEnumTypes, nil },
												"nested extensions": { (*MessageDescriptor).GetNestedExtensions, []descCase{
													{
														name: "desc_test.TestMessage.NestedMessage.AnotherNestedMessage.flags",
														number: 200,
														references: map[string]childCases {
															"owner": { (*FieldDescriptor).GetOwner, refs("desc_test.AnotherTestMessage") },
															"message type": { (*FieldDescriptor).GetMessageType, nil },
															"enum type": { (*FieldDescriptor).GetEnumType, nil },
															"one of": { (*FieldDescriptor).GetOneOf, nil },
														},
													},
												}},
												"one ofs": { (*MessageDescriptor).GetOneOfs, nil },
											},
										},
									}},
									"nested enums": { (*MessageDescriptor).GetNestedEnumTypes, nil },
									"nested extensions": { (*MessageDescriptor).GetNestedExtensions, nil },
									"one ofs": { (*MessageDescriptor).GetOneOfs, nil },
								},
							},

						}},
						"nested enums": { (*MessageDescriptor).GetNestedEnumTypes, []descCase{
							{
								name: "desc_test.TestMessage.NestedEnum",
								references: map[string]childCases {
									"values": { (*EnumDescriptor).GetValues, children(
										"desc_test.TestMessage.NestedEnum.VALUE1", "desc_test.TestMessage.NestedEnum.VALUE2"),
									},
								},
							},
						}},
						"nested extensions": { (*MessageDescriptor).GetNestedExtensions, nil },
						"one ofs": { (*MessageDescriptor).GetOneOfs, nil },
					},
				},
				{
					name: "desc_test.AnotherTestMessage",
					references: map[string]childCases {
						"fields": { (*MessageDescriptor).GetFields, []descCase {
							{
								name: "desc_test.AnotherTestMessage.dne",
								references: map[string]childCases {
									"message type": { (*FieldDescriptor).GetMessageType, nil },
									"enum type": { (*FieldDescriptor).GetEnumType, refs("desc_test.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum") },
									"one of": { (*FieldDescriptor).GetOneOf, nil },
								},
							},
						}},
						"nested messages": { (*MessageDescriptor).GetNestedMessageTypes, nil },
						"nested enums": { (*MessageDescriptor).GetNestedEnumTypes, nil },
						"nested extensions": { (*MessageDescriptor).GetNestedExtensions, nil },
						"one ofs": { (*MessageDescriptor).GetOneOfs, nil },
					},
				},
			}},
			"enums": { (*FileDescriptor).GetEnumTypes, nil },
			"services": { (*FileDescriptor).GetServices, nil },
			"extensions": { (*FileDescriptor).GetExtensions, []descCase{
				{
					name: "desc_test.xtm",
					number: 100,
					references: map[string]childCases {
						"owner": { (*FieldDescriptor).GetOwner, refs("desc_test.AnotherTestMessage") },
						"message type": { (*FieldDescriptor).GetMessageType, refs("desc_test.TestMessage") },
						"enum type": { (*FieldDescriptor).GetEnumType, nil },
						"one of": { (*FieldDescriptor).GetOneOf, nil },
					},
				},
				{
					name: "desc_test.xs",
					number: 101,
					references: map[string]childCases {
						"owner": { (*FieldDescriptor).GetOwner, refs("desc_test.AnotherTestMessage") },
						"message type": { (*FieldDescriptor).GetMessageType, nil },
						"enum type": { (*FieldDescriptor).GetEnumType, nil },
						"one of": { (*FieldDescriptor).GetOneOf, nil },
					},
				},
				{
					name: "desc_test.xi",
					number: 102,
					references: map[string]childCases {
						"owner": { (*FieldDescriptor).GetOwner, refs("desc_test.AnotherTestMessage") },
						"message type": { (*FieldDescriptor).GetMessageType, nil },
						"enum type": { (*FieldDescriptor).GetEnumType, nil },
						"one of": { (*FieldDescriptor).GetOneOf, nil },
					},
				},
				{
					name: "desc_test.xui",
					number: 103,
					references: map[string]childCases {
						"owner": { (*FieldDescriptor).GetOwner, refs("desc_test.AnotherTestMessage") },
						"message type": { (*FieldDescriptor).GetMessageType, nil },
						"enum type": { (*FieldDescriptor).GetEnumType, nil },
						"one of": { (*FieldDescriptor).GetOneOf, nil },
					},
				},
			}},
		},
	})
}

func TestOneOfDescriptors(t *testing.T) {
	fd, err := LoadFileDescriptor("desc_test2.proto")
	ok(t, err)
	md, err := LoadMessageDescriptor("desc_test.Frobnitz")
	ok(t, err)
	checkDescriptor(t, "message", 0, md, fd, fd, descCase {
		name: "desc_test.Frobnitz",
		references: map[string]childCases{
			"fields": { (*MessageDescriptor).GetFields, []descCase{
				{
					name: "desc_test.Frobnitz.a",
					references: map[string]childCases{
						"message type": { (*FieldDescriptor).GetMessageType, refs("desc_test.TestMessage") },
						"enum type": { (*FieldDescriptor).GetEnumType, nil },
						"one of": { (*FieldDescriptor).GetOneOf, nil },
					},
				},
				{
					name: "desc_test.Frobnitz.b",
					references: map[string]childCases{
						"message type": { (*FieldDescriptor).GetMessageType, refs("desc_test.AnotherTestMessage") },
						"enum type": { (*FieldDescriptor).GetEnumType, nil },
						"one of": { (*FieldDescriptor).GetOneOf, nil },
					},
				},
				{
					name: "desc_test.Frobnitz.c1",
					references: map[string]childCases{
						"message type": { (*FieldDescriptor).GetMessageType, refs("desc_test.TestMessage.NestedMessage") },
						"enum type": { (*FieldDescriptor).GetEnumType, nil },
						"one of": { (*FieldDescriptor).GetOneOf, refs("desc_test.Frobnitz.abc") },
					},
				},
				{
					name: "desc_test.Frobnitz.c2",
					references: map[string]childCases{
						"message type": { (*FieldDescriptor).GetMessageType, nil },
						"enum type": { (*FieldDescriptor).GetEnumType, refs("desc_test.TestMessage.NestedEnum") },
						"one of": { (*FieldDescriptor).GetOneOf, refs("desc_test.Frobnitz.abc") },
					},
				},
				{
					name: "desc_test.Frobnitz.d",
					references: map[string]childCases{
						"message type": { (*FieldDescriptor).GetMessageType, refs("desc_test.TestMessage.NestedMessage") },
						"enum type": { (*FieldDescriptor).GetEnumType, nil },
						"one of": { (*FieldDescriptor).GetOneOf, nil },
					},
				},
				{
					name: "desc_test.Frobnitz.e",
					references: map[string]childCases{
						"message type": { (*FieldDescriptor).GetMessageType, nil },
						"enum type": { (*FieldDescriptor).GetEnumType, refs("desc_test.TestMessage.NestedEnum") },
						"one of": { (*FieldDescriptor).GetOneOf, nil },
					},
				},
				{
					name: "desc_test.Frobnitz.f",
					references: map[string]childCases{
						"message type": { (*FieldDescriptor).GetMessageType, nil },
						"enum type": { (*FieldDescriptor).GetEnumType, nil },
						"one of": { (*FieldDescriptor).GetOneOf, nil },
					},
				},
				{
					name: "desc_test.Frobnitz.g1",
					references: map[string]childCases{
						"message type": { (*FieldDescriptor).GetMessageType, nil },
						"enum type": { (*FieldDescriptor).GetEnumType, nil },
						"one of": { (*FieldDescriptor).GetOneOf, refs("desc_test.Frobnitz.def") },
					},
				},
				{
					name: "desc_test.Frobnitz.g2",
					references: map[string]childCases{
						"message type": { (*FieldDescriptor).GetMessageType, nil },
						"enum type": { (*FieldDescriptor).GetEnumType, nil },
						"one of": { (*FieldDescriptor).GetOneOf, refs("desc_test.Frobnitz.def") },
					},
				},
				{
					name: "desc_test.Frobnitz.g3",
					references: map[string]childCases{
						"message type": { (*FieldDescriptor).GetMessageType, nil },
						"enum type": { (*FieldDescriptor).GetEnumType, nil },
						"one of": { (*FieldDescriptor).GetOneOf, refs("desc_test.Frobnitz.def") },
					},
				},
			}},
			"nested messages": { (*MessageDescriptor).GetNestedMessageTypes, nil },
			"nested enums": { (*MessageDescriptor).GetNestedEnumTypes, nil },
			"nested extensions": { (*MessageDescriptor).GetNestedExtensions, nil },
			"one ofs": { (*MessageDescriptor).GetOneOfs, []descCase{
				{
					name: "desc_test.Frobnitz.abc",
					skipParent: true,
					references: map[string]childCases{
						"fields": { (*OneOfDescriptor).GetChoices, fields(
							fld{"desc_test.Frobnitz.c1", 3},
							fld{"desc_test.Frobnitz.c2", 4}),
						},
					},
				},
				{
					name: "desc_test.Frobnitz.def",
					skipParent: true,
					references: map[string]childCases{
						"fields": { (*OneOfDescriptor).GetChoices, fields(
							fld{"desc_test.Frobnitz.g1", 8},
							fld{"desc_test.Frobnitz.g2", 9},
							fld{"desc_test.Frobnitz.g3", 10}),
						},
					},
				},
			}},
		},
	})
}

func TestServiceDescriptors(t *testing.T) {
	fd, err := LoadFileDescriptor("desc_test_proto3.proto")
	ok(t, err)
	sd := fd.FindSymbol("desc_test.TestService").(*ServiceDescriptor)
	checkDescriptor(t, "service", 0, sd, fd, fd, descCase{
		name: "desc_test.TestService",
		references: map[string]childCases{
			"methods": { (*ServiceDescriptor).GetMethods, []descCase{
				{
					name: "desc_test.TestService.DoSomething",
					references: map[string]childCases{
						"request": { (*MethodDescriptor).GetInputType, refs("desc_test.TestRequest") },
						"response": { (*MethodDescriptor).GetOutputType, refs("jhump.protoreflect.desc.Bar") },
					},
				},
				{
					name: "desc_test.TestService.DoSomethingElse",
					references: map[string]childCases{
						"request": { (*MethodDescriptor).GetInputType, refs("desc_test.TestMessage") },
						"response": { (*MethodDescriptor).GetOutputType, refs("desc_test.TestResponse") },
					},
				},
				{
					name: "desc_test.TestService.DoSomethingAgain",
					references: map[string]childCases{
						"request": { (*MethodDescriptor).GetInputType, refs("jhump.protoreflect.desc.Bar") },
						"response": { (*MethodDescriptor).GetOutputType, refs("desc_test.AnotherTestMessage") },
					},
				},
				{
					name: "desc_test.TestService.DoSomethingForever",
					references: map[string]childCases{
						"request": { (*MethodDescriptor).GetInputType, refs("desc_test.TestRequest") },
						"response": { (*MethodDescriptor).GetOutputType, refs("desc_test.TestResponse") },
					},
				},
			}},
		},
	})
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
		r[i] = descCase{ name: n, skipParent: true }
	}
	return r
}

func children(names ...string) []descCase {
	ch := make([]descCase, len(names))
	for i, n := range names {
		ch[i] = descCase{ name: n }
	}
	return ch
}

type fld struct {
	name   string
	number int32
}

func fields(flds ... fld) []descCase {
	f := make([]descCase, len(flds))
	for i, field := range flds {
		f[i] = descCase{ name: field.name, number: field.number, skipParent: true }
	}
	return f
}

func checkDescriptor(t *testing.T, caseName string, num int32, d Descriptor, parent Descriptor, fd *FileDescriptor, c descCase) {
	// name and fully-qualified name
	eq(t, c.name, d.GetFullyQualifiedName(), caseName)
	if _, ok := d.(*FileDescriptor); ok {
		eq(t, c.name, d.GetName(), caseName)
	} else {
		pos := strings.LastIndex(c.name, ".")
		n := c.name
		if pos >= 0 {
			n = c.name[pos+1:]
		}
		eq(t, n, d.GetName(), caseName)
		// check that this object matches the canonical one returned by file descriptor
		eq(t, d, d.GetFile().FindSymbol(d.GetFullyQualifiedName()), caseName)
	}

	// number
	switch d := d.(type) {
	case (*FieldDescriptor):
		n := num + 1
		if c.number != 0 {
			n = c.number
		}
		eq(t, n, d.GetNumber(), caseName)
	case (*EnumValueDescriptor):
		n := num + 1
		if c.number != 0 {
			n = c.number
		}
		eq(t, n, d.GetNumber(), caseName)
	default:
		if c.number != 0 {
			panic(fmt.Sprintf("%s: number should only be specified by fields and enum values! numnber = %d, desc = %v", caseName, c.number, d))
		}
	}

	// parent and file
	if !c.skipParent {
		eq(t, parent, d.GetParent(), caseName)
		eq(t, fd, d.GetFile(), caseName)
	}

	// comment
	if fd.GetName() == "desc_test1.proto" && d.GetName() != "desc_test1.proto" {
		eq(t, "Comment for " + d.GetName(), strings.TrimSpace(d.GetSourceInfo().GetLeadingComments()), caseName)
	}

	// references
	for name, cases := range c.references {
		caseName := fmt.Sprintf("%s>%s", caseName, name)
		children := runQuery(d, cases.query)
		if eq(t, len(cases.cases), len(children), caseName + " length") {
			for i, childCase := range cases.cases {
				caseName := fmt.Sprintf("%s[%d]", caseName, i)
				checkDescriptor(t, caseName, int32(i), children[i], d, fd, childCase)
			}
		}
	}
}

func runQuery(d Descriptor, query interface{}) []Descriptor {
	r := reflect.ValueOf(query).Call([]reflect.Value{ reflect.ValueOf(d) })[0]
	if r.Kind() == reflect.Slice {
		ret := make([]Descriptor, r.Len())
		for i := 0; i < r.Len(); i++ {
			ret[i] = r.Index(i).Interface().(Descriptor)
		}
		return ret
	} else if r.IsNil() {
		return []Descriptor{}
	} else {
		return []Descriptor{ r.Interface().(Descriptor) }
	}
}

func TestFileDescriptorDeps(t *testing.T) {
	// tests accessors for public and weak dependencies
	fd1 := createDesc(t, &dpb.FileDescriptorProto{Name: proto.String("a") })
	fd2 := createDesc(t, &dpb.FileDescriptorProto{Name: proto.String("b") })
	fd3 := createDesc(t, &dpb.FileDescriptorProto{Name: proto.String("c") })
	fd4 := createDesc(t, &dpb.FileDescriptorProto{Name: proto.String("d") })
	fd5 := createDesc(t, &dpb.FileDescriptorProto{Name: proto.String("e") })
	fd := createDesc(t, &dpb.FileDescriptorProto{
		Name: proto.String("f"),
		Dependency: []string{"a", "b", "c", "d", "e" },
		PublicDependency: []int32{1, 3 },
		WeakDependency: []int32{2, 4 },
	}, fd1, fd2, fd3, fd4, fd5)

	deps := fd.GetDependencies()
	eq(t, 5, len(deps))
	eq(t, fd1, deps[0])
	eq(t, fd2, deps[1])
	eq(t, fd3, deps[2])
	eq(t, fd4, deps[3])
	eq(t, fd5, deps[4])

	deps = fd.GetPublicDependencies()
	eq(t, 2, len(deps))
	eq(t, fd2, deps[0])
	eq(t, fd4, deps[1])

	deps = fd.GetWeakDependencies()
	eq(t, 2, len(deps))
	eq(t, fd3, deps[0])
	eq(t, fd5, deps[1])

	// Now try on a simple descriptor emitted by protoc
	fd6, err := LoadFileDescriptor("nopkg/desc_test_nopkg.proto")
	ok(t, err)
	fd7, err := LoadFileDescriptor("nopkg/desc_test_nopkg_new.proto")
	ok(t, err)
	deps = fd6.GetPublicDependencies()
	eq(t, 1, len(deps))
	eq(t, fd7, deps[0])
}

func createDesc(t *testing.T, fd *dpb.FileDescriptorProto, deps ...*FileDescriptor) *FileDescriptor {
	desc, err := CreateFileDescriptor(fd, deps...)
	ok(t, err)
	return desc
}

func eq(t *testing.T, expected, actual interface{}, context ...interface{}) bool {
	if expected != actual {
		ctxString := formatContext(context)
		if ctxString == "" {
			t.Errorf("Expecting %v, got %v", expected, actual)
		} else {
			t.Errorf("%s: Expecting %v, got %v", ctxString, expected, actual)
		}
		return false
	}
	return true
}

func ok(t *testing.T, err error, context ...interface{}) {
	if err != nil {
		ctxString := formatContext(context)
		if ctxString == "" {
			t.Fatalf("Unexpected error: %s", err.Error())
		} else {
			t.Fatalf("%s: Unexpected error: %s", ctxString, err.Error())
		}
	}
}

func formatContext(context []interface{}) string {
	if len(context) == 0 {
		return ""
	} else if len(context) == 1 {
		return context[0].(string)
	} else {
		format := context[0].(string)
		return fmt.Sprintf(format, context[1:]...)
	}
}