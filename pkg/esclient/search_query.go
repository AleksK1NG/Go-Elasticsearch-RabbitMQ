package esclient

//type SearchQuery struct {
//	Query `json:"query"`
//}
//
//type MultiMatch struct {
//	Query  string   `json:"query"`
//	Fields []string `json:"fields"`
//}
//
//type Must []struct {
//	MultiMatch `json:"multi_match"`
//}
//
//type Bool struct {
//	Must `json:"must"`
//}
//
//type Query struct {
//	Bool `json:"bool"`
//}

type MultiMatchQuery struct {
	Query Query `json:"query"`
}

type Query struct {
	Bool Bool `json:"bool"`
}

type Bool struct {
	Must []any `json:"must"`
}

type MultiMatch struct {
	Query  string   `json:"query"`
	Fields []string `json:"fields"`
}

type EsHits[T any] struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source T `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}
