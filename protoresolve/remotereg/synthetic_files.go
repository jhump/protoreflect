package remotereg

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"
)

// mergeSyntheticFiles merges synthetic files in the given map whose types must
// be defined in the same file.
//
// A type whose definition does not indicate its file is put into its own
// synthetic file. But some types must be defined in the same file as others:
//  1. A nested type must be in the same file as its enclosing type. Otherwise,
//     the enclosing type's name would be interpreted as a package name in the
//     nested type's file, which conflicts with the enclosing type.
//  2. Types that refer to one another (directly or indirectly) must be in the
//     same file, since files cannot have cyclic imports.
//
// So this merges synthetic files in two passes, one for each of the above.
// Each pass can create more work for the other: moving a nested type into its
// enclosing type's file can create a new cycle (for example, if A refers to C
// and C refers to A.B, there is a cycle once A.B is moved into A's file), and
// merging a cycle can change the merged file's package, creating a placeholder
// for an enclosing type (see mergeNestedTypes). So the passes are repeated
// until neither merges any files.
//
// In either pass, a group of files that must be merged may include at most one
// file that is not synthetic (i.e. a file whose name was indicated by the type
// definitions). If it has one, the synthetic files are merged into it. Since
// the files' contents cannot otherwise be reconciled, a nested type whose
// enclosing types are in more than one such file results in an error. A cycle
// that includes more than one such file is left alone and results in an error
// when the files are later registered.
//
// The files map is modified in place. The returned map indicates the files
// that were merged away: its keys are the names of files that were removed
// and its values are the names of the files into which they were merged.
func mergeSyntheticFiles(files map[string]*fileEntry) (map[string]string, error) {
	renamed := map[string]string{}
	for {
		// This repeats internally until it merges nothing more.
		if err := mergeNestedTypes(files, renamed); err != nil {
			return nil, err
		}
		mergedCycles, err := mergeCycles(files, renamed)
		if err != nil {
			return nil, err
		}
		// If the cycle pass merged nothing, the nested pass has nothing new to
		// do. (Every pass that merges files reduces the number of files, so
		// this always terminates.)
		if !mergedCycles {
			break
		}
	}
	// Flatten the map, in case a file was merged into one that was later
	// merged into another.
	for name := range renamed {
		renamed[name] = resolveRenamed(renamed, name)
	}
	return renamed, nil
}

// mergeNestedTypes merges files so that nested types are in the same files as
// their enclosing types.
//
// An enclosing type may not have been fetched; only some of its nested types.
// In that case, a file that has nested types and other types will have a
// placeholder for the enclosing type, which is reliable evidence that the name
// is a message and not a package. (A file's package is computed as the longest
// prefix shared by all of its types, so it can only be too long, never too
// short. So anything below the package is a message.) So placeholders are also
// considered enclosing types. And multiple files that have placeholders with
// the same name are merged, since they would otherwise all define a message
// with that name.
//
// Merging files can change the package computed for the merged file, which can
// create new placeholders. So this repeats until no more files are merged.
func mergeNestedTypes(files map[string]*fileEntry, renamed map[string]string) error {
	for {
		merged, err := mergeNestedTypesOnce(files, renamed)
		if err != nil || !merged {
			return err
		}
	}
}

