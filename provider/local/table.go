package local

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/convox/rack/helpers"
	"github.com/convox/rack/structs"
)

func (p *Provider) TableGet(app, name string) (*structs.Table, error) {
	m, _, err := helpers.AppManifest(p, app)
	if err != nil {
		return nil, err
	}

	for _, t := range m.Tables {
		if t.Name == name {
			var st *structs.Table

			if err := recode(t, &st); err != nil {
				return nil, err
			}

			return st, nil
		}
	}

	return nil, fmt.Errorf("no such table: %s", name)
}

func recode(from, to interface{}) error {
	b := bytes.Buffer{}

	if err := gob.NewEncoder(&b).Encode(from); err != nil {
		return err
	}

	if err := gob.NewDecoder(&b).Decode(to); err != nil {
		return err
	}

	return nil
}
