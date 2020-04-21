// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package retries

func Segregate(list []RetryID, isDoneFn func(RetryID) RetryState) (keepCount, removeStart int) {
	nonRemoveCount := len(list)
	removeStart = nonRemoveCount
	keepCount = 0

outer:
	for i := 0; i < nonRemoveCount; i++ {
		switch isDoneFn(list[i]) {
		case StopRetrying:
			continue
		case KeepRetrying:
			if keepCount != i {
				list[keepCount] = list[i]
			}
			keepCount++
			continue
		}

		for nonRemoveCount > i+1 {
			nonRemoveCount--
			switch isDoneFn(list[nonRemoveCount]) {
			case StopRetrying:
				continue
			case KeepRetrying:
				removeStart--
				list[keepCount], list[removeStart], list[removeStart] = list[nonRemoveCount], list[keepCount], list[i]
				keepCount++
				continue outer
			default:
				removeStart--
				if nonRemoveCount != removeStart {
					list[removeStart] = list[nonRemoveCount]
				}
			}
		}
		removeStart--
		list[removeStart] = list[i]
		break
	}
	return keepCount, removeStart
}
