package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/notnil/chess"
)

type ChessAI struct {
	QTable      map[string]map[string]float64 `json:"q_table"`
	GameCount   int                           `json:"game_count"`
	MoveHistory []string                      `json:"-"`
	mu          sync.RWMutex
}

var ai = &ChessAI{QTable: make(map[string]map[string]float64)}

const qFile = "qtable.json"

func init() {
	file, err := os.ReadFile(qFile)
	if err == nil {
		json.Unmarshal(file, &ai)
	}
}

// [핵심] 기물별 가치를 정의합니다.
func getPieceValue(p chess.Piece) float64 {
	values := map[chess.PieceType]float64{
		chess.Pawn:   10.0,
		chess.Knight: 30.0,
		chess.Bishop: 30.0,
		chess.Rook:   50.0,
		chess.Queen:  90.0,
		chess.King:   900.0, // 왕은 절대적 가치
	}
	val := values[p.Type()]
	return val
}

// 보드 상태의 점수를 계산합니다.
func evaluateBoard(pos *chess.Position) float64 {
	score := 0.0
	board := pos.Board()
	for i := 0; i < 64; i++ {
		p := board.Piece(chess.Square(i))
		if p != chess.NoPiece {
			val := getPieceValue(p)
			if p.Color() == chess.Black { // AI 색상
				score += val
			} else {
				score -= val
			}
		}
	}
	return score
}

func saveToFile() error {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	data, _ := json.MarshalIndent(ai, "", "  ")
	return os.WriteFile(qFile, data, 0644)
}

func moveHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FEN    string `json:"fen"`
		Result string `json:"result"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return
	}

	// 게임 종료 처리
	if req.Result != "" {
		ai.mu.Lock()
		ai.GameCount++
		reward := -500.0 // 패배 시 기본 감점 강화
		if req.Result == "Black" {
			reward = 500.0
		}

		for _, record := range ai.MoveHistory {
			parts := strings.Split(record, "|")
			if len(parts) == 2 {
				state, move := parts[0], parts[1]
				if ai.QTable[state] == nil {
					ai.QTable[state] = make(map[string]float64)
				}
				ai.QTable[state][move] += reward
			}
		}
		ai.MoveHistory = []string{}
		ai.mu.Unlock()
		saveToFile()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
		return
	}

	fen, _ := chess.FEN(req.FEN)
	game := chess.NewGame(fen)
	moves := game.ValidMoves()
	if len(moves) == 0 {
		return
	}

	state := req.FEN
	ai.mu.Lock()
	if ai.QTable[state] == nil {
		ai.QTable[state] = make(map[string]float64)
	}

	// [학습 로직] QTable 점수 + 현재 보드의 기물 가치 점수를 합산하여 최선의 수 선택
	sort.Slice(moves, func(i, j int) bool {
		m1, m2 := moves[i], moves[j]

		// 각 수 이후의 보드 상태 점수 계산
		g1, g2 := game.Clone(), game.Clone()
		g1.Move(m1)
		g2.Move(m2)

		s1 := ai.QTable[state][m1.String()] + evaluateBoard(g1.Position())
		s2 := ai.QTable[state][m2.String()] + evaluateBoard(g2.Position())

		return s1 > s2
	})

	selected := moves[0]
	ai.MoveHistory = append(ai.MoveHistory, state+"|"+selected.String())
	ai.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"move":       selected.String(),
		"game_count": ai.GameCount,
		"brain_size": len(ai.QTable),
	})
}

func main() {
	staticPath, _ := filepath.Abs("./static")
	http.Handle("/", http.FileServer(http.Dir(staticPath)))
	http.HandleFunc("/move", moveHandler)
	http.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		saveToFile()
		w.Write([]byte("OK"))
	})
	fmt.Println("서버 시작: http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
