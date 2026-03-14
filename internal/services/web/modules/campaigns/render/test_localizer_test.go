package render

import (
	"fmt"

	"golang.org/x/text/message"
)

type testLocalizer map[string]string

func (l testLocalizer) Sprintf(key message.Reference, args ...any) string {
	keyString := fmt.Sprint(key)
	format, ok := l[keyString]
	if !ok {
		format = keyString
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}
