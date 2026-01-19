// Package storage provides temporal data structures for AI Work Studio.
//
// This package implements the core temporal storage abstractions used throughout
// the AI Work Studio system. It provides Node and Edge structures that maintain
// full temporal history through versioning, where every modification creates a
// new version and nothing is ever deleted.
//
// The temporal model supports queries like "get version at timestamp" and is
// designed to be migration-ready for AmorphDB while currently using JSON file
// storage for simplicity.
//
// Key concepts:
//   - Node: Represents an entity (goal, method, objective) with temporal metadata
//   - Edge: Represents a relationship between nodes with temporal metadata
//   - Temporal versioning: ValidFrom/ValidUntil timestamps track version lifecycles
//   - Version immutability: Once created, versions are never modified
//   - Current version: ValidUntil == zero time indicates the active version
package storage