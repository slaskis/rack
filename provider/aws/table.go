package aws

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (p *AWSProvider) TableGet(app, name string) (*structs.Table, error) {
	return nil, fmt.Errorf("unimplemented")
}
