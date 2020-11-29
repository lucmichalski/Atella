package AtellaServer

import (
	"../AtellaConfig"
)

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

	s.configuration.MasterVectorMutex.Lock()
	s.configuration.MasterVector = make(map[string][]AtellaConfig.VectorType, 0)
	s.configuration.MasterVectorMutex.Unlock()
	for !interrupt {
		AtellaConfig.Pause(s.configuration.Agent.Interval, &interrupt)
	}
	s.CloseReplyMaster = true
}
