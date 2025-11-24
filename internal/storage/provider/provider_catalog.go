package provider

import (
	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/storage/engine"
)

// SetCatalog injects the catalog into the provider after initialization.
// This allows HeapEngine to access table schema for PK index optimization.
func (p *Provider) SetCatalog(catalog *schema.Catalog) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.catalog = catalog

	// Clear cached engines so they'll be recreated with schema info
	p.tableEngines = make(map[string]engine.Engine)
}
