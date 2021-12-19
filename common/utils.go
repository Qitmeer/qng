/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package common

func ReverseBytes(bs *[]byte) {
	length := len(*bs)
	for i := 0; i < length/2; i++ {
		index:=length-1-i
		temp := (*bs)[index]
		(*bs)[index] = (*bs)[i]
		(*bs)[i] = temp
	}
}