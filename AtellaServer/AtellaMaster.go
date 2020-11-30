package AtellaServer

import (
	"fmt"
	"time"

	"../AtellaConfig"
)

type StateVector struct {
	vector []state
}

type state struct {
	Host  string
	State bool
}

func (s *AtellaServer) MasterVectorToStateVector() error {
	return nil
}

// Function impement master server logic
func (s *AtellaServer) MasterServer() {
	if s.configuration.Agent.Master {
		s.configuration.Logger.LogSystem("[Server] I'm master server")
	} else {
		s.configuration.Logger.LogSystem("[Server] I'm not a master server")
		s.CloseReplyMaster = true
		return
	}

	var interrupt bool = false
	go func() {
		<-s.stopRequest
		s.configuration.Logger.LogSystem("[Server] Stopping master server")
		interrupt = true
	}()

	s.configuration.MasterVectorInit()
	go s.masterServerDropDeprecatedElements(&interrupt)

	for !interrupt {
		AtellaConfig.Pause(s.configuration.Agent.Interval, &interrupt)
	}
	s.CloseReplyMaster = true
}

func (s *AtellaServer) masterServerDropDeprecatedElements(interrupt *bool) {
	for !*interrupt {
		vector := s.configuration.GetMasterVector()
		for v, el := range vector {
			l := len(el)
			r := 0
			if l > 0 {
				for index, i := range el {
					timeLimit := i.Interval * 5
					if i.Timestamp+timeLimit < time.Now().Unix() {
						s.configuration.Logger.LogInfo(fmt.Sprintf("[Server] Removing element %d for %s\n", index, v))
						r = r + 1
						s.configuration.MasterVectorDelDeprecatedElement(v, index, timeLimit)
					}
				}
				if r >= l {
					s.configuration.Logger.LogInfo(fmt.Sprintf("[Server] Removing map element %s, %d %d\n", v, l, r))
					s.configuration.MasterVectorDelDeprecatedElement(v, -1, -1)
				}
			}
		}
		AtellaConfig.Pause(60, interrupt)
	}
}
