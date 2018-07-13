package manifest

type Table struct {
	Name string `yaml:"-"`

	Indexes TableIndexes `yaml:"indexes"`
}

type Tables []Table

type TableIndex struct {
	Name string `yaml:"-"`

	Hash  string `yaml:"hash"`
	Range string `yaml:"range"`
}

type TableIndexes []TableIndex

func (t Table) GetName() string {
	return t.Name
}

func (t *Table) SetName(name string) error {
	t.Name = name
	return nil
}

func (ti TableIndex) GetName() string {
	return ti.Name
}

func (ti *TableIndex) SetName(name string) error {
	ti.Name = name
	return nil
}
