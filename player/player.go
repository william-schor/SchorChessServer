package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"lanChessServer/chesslogic"
	"lanChessServer/globals"
	"lanChessServer/util"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
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
		m := util.Message{Type: "CREDENTIAL STATUS", Val: "INVALID"}
		b, err := json.Marshal(m)
		util.Check(err, "JSON Written")

		conn.Write(append(b, "\n"...))
		return -1
	}

	settings, rating := fetchSettings(credentials[0])

	m := util.Message{Type: "SETTINGS", Val: settings}
	b, err := json.Marshal(m)
	util.Check(err, "JSON Written")

	conn.Write(append(b, "\n"...))
	globals.SignOn(username, rating)

	return repl(conn, username, rating)

}

func auth(username, password string) bool {
	return strings.TrimSpace(username) == strings.TrimSpace(password)
}

func signOut(conn net.Conn, username string) {
	// sign out
	// TODO: remove from online list
	fmt.Println("Signing out")
	for i, p := range globals.Online {
		p.Lock()
		defer p.Unlock()
		if p.Username == username {
			globals.Online = append(globals.Online[:i], globals.Online[i+1:]...)
		}
	}
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
			signOut(conn, username)
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
		defer p.Unlock()

		fmt.Printf("%d : [%d, %d]", i, p.TimeControl[0], p.TimeControl[1])
		fmt.Printf("Local : [%d, %d]", timeControl[0], timeControl[1])

		if p.TimeControl == timeControl {
			if p.Rating > (rating-offset) && p.Rating < (rating+offset) {
				// criteria matches, start a new game
				// 1. copy comms and remove player from list
				gameComms := p.GameComms
				boardComms := p.BoardComms

				globals.Seeking = append(globals.Seeking[:i], globals.Seeking[i+1:]...)

				// 2. flip coin for color
				colors := []string{"white", "black"}

				rand.Seed(time.Now().UTC().UnixNano())
				myColor := rand.Intn(2)
				opponentsColor := 1 - myColor

				// tell opponent his color
				gameComms <- (colors[opponentsColor] + ":" + username + ":" + strconv.Itoa(rating))

				// initialize board
				board := chesslogic.Board()

				// tell opponent about the new board
				boardComms <- board

				// 3. play!
				play(colors[myColor], conn, gameComms, board, p.Rating, p.Username)
				played = true
			}
		}

		if played {
			// Player object should be garbage collected
			break
		}
	}

	// no match found. Add self to "seeking" list
	if !played {
		fmt.Println("ADDING SELF TO LIST")
		gameComms, boardComms := globals.Seek(username, rating, timeControl)
		// TODO: launch a goroutine to listen for a cancel

		// hang until someone plays you
		opponentData := <-gameComms
		board := <-boardComms

		fmt.Println("DONE HANGING")

		//parse the data they send
		splitData := strings.Split(opponentData, ":")
		myColor, oppoUsername, oppoRating := splitData[0], splitData[1], splitData[2]
		oppoRatingInt, _ := strconv.Atoi(oppoRating)

		// play with the data they send you
		play(myColor, conn, gameComms, board, oppoRatingInt, oppoUsername)
	}

}

func play(myColor string, conn net.Conn, gameComms chan string, board *chesslogic.SyncedBoard, opponentRating int, opponentName string) {
	// tell client about the new game
	gameData := myColor + ":" + strconv.Itoa(opponentRating) + ":" + opponentName
	util.SendMessage(conn, "GAME ANNOUNCEMENT", gameData)

	// get ready to listen, either from client or from other player thread
	buf := bufio.NewReader(conn)

	if myColor == "white" {
		var wStatus int
		var rStatus int
		for {
			wStatus = writeMove(buf, gameComms, conn, board, myColor)
			if wStatus == 0 {
				break
			}
			rStatus = readMove(buf, gameComms, conn)
			if rStatus == 0 {
				break
			}

		}

	} else {
		var wStatus int
		var rStatus int
		for {
			rStatus = readMove(buf, gameComms, conn)
			if rStatus == 0 {
				break
			}
			wStatus = writeMove(buf, gameComms, conn, board, myColor)
			if wStatus == 0 {
				break
			}
		}
	}
}

func writeMove(buf *bufio.Reader, gameComms chan string, conn net.Conn, board *chesslogic.SyncedBoard, myColor string) int {
	board.Lock()
	defer board.Unlock()
	for {
		data, err := buf.ReadString('\n')
		util.Check(err, "Read move from client")

		move := strings.TrimSpace(data)

		// make it my move
		chesslogic.MyMove(board, myColor)
		// check that move is legal
		legal, draw, resign, moveObj := chesslogic.IsLegalMove(move, board.Board)
		if legal {
			// make move on board
			board.Board.MakeMove(moveObj)

			checkmate, stalemate := chesslogic.GameIsOver(board.Board)
			if !(checkmate || stalemate) {
				// send move to opponent
				gameComms <- move
				return 1
			}

			if checkmate {
				// tell opponent they lost and end the game
				gameComms <- "GAMEOVER:LOST"
				// tell client they won
				util.SendMessage(conn, "GAME RESULT", "WON")
			}

			if stalemate {
				// tell opponent it is stalemate
				gameComms <- "GAMEOVER:STALEMATE"
				// tell client it is stalemate
				util.SendMessage(conn, "GAME RESULT", "STALEMATE")
			}

			return 0

		}

		if draw {
			gameComms <- "DRAWOFFER"
			// loop here. Do not return because draws do not make it not your turn
		} else if resign {
			// tell opponent they won
			gameComms <- "GAMEOVER:WON"
			// tell client they lost
			util.SendMessage(conn, "GAME RESULT", "LOST")
			return 0
		} else {
			util.SendMessage(conn, "MOVE STATUS", "ILLEGAL MOVE")
			// loop
		}

	}
}

func readMove(buf *bufio.Reader, gameComms chan string, conn net.Conn) int {
	// hang until opponent sends a move
	oppoMove := <-gameComms

	if strings.Contains(oppoMove, "GAMEOVER") {
		if strings.Contains(oppoMove, "WON") {
			// react to win
			util.SendMessage(conn, "GAME RESULT", "WON")
		} else {
			// react to loss
			util.SendMessage(conn, "GAME RESULT", "LOST")
		}

		return 0
	}

	util.SendMessage(conn, "MOVE", oppoMove)

	return 1

}
