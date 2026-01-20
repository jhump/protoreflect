## bufbuild/protocompile

This is a fork of [`bufbuild/protocompile`](https://github.com/bufbuild/protocompile) as of SHA [82d65](https://github.com/bufbuild/protocompile/commits/82d654092976869da1da9a4a944ecfd52feda7c2),
from December 15th, 2025 (over a year after v0.14.1, though most of the work in between time was experimental).

The upstream `protocompile` project has a new experimental compile and API. It will soon be promoted to non-experimental
and the old compiler will be deleted. To avoid that change being a breaking change for all `jhump/protoreflect` users,
this change seeks to insulate users of `jhump/protoreflect`. We fork the version of `protocompile` into an internal
package so it remains stable.

Most of the changes in this fork are to remove the experimental code that is not needed by `jhump/protoreflect`.

This version does include a parser and grammar that supports Edition 2024. However, the actual compiler does not yet
support Edition 2024 since most of the rules are not enforced and the semantics not fully implemented. The
`github.com/jhump/protoreflect/desc/protoparse` provides the ability to enable this not-fully-implemented support
for Edition 2024 via the `AllowExperimentalEditions` flag.
