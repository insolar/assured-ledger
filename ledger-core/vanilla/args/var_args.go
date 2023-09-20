package args

func AppendIntfArgs(args []interface{}, moreArgs ...interface{}) []interface{} {
	switch {
	case len(moreArgs) == 0:
		return args
	case len(args) == 0:
		return moreArgs
	default:
		return append(append(make([]interface{}, 0, len(args) + len(moreArgs)), args...), moreArgs...)
	}
}
