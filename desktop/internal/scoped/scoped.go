// Package scoped wraps the lists the front end receives as events with the project they belong to,
// so a late event of a project the user has left can be told from the current one.
package scoped

// List is the payload of the sessions, notes, issues and cleanups events.
type List[T any] struct {
	Project string `json:"project"` // the project key
	Items   []T    `json:"items"`
}

// Of wraps items for the project. A nil list becomes an empty one: the front end never sees null.
func Of[T any](project string, items []T) List[T] {
	if items == nil {
		items = []T{}
	}
	return List[T]{Project: project, Items: items}
}
