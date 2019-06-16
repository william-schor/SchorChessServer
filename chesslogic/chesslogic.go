package chesslogic

import (
	"fmt"
	"lanChessServer/util"
	"strconv"
	"sync"

	chess "github.com/malbrecht/chess"
)

// SyncedBoard wraps around a board and a mutex
type SyncedBoard struct {
	Board *chess.Board
	sync.RWMutex
}

// MyMove modifies a board struct and sets it to the correct person's move
func MyMove(sb *SyncedBoard, color string) {
	fmt.Println(color)
	m := make(map[string]int)
	m["white"] = 0
	m["black"] = 1
	sb.Board.SideToMove = m[color]
}

// Board initializes a board in a starting position
func Board() *SyncedBoard {
	board, err := chess.ParseFen("")
	util.Check(err, "Board initialized")

	var sBoard SyncedBoard

	sBoard.Board = board
	return &sBoard
}

/*
IsLegalMove takes a move and a current board and returns
three booleans: (move is legal) (move is a draw offer) (move is a resignation) (Move m)
*/
func IsLegalMove(move string, board *chess.Board) (bool, bool, bool, chess.Move) {
	// return: legal, draw, resign
	if move == "RESIGN" {
		return false, false, true, chess.Move{}
	}

	if move == "DRAW" {
		return false, true, false, chess.Move{}
	}
	fmt.Println("Side: " + strconv.Itoa(board.SideToMove))

	fmt.Println("move: " + move)
	moveObj, err := board.ParseMove(move)
	fmt.Println(moveObj)
	fmt.Println(err)
	if (err != nil || moveObj == chess.Move{}) {
		fmt.Println("here")
		return false, false, false, chess.Move{}
	}

	return true, false, false, moveObj
}

// GameIsOver takes a board a indicates whether or not the game is over
// by checkmate or stalemate
func GameIsOver(board *chess.Board) (checkmate bool, stalemate bool) {
	check, mate := board.IsCheckOrMate()

	checkmate = mate && check
	stalemate = mate && !check

	return checkmate, stalemate

}
