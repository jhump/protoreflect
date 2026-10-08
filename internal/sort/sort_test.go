package sort

import (
	"slices"
	"strings"
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
	name1 := "file1.proto"
	name2 := "file2.proto"
	name3 := "file3.proto"

	files := []*descriptorpb.FileDescriptorProto{
		{
			Name:       &name1,
			Dependency: []string{"file2.proto"},
		},
		{
			Name:       &name2,
			Dependency: []string{"file3.proto"},
		},
		{
			Name:       &name3,
			Dependency: []string{"file1.proto"},
		},
	}
	original := slices.Clone(files)

	err := SortFiles(files)
	if err == nil || !strings.Contains(err.Error(), "import cycle") {
		t.Errorf("Expected import cycle error, got %v", err)
	}
	if !slices.Equal(original, files) {
		t.Errorf("Files should be unchanged after an error")
	}
}

func TestSortFiles_SelfDependency(t *testing.T) {
	name := "file.proto"

	files := []*descriptorpb.FileDescriptorProto{
		{
			Name:       &name,
			Dependency: []string{"file.proto"},
		},
	}

	// A file that imports itself is a cycle.
	err := SortFiles(files)
	if err == nil || !strings.Contains(err.Error(), "import cycle") {
		t.Errorf("Expected import cycle error, got %v", err)
	}
}

func TestSortFiles_UnchangedOnError(t *testing.T) {
	// Several files can be sorted before the missing import is found.
	names := []string{"a.proto", "b.proto", "c.proto", "d.proto", "e.proto", "f.proto", "g.proto", "h.proto"}
	for range 10 {
		files := make([]*descriptorpb.FileDescriptorProto, len(names))
		for i := range names {
			files[i] = &descriptorpb.FileDescriptorProto{Name: &names[i]}
		}
		// b.proto imports h.proto, which imports a missing file.
		files[1].Dependency = []string{"h.proto"}
		files[7].Dependency = []string{"missing.proto"}
		original := slices.Clone(files)

		if err := SortFiles(files); err == nil {
			t.Fatal("Expected error for missing import")
		}
		if !slices.Equal(original, files) {
			t.Fatal("Files should be unchanged after an error")
		}
	}
}

func TestSortFiles_Deterministic(t *testing.T) {
	names := []string{"a.proto", "b.proto", "c.proto", "d.proto", "e.proto", "f.proto"}
	newFiles := func() []*descriptorpb.FileDescriptorProto {
		files := make([]*descriptorpb.FileDescriptorProto, len(names))
		for i := range names {
			files[i] = &descriptorpb.FileDescriptorProto{Name: &names[i]}
		}
		// f.proto imports a.proto; the rest have no imports.
		files[5].Dependency = []string{"a.proto"}
		return files
	}

	// Independent files keep their order.
	for range 20 {
		files := newFiles()
		if err := SortFiles(files); err != nil {
			t.Fatalf("SortFiles failed: %v", err)
		}
		for i, file := range files {
			if file.GetName() != names[i] {
				t.Fatalf("Expected %s at index %d, got %s", names[i], i, file.GetName())
			}
		}
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
