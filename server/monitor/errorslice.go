package monitor

type ErrorSlice []error

func (errs ErrorSlice) Error() string {
	var str string
	for i, e := range errs {
		if str == "" {
			str += e.Error()
		} else {
			str += " | " + e.Error()
		}
	}

	return str
}
