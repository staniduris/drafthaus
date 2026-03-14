package graph

// ListOpts controls pagination and filtering for entity list queries.
type ListOpts struct {
	Status  string
	Limit   int
	Offset  int
	OrderBy string
	Order   string // "asc" or "desc"
}
