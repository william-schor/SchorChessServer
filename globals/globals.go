package globals

import (
	"fmt"
	"sync"
)

// contains globally available "online" and "seeking" lists

// Player represents a player
type Player struct {
	Username    string
	Playing     bool
	Online      bool
	Rating      int
	TimeControl [2]int // [0]: total time in seconds, [1]: increment
	Comms       chan string
	sync.RWMutex
}

// Online lists all online Players
var Online []*Player

// Seeking lists all Players seeking a game
var Seeking []*Player

// SignOn allows a thread to add themself to the online list
func SignOn(username string, rating int) chan string {
	p := Player{}
	p.Username, p.Rating = username, rating
	p.Online, p.Playing = true, false

	Online = append(Online, &p)

	return p.Comms
}

// Seek allows a thread to add themself to the seeking list
func Seek(username string, rating int, timeControl [2]int) chan string {
	p := Player{}
	p.Username, p.Rating = username, rating
	p.Online, p.Playing = true, false
	p.Comms = make(chan string)
	p.TimeControl = timeControl
	Seeking = append(Seeking, &p)

	fmt.Println(len(Seeking))
	return p.Comms
}
