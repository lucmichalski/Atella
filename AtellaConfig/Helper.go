package AtellaConfig

import "time"

func Pause(interval int64, interrupt *bool) {
	st := time.Now().Unix()
  for {
		if *interrupt {
			break
		}
		diff := time.Now().Unix() - st
		if diff >= interval {
			break
		}
	}
}
