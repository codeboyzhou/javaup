package project

import "fmt"

// Inspector reads the saved environment for the project containing a directory.
type Inspector struct {
	store configFinder
}

// NewDefaultInspector creates an inspector backed by the user project configuration store.
func NewDefaultInspector() (*Inspector, error) {
	store, err := NewDefaultConfigStore()
	if err != nil {
		return nil, err
	}
	return NewInspector(store), nil
}

// NewInspector creates a project inspector from a replaceable configuration store.
func NewInspector(store configFinder) *Inspector {
	return &Inspector{store: store}
}

// Inspect returns the configuration for the initialized project containing root.
func (i *Inspector) Inspect(root string) (Config, error) {
	config, _, found, err := i.store.Find(root)
	if err != nil {
		return Config{}, err
	}
	if !found {
		return Config{}, fmt.Errorf("no initialized javaup project found from %s; run jup init", root)
	}
	return config, nil
}