// mergeNestedTypesOnce performs one pass of mergeNestedTypes, returning true
// if any files were merged.
func mergeNestedTypesOnce(files map[string]*fileEntry, renamed map[string]string) (bool, error) {
	groups := newUnionFind()

	// Find the file for every message, since any of them could be an
	// enclosing type.
	type messageLocation struct {
		fileName    string
		placeholder bool
	}
	messageFiles := map[string]messageLocation{}
	for _, fileName := range slices.Sorted(maps.Keys(files)) {
		files[fileName].forEachMessage(func(name string, placeholder bool) {
			existing, ok := messageFiles[name]
			if !ok {
				messageFiles[name] = messageLocation{fileName: fileName, placeholder: placeholder}
				return
			}
			if existing.placeholder || placeholder {
				// Both files define this message, and at least one is just a
				// placeholder. So they must be the same message. (If both
				// have real definitions, it is a conflict that is reported
				// when the files are registered.)
				groups.union(existing.fileName, fileName)
				if existing.placeholder && !placeholder {
					messageFiles[name] = messageLocation{fileName: fileName}
				}
			}
		})
	}

	for _, fileName := range slices.Sorted(maps.Keys(files)) {
		files[fileName].types.forEachType("", func(typeName string, _ proto.Message) {
			// Find the nearest enclosing message.
			for prefix := typeName; ; {
				pos := strings.LastIndexByte(prefix, '.')
				if pos == -1 {
					return
				}
				prefix = prefix[:pos]
				if enclosing, ok := messageFiles[prefix]; ok {
					groups.union(fileName, enclosing.fileName)
					return
				}
			}
		})
	}

	allGroups := groups.groups()
	for _, group := range allGroups {
		target, realFiles := mergeTarget(files, group)
		if len(realFiles) > 1 {
			return false, fmt.Errorf("nested types and their enclosing types are defined in multiple files: %s", strings.Join(realFiles, ", "))
		}
		if err := mergeFiles(files, target, group, renamed, "nested types and their enclosing types"); err != nil {
			return false, err
		}
	}
	rewriteImports(files, renamed)
	return len(allGroups) > 0, nil
}

// mergeCycles finds cycles among the files and merges each one into a single
// file. It finds strongly connected components in the import graph of the
// files and merges each component that includes synthetic files.
//
// Cycles that include more than one file that is not synthetic are left alone.
// They will result in an error when the files are later registered.
//
// It returns true if any files were merged.
func mergeCycles(files map[string]*fileEntry, renamed map[string]string) (bool, error) {
	finder := sccFinder{
		files: files,
		index: map[string]int{},
		low:   map[string]int{},
		onStk: map[string]bool{},
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if _, visited := finder.index[name]; !visited {
			finder.visit(name)
		}
	}

	merged := false
	for _, component := range finder.components {
		if len(component) == 1 {
			continue
		}
		slices.Sort(component)
		target, realFiles := mergeTarget(files, component)
		if len(realFiles) > 1 {
			continue
		}
		if err := mergeFiles(files, target, component, renamed, "types with cyclic references"); err != nil {
			return false, err
		}
		merged = true
	}
	rewriteImports(files, renamed)
	return merged, nil
}

// mergeTarget determines the file into which the given group of files should
// be merged. It also returns the files in the group that are not synthetic.
// If there is exactly one, it is the target. Otherwise, the target is the file
// whose name sorts first. (Callers must not merge the group if there are
// multiple files that are not synthetic.) The given group must be sorted.
func mergeTarget(files map[string]*fileEntry, group []string) (string, []string) {
	var realFiles []string
	for _, fileName := range group {
		if !files[fileName].synthetic {
			realFiles = append(realFiles, fileName)
		}
	}
	if len(realFiles) == 1 {
		return realFiles[0], realFiles
	}
	return group[0], realFiles
}

// mergeFiles merges the given files into the target file, which must be one of
// them. The merged files are removed from the files map and recorded in the
// renamed map. The given description of the files' types is used in an error
// message if the files do not all have the same syntax.
func mergeFiles(files map[string]*fileEntry, target string, fileNames []string, renamed map[string]string, description string) error {
	targetFile := files[target]
	for _, name := range fileNames {
		if files[name].proto3 != targetFile.proto3 {
			return fmt.Errorf("%s have mixed syntax: %s", description, strings.Join(fileNames, ", "))
		}
	}
	for _, name := range fileNames {
		if name == target {
			continue
		}
		file := files[name]
		file.types.forEachType("", func(typeName string, typ proto.Message) {
			targetFile.types.addType(typeName, typ)
		})
		for dep := range file.deps {
			if targetFile.deps == nil {
				targetFile.deps = map[string]struct{}{}
			}
			targetFile.deps[dep] = struct{}{}
		}
		delete(files, name)
		renamed[name] = target
	}
	return nil
}

