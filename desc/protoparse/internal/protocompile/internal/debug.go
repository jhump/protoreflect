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

//go:build debug

package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/petermattis/goid"
)

// Debug is true if the compiler is being built with the debug tag, which
// enables various debugging features.
const Debug = true

var debugLogLock sync.Mutex

// DebugLog prints debugging information to stderr.
//
// context is optional args for `fmt.Printf` that are printed before
// operation. This is useful for cases where you want to have
// information that identifies a set of operations that are related to appear
// before operation does.
func DebugLog(context []any, operation string, format string, args ...any) {
	// Determine the package and file which called us.
	pc, file, _, _ := runtime.Caller(1)

	fn := runtime.FuncForPC(pc)
	pkg := fn.Name()
	pkg = strings.TrimPrefix(pkg, "github.com/bufbuild/protocompile/")
	pkg = pkg[:strings.Index(pkg, ".")]

	file = filepath.Base(file)

	// Ensure that we do not get partial writes, since there isn't really
	// any guarantee that Fprintf acquires a lock on os.Stderr.
	debugLogLock.Lock()
	defer debugLogLock.Unlock()

	_, _ = fmt.Fprintf(os.Stderr, "%s/%s [g%04d", pkg, file, goid.Get())
	if len(context) >= 1 {
		_, _ = fmt.Fprintf(os.Stderr, ", "+context[0].(string), context[1:]...)
	}
	_, _ = fmt.Fprintf(os.Stderr, "] %s: ", operation)
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
	_, _ = os.Stderr.Write([]byte{'\n'})
	_ = os.Stderr.Sync()
}
