package clipboard

// Command types sent from Go to PowerShell
const (
	CmdCheck  = "CHECK"
	CmdUpdate = "UPDATE"
	CmdNotify = "NOTIFY"
	CmdExit   = "EXIT"
)

// Response types sent from PowerShell to Go
const (
	RspReady = "READY"
	RspNone  = "NONE"
	RspImage = "IMAGE"
	RspEnd   = "END"
	RspOK    = "OK"
	RspErr   = "ERR"
)