// rewriteImports updates the imports of the given files to refer to the files
// into which their dependencies were merged.
func rewriteImports(files map[string]*fileEntry, renamed map[string]string) {
	for name, file := range files {
		for dep := range file.deps {
			if _, ok := renamed[dep]; !ok {
				continue
			}
			delete(file.deps, dep)
			if newDep := resolveRenamed(renamed, dep); newDep != name {
				file.deps[newDep] = struct{}{}
			}
		}
		delete(file.deps, name)
	}
}

// resolveRenamed returns the name of the file into which the named file was
// ultimately merged. It returns the given name if it was not merged.
func resolveRenamed(renamed map[string]string, name string) string {
	for {
		newName, ok := renamed[name]
		if !ok {
			return name
		}
		name = newName
	}
}

// unionFind is a disjoint-set data structure for grouping file names.
type unionFind struct {
	parent map[string]string
}

func newUnionFind() *unionFind {
	return &unionFind{parent: map[string]string{}}
}

func (u *unionFind) find(name string) string {
	root := name
	for {
		parent, ok := u.parent[root]
		if !ok || parent == root {
			break
		}
		root = parent
	}
	// path compression
	for name != root {
		next := u.parent[name]
		u.parent[name] = root
		name = next
	}
	return root
}

func (u *unionFind) union(name1, name2 string) {
	root1, root2 := u.find(name1), u.find(name2)
	if root1 == root2 {
		return
	}
	// make the root the one that sorts first, for deterministic results
	if root2 < root1 {
		root1, root2 = root2, root1
	}
	u.parent[root1] = root1
	u.parent[root2] = root1
}

// groups returns the groups with more than one member. Each group is sorted
// and the groups are sorted by their first member.
func (u *unionFind) groups() [][]string {
	byRoot := map[string][]string{}
	for name := range u.parent {
		root := u.find(name)
		byRoot[root] = append(byRoot[root], name)
	}
	var result [][]string
	for _, root := range slices.Sorted(maps.Keys(byRoot)) {
		group := byRoot[root]
		if len(group) > 1 {
			slices.Sort(group)
			result = append(result, group)
		}
	}
	return result
}

// sccFinder uses Tarjan's algorithm to find strongly connected components in
// the import graph of files. The recursion depth is bounded by the number of
// files, which is in turn bounded by Registry.MaxReachableTypes.
type sccFinder struct {
	files      map[string]*fileEntry
	index      map[string]int
	low        map[string]int
	onStk      map[string]bool
	stack      []string
	next       int
	components [][]string
}

func (f *sccFinder) visit(name string) {
	f.index[name] = f.next
	f.low[name] = f.next
	f.next++
	f.stack = append(f.stack, name)
	f.onStk[name] = true

	for _, dep := range slices.Sorted(maps.Keys(f.files[name].deps)) {
		if f.files[dep] == nil {
			// not one of our files (e.g. resolved from the fallback resolver)
			continue
		}
		if _, visited := f.index[dep]; !visited {
			f.visit(dep)
			f.low[name] = min(f.low[name], f.low[dep])
		} else if f.onStk[dep] {
			f.low[name] = min(f.low[name], f.index[dep])
		}
	}

	if f.low[name] != f.index[name] {
		return
	}
	// name is the root of a component; pop it off the stack
	var component []string
	for {
		top := f.stack[len(f.stack)-1]
		f.stack = f.stack[:len(f.stack)-1]
		f.onStk[top] = false
		component = append(component, top)
		if top == name {
			break
		}
	}
	f.components = append(f.components, component)
}
