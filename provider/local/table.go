package local

import "github.com/convox/rack/structs"

func (p *Provider) TableGet(app, name string) (*structs.Table, error) {
	return &structs.Table{}, nil
}
