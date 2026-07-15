// export_test.go re-exports unexported symbols that external test packages
// (package engine_test) need to reference. This file is compiled only during
// "go test" runs and has no effect on production builds.
package engine

// AnnotBoundaryOpen and AnnotBoundaryClose expose the internal delimiter
// constants so that engine_test assertions can reference them by name rather
// than duplicating the literal strings.
var (
	AnnotBoundaryOpen  = annotationBoundaryOpen
	AnnotBoundaryClose = annotationBoundaryClose
)
