package pkg

type SearcherStatus uint8

const (
	SearcherStatusNotRun SearcherStatus = iota
	SearcherStatusError
	SearcherStatusRunning
	SearcherStatusFinish
)
