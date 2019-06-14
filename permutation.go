package packong

import "github.com/innermond/pak"

func permutations(arr []*pak.Box) [][]*pak.Box {
	var helper func([]*pak.Box, int)
	res := [][]*pak.Box{}

	helper = func(arr []*pak.Box, n int) {
		if n == 1 {
			tmp := make([]*pak.Box, len(arr))
			copy(tmp, arr)
			res = append(res, tmp)
		} else {
			for i := 0; i < n; i++ {
				helper(arr, n-1)
				if n%2 == 1 {
					tmp := arr[i]
					arr[i] = arr[n-1]
					arr[n-1] = tmp
				} else {
					tmp := arr[0]
					arr[0] = arr[n-1]
					arr[n-1] = tmp
				}
			}
		}
	}
	helper(arr, len(arr))
	return res
}
