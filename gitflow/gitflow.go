package gitflow

func KindColor(kind BranchKind) string {
	switch kind {
	case KindFeature:
		return "39" // blue-ish
	case KindRelease:
		return "42" // green-ish
	case KindHotfix:
		return "196" // red
	case KindMain:
		return "220" // gold
	case KindDevelop:
		return "81" // cyan
	default:
		return "250"
	}
}
