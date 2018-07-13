package structs

type Table struct {
	Name string `json:"name"`

	Indexes TableIndexes `json:"indexes"`
}

type Tables []Table

type TableIndex struct {
	Name string `json:"name"`

	Hash  string `json:"hash"`
	Range string `json:"range"`
}

type TableIndexes []TableIndex
