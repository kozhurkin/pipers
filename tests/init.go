package tests

func init() {

}

var youtube = struct {
	GetViews func(string) (int, error)
}{
	GetViews: func(vid string) (int, error) {
		return 14.2e9, nil
	},
}
