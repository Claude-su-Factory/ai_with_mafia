ALTER TABLE game_results ADD CONSTRAINT game_results_room_id_fkey
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE;

ALTER TABLE game_result_players ADD CONSTRAINT game_result_players_game_result_id_fkey
    FOREIGN KEY (game_result_id) REFERENCES game_results(id) ON DELETE CASCADE;
