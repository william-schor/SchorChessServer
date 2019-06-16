package player

import (
	"bufio"
	"fmt"
	"lanChessServer/globals"
	"lanChessServer/util"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

// Player starts a new thread for a player
func Player(conn net.Conn) int {
	buf := bufio.NewReader(conn)
	data, err := buf.ReadString('\n')
	util.Check(err, "credentials read")
	credentials := strings.Split(data, ".")
	username, pswd := credentials[0], credentials[1]

	fmt.Printf("username: %s  password: %s", username, pswd)

	if !auth(username, pswd) {
		conn.Write([]byte("credentials invalid\n"))
		return -1
	}

	settings, rating := fetchSettings(credentials[0])

	conn.Write([]byte(settings + "\n"))
	globals.SignOn(username, rating)

	return repl(conn, username, rating)

}

func auth(username, password string) bool {
	return strings.TrimSpace(username) == strings.TrimSpace(password)
}

func signOut(conn net.Conn) {
	// sign out
	// TODO: remove from online list
	fmt.Println("Signing out")
	shutdownWrite(conn)
	conn.Close()
}

func shutdownWrite(conn net.Conn) {
	if v, ok := conn.(interface{ CloseWrite() error }); ok {
		v.CloseWrite()
	}
}

func fetchSettings(username string) (string, int) {
	// pull settings from database
	return username + " : white : black : staunton", 1700
}

func repl(conn net.Conn, username string, rating int) int {
	buf := bufio.NewReader(conn)
	for {

		data, err := buf.ReadString('\n')
		util.Check(err, "Read Succeeded")

		data = strings.TrimSpace(data)

		fmt.Println(data)

		if data == "new game" {
			// all these numbers will come through data eventually
			newGame(conn, [2]int{300, 5}, username, rating, 100)

		} else if data == "sign out" {
			signOut(conn)
			break
		}

	}
	return 0
}

func talk(comms chan string) {
	// non game related communication happens here
}

func newGame(conn net.Conn, timeControl [2]int, username string, rating int, offset int) {
	played := false

	// look for match on list
	for i, p := range globals.Seeking {
		fmt.Println("entered loop")
		p.Lock()
		fmt.Printf("%d : [%d, %d]", i, p.TimeControl[0], p.TimeControl[1])
		fmt.Printf("Local : [%d, %d]", timeControl[0], timeControl[1])

		if p.TimeControl == timeControl {
			fmt.Println("First Check passed")
			if p.Rating > (rating-offset) && p.Rating < (rating+offset) {
				fmt.Println("Second Check passed")
				// criteria matches, start a new game
				// 1. copy comms and remove player from list
				gameComms := p.Comms
				globals.Seeking = append(globals.Seeking[:i], globals.Seeking[i+1:]...)

				// 2. flip coin for color
				colors := []string{"white", "black"}

				myColor := rand.Intn(2)
				opponentsColor := 1 - myColor

				// tell opponent his color
				gameComms <- (colors[opponentsColor] + ":" + username + ":" + strconv.Itoa(rating))

				// 3. play!
				play(colors[myColor], conn, gameComms, p.Rating, p.Username)
				played = true
			}
		}
		p.Unlock()

		if played {
			// Player object should be garbage collected
			break
		}
	}

	// no match found. Add self to "seeking" list
	if !played {
		fmt.Println("ADDING SELF TO LIST")
		gameComms := globals.Seek(username, rating, timeControl)
		// TODO: launch a goroutine to listen for a cancel

		// hang until someone plays you
		opponentData := <-gameComms

		fmt.Println("DONE HANGING")

		//parse the data they send
		splitData := strings.Split(opponentData, ":")
		myColor, oppoUsername, oppoRating := splitData[0], splitData[1], splitData[2]
		oppoRatingInt, _ := strconv.Atoi(oppoRating)

		// play with the data they send you
		play(myColor, conn, gameComms, oppoRatingInt, oppoUsername)
	}

}

func play(myColor string, conn net.Conn, comms chan string, opponentRating int, opponentName string) {
	// tell client about the new game
	conn.Write([]byte(myColor + ":" + strconv.Itoa(opponentRating) + ":" + opponentName + "\n"))

	// get ready to listen, either from client or from other player thread
	buf := bufio.NewReader(conn)

	if myColor == "white" {
		var status int
		for {
			status = writeMove(buf, comms, conn)
			if status == 0 {
				break
			}
			readMove(buf, comms, conn)
		}

	} else {
		var status int
		for {
			readMove(buf, comms, conn)

			status = writeMove(buf, comms, conn)
			if status == 0 {
				break
			}
		}
	}
}

func writeMove(buf *bufio.Reader, comms chan string, conn net.Conn) int {
	for {
		data, err := buf.ReadString('\n')
		util.Check(err, "Read move from client")
		data = strings.TrimSpace(data)

		legal, draw, resign := isLegalMove(data)
		if legal {
			// also check for draw offer and resignation
			if !gameIsOver(data) {
				// send move to client
				comms <- data
				return 1
			}

			// tell opponent they lost and end the game
			comms <- "GAMEOVER:LOST"
			return 0
		}

		if draw {
			comms <- "DRAWOFFER"
		} else if resign {
			comms <- "GAMEOVER:WON"
		} else {
			conn.Write([]byte("ILLEGAL MOVE\n"))
		}

	}
}

func readMove(buf *bufio.Reader, comms chan string, conn net.Conn) int {
	// hang until opponent sends a move
	oppoMove := <-comms

	if strings.Contains(oppoMove, "GAMEOVER") {
		if strings.Contains(oppoMove, "WON") {
			// react to win server side
		} else {
			// react to loss server side
		}
	}

	conn.Write([]byte(oppoMove + "\n"))

	return 1

}

func isLegalMove(move string) (bool, bool, bool) {
	return true, false, false
}

func gameIsOver(move string) bool {
	return false
}
