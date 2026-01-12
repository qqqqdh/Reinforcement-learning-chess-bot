import chess
import random
import pickle
import os

class ChessAI:
    def __init__(self, filename="chess_brain_v2.pkl"):
        self.filename = filename
        self.q_table = {}
        self.learning_rate = 0.5
        self.discount_factor = 0.9
        self.epsilon = 0.2  # 20% 확률로 탐색
        
        # 기물별 가치 설정
        self.piece_values = {
            chess.PAWN: 1,
            chess.KNIGHT: 3,
            chess.BISHOP: 3,
            chess.ROOK: 5,
            chess.QUEEN: 9,
            chess.KING: 0
        }
        self.load_brain()

    def get_state(self, board):
        return board.fen()

    def get_material_score(self, board):
        """현재 보드의 기물 상황을 점수화 (AI인 흑색 기준)"""
        score = 0
        for square in chess.SQUARES:
            piece = board.piece_at(square)
            if piece:
                val = self.piece_values[piece.piece_type]
                if piece.color == chess.BLACK: # AI 기물이면 플러스
                    score += val
                else: # 사람 기물이면 마이너스
                    score -= val
        return score

    def choose_move(self, board):
        state = self.get_state(board)
        legal_moves = list(board.legal_moves)
        
        if state not in self.q_table:
            self.q_table[state] = {move.uci(): 0.0 for move in legal_moves}

        if random.random() < self.epsilon:
            return random.choice(legal_moves)
        
        state_moves = self.q_table[state]
        best_move_uci = max(state_moves, key=state_moves.get)
        return chess.Move.from_uci(best_move_uci)

    def learn(self, history, final_reward):
        """게임 기록을 바탕으로 학습"""
        reward = final_reward
        for state, move_uci, step_reward in reversed(history):
            if state not in self.q_table:
                self.q_table[state] = {move_uci: 0.0}
            
            # 총 보상 = 기물 획득 보상 + 최종 승패 보상
            total_reward = step_reward + reward
            self.q_table[state][move_uci] += self.learning_rate * (total_reward - self.q_table[state][move_uci])
            reward *= self.discount_factor

    def save_brain(self):
        with open(self.filename, 'wb') as f:
            pickle.dump(self.q_table, f)
        print("\n[시스템] AI가 지식을 저장했습니다.")

    def load_brain(self):
        if os.path.exists(self.filename):
            try:
                with open(self.filename, 'rb') as f:
                    self.q_table = pickle.load(f)
                print(f"[시스템] 지능 로드 완료. (기억하는 상황: {len(self.q_table)}개)")
            except:
                self.q_table = {}

def play_game():
    ai = ChessAI()
    board = chess.Board()
    history = [] 

    print("--- 기물 보너스 학습 모드 시작 (AI는 흑색입니다) ---")
    
    while not board.is_game_over():
        print("\n", board)
        
        if board.turn == chess.WHITE:
            move_str = input("\n당신의 수 (예: e2e4): ")
            try:
                move = board.parse_san(move_str)
                board.push(move)
            except:
                print("유효하지 않은 수입니다.")
                continue
        else:
            state = ai.get_state(board)
            
            # 이동 전 기물 점수 계산
            old_score = ai.get_material_score(board)
            
            move = ai.choose_move(board)
            board.push(move)
            
            # 이동 후 기물 점수 계산 (보너스 산출)
            new_score = ai.get_material_score(board)
            step_reward = (new_score - old_score) * 0.1 # 기물 이득 시 보너스
            
            history.append((state, move.uci(), step_reward))
            print(f"AI의 수: {move} (기물 보상: {step_reward:.1f})")

    print("\n최종 결과:", board.result())
    
    # 최종 승패 보상
    final_reward = 0
    if board.result() == "0-1": final_reward = 1.0
    elif board.result() == "1-0": final_reward = -1.0
    
    ai.learn(history, final_reward)
    ai.save_brain()

if __name__ == "__main__":
    play_game()