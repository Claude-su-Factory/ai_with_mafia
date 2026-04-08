ALTER TABLE game_results DROP CONSTRAINT IF EXISTS game_results_room_id_fkey;
ALTER TABLE game_result_players DROP CONSTRAINT IF EXISTS game_result_players_game_result_id_fkey;
