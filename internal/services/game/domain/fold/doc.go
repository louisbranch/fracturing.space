// Package fold defines the canonical Folder interface and registration-based
// fold routing for core domain folds.
//
// The Folder interface is the shared contract used by the engine write path,
// the replay pipeline, and system modules. Engine and replay define narrower
// local interfaces when they only need the Fold method.
//
// CoreFoldRouter eliminates the sync-drift risk between a domain's Fold
// switch and its FoldHandledTypes list by deriving the type list from
// registered handler functions.
package fold
