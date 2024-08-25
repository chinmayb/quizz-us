step by step implementation

1) game is started by the host by sending a message with GameID to the queue
2) processor listens to the topic
3) processor gets the gameID & uses the in memory game registry
4) 