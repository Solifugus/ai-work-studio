// Package core provides the fundamental data models and business logic for AI Work Studio.
//
// This package implements the core entities that drive the agent system:
//
//   - Goal: User objectives that the system serves, with hierarchical relationships
//   - Method: Proven approaches for achieving objectives (to be implemented)
//   - Objective: Specific tasks with minimal context references (to be implemented)
//
// The package follows the design principles of simplicity over complexity and
// judgment over rules. All entities support temporal evolution through the
// storage layer, allowing the system to learn and adapt over time.
//
// Key Design Principles:
//   - Store as storage.Node with specific types
//   - Use edges for relationships (e.g., goal hierarchy)
//   - Support temporal evolution for all entities
//   - Minimal context design - pass references, not full data
//   - Single user focus with personalization through learning
package core