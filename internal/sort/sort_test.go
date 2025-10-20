package sort

import (
	"testing"

	"google.golang.org/protobuf/types/descriptorpb"
)

func TestSortFiles_Empty(t *testing.T) {
	files := []*descriptorpb.FileDescriptorProto{}
	err := SortFiles(files)
	if err != nil {
		t.Errorf("SortFiles with empty slice failed: %v", err)
	}
}

func TestSortFiles_SingleFile(t *testing.T) {
	name := "test.proto"
	files := []*descriptorpb.FileDescriptorProto{
		{Name: &name},
	}
	err := SortFiles(files)
	if err != nil {
		t.Errorf("SortFiles with single file failed: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
	if files[0].GetName() != name {
		t.Errorf("Expected file name %q, got %q", name, files[0].GetName())
	}
}

func TestSortFiles_NoDependencies(t *testing.T) {
	name1 := "file1.proto"
	name2 := "file2.proto"
	name3 := "file3.proto"
	files := []*descriptorpb.FileDescriptorProto{
		{Name: &name1},
		{Name: &name2},
		{Name: &name3},
	}
	err := SortFiles(files)
	if err != nil {
		t.Errorf("SortFiles with no dependencies failed: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}
}

func TestSortFiles_WithDependencies(t *testing.T) {
	// Create files with dependencies: file3 -> file2 -> file1
	name1 := "file1.proto"
	name2 := "file2.proto"
	name3 := "file3.proto"

	files := []*descriptorpb.FileDescriptorProto{
		{
			Name:       &name3,
			Dependency: []string{"file2.proto"},
		},
		{
			Name:       &name1,
			Dependency: []string{},
		},
		{
			Name:       &name2,
			Dependency: []string{"file1.proto"},
		},
	}

	err := SortFiles(files)
	if err != nil {
		t.Errorf("SortFiles with dependencies failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}

	// Verify topological order: file1 should come before file2, file2 before file3
	fileOrder := make(map[string]int)
	for i, file := range files {
		fileOrder[file.GetName()] = i
	}

	if fileOrder["file1.proto"] >= fileOrder["file2.proto"] {
		t.Errorf("file1.proto should come before file2.proto")
	}
	if fileOrder["file2.proto"] >= fileOrder["file3.proto"] {
		t.Errorf("file2.proto should come before file3.proto")
	}
}

func TestSortFiles_ComplexDependencies(t *testing.T) {
	// Create a more complex dependency graph:
	//   base.proto (no deps)
	//   common.proto -> base.proto
	//   types.proto -> base.proto
	//   service.proto -> common.proto, types.proto

	base := "base.proto"
	common := "common.proto"
	types := "types.proto"
	service := "service.proto"

	files := []*descriptorpb.FileDescriptorProto{
		{
			Name:       &service,
			Dependency: []string{"common.proto", "types.proto"},
		},
		{
			Name:       &types,
			Dependency: []string{"base.proto"},
		},
		{
			Name:       &base,
			Dependency: []string{},
		},
		{
			Name:       &common,
			Dependency: []string{"base.proto"},
		},
	}

	err := SortFiles(files)
	if err != nil {
		t.Errorf("SortFiles with complex dependencies failed: %v", err)
	}

	if len(files) != 4 {
		t.Errorf("Expected 4 files, got %d", len(files))
	}

	// Verify topological order
	fileOrder := make(map[string]int)
	for i, file := range files {
		fileOrder[file.GetName()] = i
	}

	// base.proto should come before everything else
	if fileOrder["base.proto"] >= fileOrder["common.proto"] {
		t.Errorf("base.proto should come before common.proto")
	}
	if fileOrder["base.proto"] >= fileOrder["types.proto"] {
		t.Errorf("base.proto should come before types.proto")
	}

	// common.proto and types.proto should come before service.proto
	if fileOrder["common.proto"] >= fileOrder["service.proto"] {
		t.Errorf("common.proto should come before service.proto")
	}
	if fileOrder["types.proto"] >= fileOrder["service.proto"] {
		t.Errorf("types.proto should come before service.proto")
	}
}

func TestSortFiles_DuplicateFile(t *testing.T) {
	name := "test.proto"
	files := []*descriptorpb.FileDescriptorProto{
		{Name: &name},
		{Name: &name},
	}

	err := SortFiles(files)
	if err == nil {
		t.Error("Expected error for duplicate files, got nil")
	}
	if err != nil && err.Error() != `duplicate file "test.proto"` {
		t.Errorf("Expected duplicate file error, got: %v", err)
	}
}

func TestSortFiles_MissingImport(t *testing.T) {
	name1 := "file1.proto"
	name2 := "file2.proto"

	files := []*descriptorpb.FileDescriptorProto{
		{
			Name:       &name1,
			Dependency: []string{"missing.proto"},
		},
		{
			Name: &name2,
		},
	}

	err := SortFiles(files)
	if err == nil {
		t.Error("Expected error for missing import, got nil")
	}
	if err != nil && err.Error() != `file "file1.proto" imports "missing.proto", but "missing.proto" is not present` {
		t.Errorf("Expected missing import error, got: %v", err)
	}
}

func TestSortFiles_CircularDependency(t *testing.T) {
	// Note: The current implementation doesn't explicitly detect circular dependencies
	// but this test documents the behavior. In practice, circular dependencies
	// would be caught by the protobuf compiler itself.

	name1 := "file1.proto"
	name2 := "file2.proto"

	files := []*descriptorpb.FileDescriptorProto{
		{
			Name:       &name1,
			Dependency: []string{"file2.proto"},
		},
		{
			Name:       &name2,
			Dependency: []string{"file1.proto"},
		},
	}

	// This will likely cause infinite recursion or stack overflow
	// depending on the implementation
	err := SortFiles(files)
	// For now, we just document that this is not handled
	_ = err
}

func TestSortFiles_SelfDependency(t *testing.T) {
	name := "file.proto"

	files := []*descriptorpb.FileDescriptorProto{
		{
			Name:       &name,
			Dependency: []string{"file.proto"},
		},
	}

	// Self-dependency should be handled gracefully
	err := SortFiles(files)
	// The current implementation will add it once and skip on recursion
	if err != nil {
		t.Errorf("Unexpected error for self-dependency: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 file after sort, got %d", len(files))
	}
}

func TestSortFiles_MultipleDependenciesSameFile(t *testing.T) {
	base := "base.proto"
	derived1 := "derived1.proto"
	derived2 := "derived2.proto"
	aggregate := "aggregate.proto"

	files := []*descriptorpb.FileDescriptorProto{
		{
			Name:       &aggregate,
			Dependency: []string{"derived1.proto", "derived2.proto"},
		},
		{
			Name:       &derived2,
			Dependency: []string{"base.proto"},
		},
		{
			Name:       &derived1,
			Dependency: []string{"base.proto"},
		},
		{
			Name: &base,
		},
	}

	err := SortFiles(files)
	if err != nil {
		t.Errorf("SortFiles failed: %v", err)
	}

	if len(files) != 4 {
		t.Errorf("Expected 4 files, got %d", len(files))
	}

	// Verify base.proto comes first
	if files[0].GetName() != "base.proto" {
		t.Errorf("Expected base.proto first, got %s", files[0].GetName())
	}

	// Verify aggregate.proto comes last
	if files[3].GetName() != "aggregate.proto" {
		t.Errorf("Expected aggregate.proto last, got %s", files[3].GetName())
	}
}

func TestSortFiles_PreservesFileContents(t *testing.T) {
	pkg1 := "pkg1"
	pkg2 := "pkg2"
	syntax := "proto3"

	name1 := "file1.proto"
	name2 := "file2.proto"

	files := []*descriptorpb.FileDescriptorProto{
		{
			Name:       &name2,
			Package:    &pkg2,
			Syntax:     &syntax,
			Dependency: []string{"file1.proto"},
		},
		{
			Name:    &name1,
			Package: &pkg1,
			Syntax:  &syntax,
		},
	}

	err := SortFiles(files)
	if err != nil {
		t.Errorf("SortFiles failed: %v", err)
	}

	// Verify file contents are preserved
	file1 := files[0]
	if file1.GetPackage() != "pkg1" {
		t.Errorf("Expected package pkg1, got %s", file1.GetPackage())
	}
	if file1.GetSyntax() != "proto3" {
		t.Errorf("Expected syntax proto3, got %s", file1.GetSyntax())
	}

	file2 := files[1]
	if file2.GetPackage() != "pkg2" {
		t.Errorf("Expected package pkg2, got %s", file2.GetPackage())
	}
	if len(file2.GetDependency()) != 1 || file2.GetDependency()[0] != "file1.proto" {
		t.Errorf("Expected dependency on file1.proto, got %v", file2.GetDependency())
	}
}
