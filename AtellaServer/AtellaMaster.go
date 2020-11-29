package AtellaServer

import (
	"fmt"
	"time"

	"../AtellaConfig"
)

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
			for index, i := range el {
				timeLimit := i.Interval * 5
				if i.Timestamp+timeLimit < time.Now().Unix() {
					s.configuration.Logger.LogInfo(fmt.Sprintf("Removing element %d for %s\n", index, v))
					s.configuration.MasterVectorDelDeprecatedElement(v, index, timeLimit)
				}
			}
		}
		AtellaConfig.Pause(60, interrupt)
	}
}
