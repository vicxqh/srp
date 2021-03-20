.PHONY : server all agent

server :
	@echo "building server..."
	go build -o ./bin/server ./server

agent :
	@echo "building agent..."
	go build -o ./bin/agent ./agent

server-lin64 :
	@echo "building server..."
	GOOS=linux GOARCH=amd64 go build -o ./bin/lin64/server ./server

agent-lin64 :
	@echo "building agent..."
	GOOS=linux GOARCH=amd64 go build -o ./bin/lin64/agent ./agent

all : server agent
all-lin64 : server-lin64 agent-lin64