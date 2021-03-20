package types

// Agent represents a agent connection that is used for forwarding data between a user and a server.
type Agent struct {
	ID          string // unique
	Description string
}

// Service represents a intranet service exposed on a server.
type Service struct {
	ID          string // unique
	Addr        string
	Description string
	ExposedBy   string // Agent.ID
	ServerPort  string // which server port exposes this service
	//Enabled     bool   // access to users enabled?
}
