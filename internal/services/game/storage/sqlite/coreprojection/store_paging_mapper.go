package coreprojection

// mapPageRows applies a row-to-domain mapper to at most pageSize rows and returns
// the next-page token when additional rows are available.
//
// Callers are expected to query using pageSize+1 rows and validate pageSize before
// calling this helper.
func mapPageRows[Row any, Item any](
	rows []Row,
	pageSize int,
	rowID func(Row) string,
	mapRow func(Row) (Item, error),
) ([]Item, string, error) {
	capHint := pageSize
	if capHint > len(rows) {
		capHint = len(rows)
	}
	items := make([]Item, 0, capHint)

	for i, row := range rows {
		if i >= pageSize {
			return items, rowID(rows[pageSize-1]), nil
		}
		item, err := mapRow(row)
		if err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}

	return items, "", nil
}
