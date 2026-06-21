package main

import "slices"

func diffSortedSlices(oldSlice, newSlice []string) (added, removed []string) {
	if slices.Equal(oldSlice, newSlice) {
		return nil, nil
	}

	for _, v := range newSlice {
		if _, found := slices.BinarySearch(oldSlice, v); !found {
			added = append(added, v)
		}
	}

	for _, v := range oldSlice {
		if _, found := slices.BinarySearch(newSlice, v); !found {
			removed = append(removed, v)
		}
	}

	return added, removed
}
