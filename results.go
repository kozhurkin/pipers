package pipers

type Results[R any] []R

func (r *Results[R]) Shift() R {
	value := (*r)[0]
	*r = (*r)[1:len(*r)]
	return value
}
