package AtellaConfig

import "time"

func Pause(interval int64, interrupt *bool) {
	c := int64(0)
	one := time.Duration(1) * time.Second
  for {
		if *interrupt {
			break
		}
		if c >= interval {
			break
		}
		c = c + 1
		time.Sleep(one)
	}
}
