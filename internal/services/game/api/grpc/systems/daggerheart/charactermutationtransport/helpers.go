package charactermutationtransport

func tierForLevel(level int) int {
	switch {
	case level <= 1:
		return 1
	case level <= 4:
		return 2
	case level <= 7:
		return 3
	default:
		return 4
	}
}
