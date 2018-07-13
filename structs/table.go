package structs

type Table struct {
	Name string `json:"name"`

	Indexes TableIndexes `json:"indexes"`
}

type Tables []Table

type TableIndex struct {
	Name string `json:"name"`

	Key   string `json:"key"`
	Range string `json:"range"`
}

type TableIndexes []TableIndex
